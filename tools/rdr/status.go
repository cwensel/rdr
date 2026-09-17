package main

// `recs status` — the navigator's read, in one call.
//
// `skills/rdr-status/SKILL.md` answers "I lost my place, what do I run
// next?" and used to pay for that answer in choreography: an `inspect
// --json --filter …`, a `--select §decision-rationale` byte read, an `ls`
// per lens folder, a `status.md` read, and then a model re-deriving a
// prose signal table over the results. facts.go turned the signals into
// data (`models/rdr-facts.toml`); this file is the verb that runs them,
// so the whole read is one invocation — and an invocation is a TURN,
// which re-sends the conversation.
//
// Four renderings of one evaluation, because four different consumers
// need it: text for a person, `--json` for anything structured, `--tags`
// for a resolver's argv, and `--flat` for a declared command reader's
// flat object of strings (intrastate RDR 0025) — the shape an accessor
// reads a record's own state back through after a write. The evaluation
// itself is identical in all four — a rendering that could disagree with
// another rendering of the same facts would be a second source of truth.
//
// It never writes. Neither the records, nor evidence, nor a resolver's
// owned state: that file format belongs to the other side of the seam,
// and a read-only navigator that writes is no longer derivable-from-disk.

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// statusCmd is `recs status [NNNN…]`: one record's fact vector, a NAMED
// SET of records', or the in-flight worklist with each record's facts.
//
// The three arities are three different questions and two different
// costs. One record and a named set resolve each argument by name and
// read only those files; the worklist scans the corpus. That gap is
// measured and it is large — 47ms against 2.0s on the reference corpus —
// so a caller who knows which records it means must never pay the scan
// to ask about them. That is exactly what the set form is for: Stage 8's
// predecessor precheck and 7.1's Final-and-unimplemented filter each
// hold a list of records already, and both used to read N status.md
// headers by hand rather than ask.
func statusCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	explicit := ""
	if f.facts != nil {
		explicit = *f.facts
	}
	path, err := factTablePath(explicit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	tbl, err := LoadFactTable(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	// `--checklist` reads the WHOLE vector three-valued, so a filter (which
	// would turn every unasked fact into an absence) and the two argv
	// renderings are refused with it; `--argv` is a line-per-record form
	// and does not compose with the one-record renderings either.
	filtered := f.filter != nil && *f.filter != ""
	flat := f.flat != nil && *f.flat
	switch {
	case *f.checklist && (*f.tags || *f.argv || flat || filtered):
		fmt.Fprintln(stderr, "stopped:usage (--checklist renders the whole vector, as text or beside the facts under --json; not with --tags, --argv, --flat or --filter)")
		return 2
	case *f.argv && (*f.tags || *f.json || flat):
		fmt.Fprintln(stderr, "stopped:usage (--argv is one line per record; not with --tags, --json or --flat)")
		return 2
	case flat && (*f.tags || *f.json):
		// Three renderings of one vector; naming two says which is meant
		// nowhere. `--flat` is a reader's wire shape and `--json` is the
		// tool's own answer — silently preferring either would put the
		// wrong object on an accessor's stdin.
		fmt.Fprintln(stderr, "stopped:usage (--flat is the command-reader object; not with --tags or --json)")
		return 2
	}
	switch {
	case len(args) == 0:
		return statusWorklist(tbl, f, stdout, stderr)
	case len(args) == 1:
		return statusOne(tbl, args[0], f, stdout, stderr)
	}
	return statusSet(tbl, args, f, stdout, stderr)
}

// peers is one status call's answer to FactEnv.Peer and FactEnv.Corpus:
// a peer record's env by number, read once and cached, and the corpus on
// first need. Bound from the same records dir and table as the record in
// hand, so a rollup and a direct `status` of the peer agree by
// construction. With the corpus already in hand (the worklist) no file
// is re-read; otherwise a peer is resolved by name and the corpus is
// scanned only if a fact asks for it.
type peers struct {
	tbl      *FactTable
	f        *flags
	byRecord map[string]*scan.Document
	cache    map[string]*FactEnv
	docs     []*scan.Document
	once     sync.Once
}

func newPeers(tbl *FactTable, f *flags, docs []*scan.Document) *peers {
	p := &peers{tbl: tbl, f: f, cache: map[string]*FactEnv{}, docs: docs}
	if docs != nil {
		p.byRecord = map[string]*scan.Document{}
		for _, d := range docs {
			p.byRecord[d.Record] = d
		}
	}
	return p
}

func (p *peers) corpus() []*scan.Document {
	p.once.Do(func() {
		if p.docs == nil {
			p.docs, _, _ = records(p.f, io.Discard)
		}
	})
	return p.docs
}

func (p *peers) peer(num string) (*FactEnv, bool) {
	if env, ok := p.cache[num]; ok {
		return env, env != nil
	}
	var doc *scan.Document
	if p.byRecord != nil {
		doc = p.byRecord[num]
	} else if path, err := resolve(num, *p.f.records); err == nil {
		doc, _ = scan.File(path, scan.Options{Project: *p.f.project})
	}
	if doc == nil || doc.Record == "" {
		p.cache[num] = nil
		return nil, false
	}
	env := NewFactEnv(p.tbl, doc, recordSlug(doc.Path))
	p.cache[num] = env
	return env, true
}

func (p *peers) bind(env *FactEnv) {
	env.Peer, env.Corpus = p.peer, p.corpus
}

// statusOne evaluates one record.
func statusOne(tbl *FactTable, arg string, f *flags, stdout, stderr io.Writer) int {
	path, err := resolve(arg, *f.records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	doc, err := scan.File(path, scan.Options{Project: *f.project})
	if err != nil {
		fmt.Fprintf(stderr, "stopped:unreadable (%v)\n", err)
		return 2
	}
	if doc.Record == "" {
		fmt.Fprintln(stderr, "stopped:no-record-number (neither the title nor the filename carries NNNN)")
		return 2
	}
	env := NewFactEnv(tbl, doc, recordSlug(path))
	env.Want = wantedFacts(tbl, f)
	env.ResolveEdges = func() { resolveEdges(doc, f, stderr) }
	newPeers(tbl, f, nil).bind(env)
	facts := tbl.Evaluate(env)
	if facts, err = filterFacts(tbl, facts, f); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	switch {
	case *f.flat:
		return emitFlat(tbl, facts, wantedFacts(tbl, f), stdout, stderr)
	case *f.tags:
		return emitTags(tbl, facts, wantedFacts(tbl, f), stdout, stderr)
	case *f.argv:
		return emitArgv(tbl, []statusRow{{Summary: scan.Summarize(doc), Facts: facts}}, wantedFacts(tbl, f), stdout, stderr)
	case *f.checklist:
		// The env's resolved roots travel with the facts: a root missing
		// from them is unbound, which is what turns an absence into `?`.
		rows := renderChecklist(tbl, facts, env.Roots)
		if *f.json {
			return emit(map[string]any{
				"schema": schemaVersion, "record": doc.Record, "facts": facts, "checklist": rows,
			}, stdout, stderr)
		}
		return emitChecklist(rows, stdout)
	case *f.json:
		return emitFacts(facts, doc.Record, stdout, stderr)
	}
	return emitFactLines(facts, stdout)
}

// statusSet evaluates a NAMED set of records — the question both Stage 8's
// predecessor precheck and Stage 7.1's completeness filter actually ask.
//
// Each argument is resolved by name, so this reads exactly the files it
// was given and never scans the records dir. It is the same evaluation
// `statusOne` runs, once per record, which is what lets the two forms
// agree by construction rather than by care.
//
// AN UNRESOLVABLE ARGUMENT IS A `skipped` ROW, NOT A HALT. That is the
// three-valued discipline this table is built on, carried up to the set:
// a predecessor whose record or artifact dir is missing must read as
// "nothing looked", and the caller has to be able to tell that from
// "looked, not COMPLETE" — Stage 8 halts on one and not the other. A
// refusal here would collapse the two, since a set that cannot be
// answered at all says nothing about any of its members. The skipped row
// carries the reason, so absence is reported rather than inferred.
func statusSet(tbl *FactTable, args []string, f *flags, stdout, stderr io.Writer) int {
	// `--tags` renders ONE resolver's argv; the worklist refuses it for
	// the same reason and with the same words.
	if *f.tags {
		fmt.Fprintln(stderr, "stopped:usage (--tags renders one record's argv; name a record)")
		return 2
	}
	if f.flat != nil && *f.flat {
		fmt.Fprintln(stderr, "stopped:usage (--flat renders one record's object; name a record)")
		return 2
	}
	if *f.checklist {
		fmt.Fprintln(stderr, "stopped:usage (--checklist renders one record's checklist; name a record)")
		return 2
	}
	rows := []statusRow{}
	skipped := []map[string]string{}
	peers := newPeers(tbl, f, nil)
	for _, arg := range args {
		path, err := resolve(arg, *f.records)
		if err != nil {
			skipped = append(skipped, map[string]string{"target": arg, "why": err.Error()})
			continue
		}
		doc, err := scan.File(path, scan.Options{Project: *f.project})
		if err != nil {
			skipped = append(skipped, map[string]string{"target": arg, "why": fmt.Sprintf("unreadable (%v)", err)})
			continue
		}
		if doc.Record == "" {
			skipped = append(skipped, map[string]string{"target": arg,
				"why": "no-record-number (neither the title nor the filename carries NNNN)"})
			continue
		}
		env := NewFactEnv(tbl, doc, recordSlug(path))
		env.Want = wantedFacts(tbl, f)
		env.ResolveEdges = func() { resolveEdges(doc, f, stderr) }
		peers.bind(env)
		facts := tbl.Evaluate(env)
		facts, err = filterFacts(tbl, facts, f)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		rows = append(rows, statusRow{Summary: scan.Summarize(doc), Facts: facts})
	}
	if *f.json {
		return emit(map[string]any{
			"schema": schemaVersion, "records": rowsFor(rows, f), "skipped": skipped,
		}, stdout, stderr)
	}
	if *f.argv {
		if code := emitArgv(tbl, rows, wantedFacts(tbl, f), stdout, stderr); code != 0 {
			return code
		}
	} else {
		for _, r := range rows {
			fmt.Fprintf(stdout, "%s %-8s %s\n",
				filepath.Base(strings.TrimSuffix(r.Path, ".md")), r.Status.Value, qualifier(r.Summary))
			emitFactLines(r.Facts, indentWriter{stdout})
		}
	}
	for _, s := range skipped {
		fmt.Fprintf(stdout, "%s skipped  %s\n", s["target"], s["why"])
	}
	return 0
}

// filterFacts keeps only the named facts, and is the whole reason a set
// call is affordable to READ.
//
// The vector is 48 facts and ~7KB of JSON per record, which a five-member
// cluster turns into ~27KB of a caller's context to answer one word per
// record. Every other projection verb already has `--filter` for exactly
// this; `status` was the one that did not, so its callers paid the full
// vector or went back to reading files by hand.
//
// A name the table does not declare is a REFUSAL, not an empty result,
// and the message lists what could have been asked for — the same rule
// `--filter` follows on `inspect` and `index`, because a filter that
// silently answers nothing is read as "the fact is absent", which is a
// claim about the record rather than about the request.
//
// Filtering is applied AFTER evaluation, never before: a fact's value can
// depend on the table being evaluated whole, and a filter is a question
// about the OUTPUT, not an instruction to look at less.
func filterFacts(tbl *FactTable, facts []Fact, f *flags) ([]Fact, error) {
	filtered := f.filter != nil && *f.filter != ""
	drop := exceptedFacts(f)
	if !filtered && len(drop) == 0 {
		return facts, nil
	}
	// An --except name must be declared, exactly as a --filter name must:
	// silently ignoring a typo would hand the caller a vector still
	// carrying the fact they meant to drop, and the refusal it then takes
	// names the wrong thing.
	if err := declaredFacts(tbl, drop); err != nil {
		return nil, err
	}
	keep := wantedFacts(tbl, f)
	if keep == nil {
		// --except with no --filter: keep everything the table declares,
		// minus the excepted names. Resolved here because this is where
		// the table is in hand; wantedFacts has only the flags.
		keep = map[string]bool{}
		for _, d := range tbl.Facts {
			if !drop[d.Name] {
				keep[d.Name] = true
			}
		}
	}
	// A --tags call still renders every declared sentinel, so the filter
	// narrows what is EVALUATED (FactEnv.Want) as well as what is kept.
	return filterFactsBy(tbl, facts, keep)
}

// wantedFacts is the --filter list as a set, or nil when there is none —
// nil meaning "everything", which is what FactEnv.Want reads it as.
//
// `--except` subtracts from whichever set that is. It exists because a
// consumer model may declare a fact OWNED (intrastate reads owned state
// from its declared accessors and refuses it as argv, `flow-tag-owned`),
// while the same vector feeds models that guard on it as observed. The
// caller therefore needs the whole vector MINUS a couple of names, and
// `--filter` cannot say that: naming the other seventy is a list that
// rots the moment the table grows a fact.
//
// Subtracting rather than intersecting is the point — a fact added to
// `rdr-facts.toml` tomorrow flows to the caller without an edit, which is
// what a keep-list gives up.
// It takes the TABLE because `--except` alone has no keep-list to
// subtract from: "everything" is nil, and nil cannot carry a subtraction.
// Every renderer must see the same set — the sentinel pass reads it too,
// and a fact dropped from evaluation but not from `want` comes back as
// its declared absent sentinel, which is the whole thing being avoided.
func wantedFacts(tbl *FactTable, f *flags) map[string]bool {
	var keep map[string]bool
	if f.filter != nil && *f.filter != "" {
		keep = map[string]bool{}
		for _, name := range strings.Split(*f.filter, ",") {
			if name = strings.TrimSpace(name); name != "" {
				keep[name] = true
			}
		}
	}
	drop := exceptedFacts(f)
	if len(drop) == 0 {
		return keep
	}
	if keep == nil {
		// Materialising "everything" turns `want` non-nil, which would
		// otherwise flip `skipSentinel` from "on-demand facts are out" to
		// "every named fact is in". Carry the on-demand rule across by
		// leaving those names out, exactly as the nil case does.
		keep = make(map[string]bool, len(tbl.Facts))
		for _, d := range tbl.Facts {
			if !d.OnDemand {
				keep[d.Name] = true
			}
		}
	}
	for name := range drop {
		delete(keep, name)
	}
	return keep
}

// exceptedFacts is the --except list as a set, nil when there is none.
func exceptedFacts(f *flags) map[string]bool {
	if f.except == nil || *f.except == "" {
		return nil
	}
	drop := map[string]bool{}
	for _, name := range strings.Split(*f.except, ",") {
		if name = strings.TrimSpace(name); name != "" {
			drop[name] = true
		}
	}
	return drop
}

// declaredFacts refuses a name the table does not declare. Shared by
// --filter and --except so a typo reads the same either way.
func declaredFacts(tbl *FactTable, names map[string]bool) error {
	declared := make(map[string]bool, len(tbl.Facts))
	for _, d := range tbl.Facts {
		declared[d.Name] = true
	}
	for name := range names {
		if !declared[name] {
			have := make([]string, 0, len(tbl.Facts))
			for _, d := range tbl.Facts {
				have = append(have, d.Name)
			}
			sort.Strings(have)
			return fmt.Errorf("stopped:no-such-fact (%s; have %s)", name, strings.Join(have, " "))
		}
	}
	return nil
}

func filterFactsBy(tbl *FactTable, facts []Fact, keep map[string]bool) ([]Fact, error) {
	if err := declaredFacts(tbl, keep); err != nil {
		return nil, err
	}
	out := make([]Fact, 0, len(keep))
	for _, fact := range facts {
		if keep[fact.Name] {
			out = append(out, fact)
		}
	}
	return out, nil
}

// statusWorklist is the no-arg form: the Draft/Final worklist, each row
// carrying its own facts.
//
// It absorbed `index --in-flight`, which answered the same question
// without the facts and left the caller to go and derive them per
// record — the choreography this verb exists to end. `index --status`
// stays where it is: grouping every record by status is a corpus
// question, not a navigator's.
func statusWorklist(tbl *FactTable, f *flags, stdout, stderr io.Writer) int {
	// A worklist is many records and `--tags` renders ONE resolver's
	// argv. Concatenating twenty records' tags would produce a command
	// line that names no record and resolves nothing — so this refuses
	// rather than emitting something a caller could paste.
	if *f.tags {
		fmt.Fprintln(stderr, "stopped:usage (--tags renders one record's argv; name a record)")
		return 2
	}
	if f.flat != nil && *f.flat {
		fmt.Fprintln(stderr, "stopped:usage (--flat renders one record's object; name a record)")
		return 2
	}
	if *f.checklist {
		fmt.Fprintln(stderr, "stopped:usage (--checklist renders one record's checklist; name a record)")
		return 2
	}
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}
	// The whole corpus is in hand, so the edge tallies resolve against it
	// in ONE primed pass, on the first row that asks — as `index` does —
	// rather than one scan per record.
	var resolveOnce sync.Once
	resolveAll := func() {
		resolveOnce.Do(func() { scan.NewResolver(docs, *f.repo).ResolveAll(docs) })
	}
	rows := []statusRow{}
	peers := newPeers(tbl, f, docs)
	for _, d := range docs {
		s := scan.Summarize(d)
		// `--argv` feeds a resolver that has a cell for a parked record
		// (locate-parked prints the revisit condition), so Deferred rows
		// travel with it: a trigger nobody re-reads is how a park becomes
		// an abandon. The text and JSON worklists stay the in-flight set.
		if !s.InFlight && !(*f.argv && s.Status != nil && s.Status.Value == "Deferred") {
			continue
		}
		env := NewFactEnv(tbl, d, recordSlug(s.Path))
		env.Want = wantedFacts(tbl, f)
		env.ResolveEdges = resolveAll
		peers.bind(env)
		facts, err := filterFacts(tbl, tbl.Evaluate(env), f)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		rows = append(rows, statusRow{Summary: s, Facts: facts})
	}
	if *f.json {
		return emit(map[string]any{
			"schema": schemaVersion, "records": rowsFor(rows, f), "skipped": skipped,
		}, stdout, stderr)
	}
	if *f.argv {
		return emitArgv(tbl, rows, wantedFacts(tbl, f), stdout, stderr)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%s %-8s %s\n",
			filepath.Base(strings.TrimSuffix(r.Path, ".md")), r.Status.Value, qualifier(r.Summary))
		emitFactLines(r.Facts, indentWriter{stdout})
	}
	fmt.Fprintf(stdout, "total %d in flight over %d records\n", len(rows), len(docs))
	return 0
}

// statusRow is one worklist row: the summary `index --in-flight`
// published, plus the facts that were the reason to open the record
// again.
type statusRow struct {
	scan.Summary
	Facts []Fact `json:"facts"`
}

// leanRow is what a row becomes under `--filter`: the identity keys, and
// the facts that were asked for.
//
// The summary is not free. `Profile` alone carries the field's whole
// rationale tail — a paragraph on most records — so a five-record set
// emits ~44KB unfiltered and still ~12KB with the facts filtered, when
// the facts themselves are about 1KB. Filtering the vector while shipping
// the summary anyway would answer the letter of the request and miss its
// point, since the reason to filter is a caller's context budget.
//
// Identity is kept for the same reason `filterKeys` keeps it: a row a
// caller cannot attribute to a record is not an answer. `record` and
// `path` are that minimum, and `path` stays because the flow's next act
// on a member is usually to name a file under it.
type leanRow struct {
	Record string `json:"record"`
	Path   string `json:"path"`
	Facts  []Fact `json:"facts"`
}

// rowsFor renders the rows at the width the request asked for.
func rowsFor(rows []statusRow, f *flags) any {
	if f.filter == nil || *f.filter == "" {
		return rows
	}
	lean := make([]leanRow, 0, len(rows))
	for _, r := range rows {
		lean = append(lean, leanRow{Record: r.Record, Path: r.Path, Facts: r.Facts})
	}
	return lean
}

// recordSlug is the filename slug a probe's paths hang under. The
// evidence tree is keyed by `NNNN-slug`, which is the file's own name —
// not the title, which authors rewrite.
func recordSlug(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".md")
}

// emitFactLines is the human form: one fact per line, name first so the
// output greps and diffs. A set prints its members in the same canonical
// array the tag rendering uses, so the two forms never disagree about
// what a set holds.
func emitFactLines(facts []Fact, stdout io.Writer) int {
	for _, f := range facts {
		fmt.Fprintf(stdout, "%-28s %-7s %s\n", f.Name, f.Kind, factValueText(f))
	}
	return 0
}

func factValueText(f Fact) string {
	if f.Kind == "set" {
		return jsonArray(f.Members)
	}
	return f.Value
}

// indentWriter prefixes each line, so a worklist row's facts read as
// belonging to the row above them.
type indentWriter struct{ w io.Writer }

func (i indentWriter) Write(p []byte) (int, error) {
	s := strings.TrimSuffix(string(p), "\n")
	for _, line := range strings.Split(s, "\n") {
		if _, err := fmt.Fprintf(i.w, "  %s\n", line); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// emitTags renders the facts as a resolver's argv: `--tag` and `k=v` on
// their own lines.
//
// The consuming call is `intrastate flow resolve --model … $(recs status
// --tags NNNN)`, and an UNQUOTED command substitution splits its output
// on IFS whitespace and then globs the words — it does not split on
// lines, and quotes inside the output are literal characters, not
// syntax. So a value carrying a space does not arrive as one argument:
// it arrives as several, and the FIRST of them is `k=<head>`, a
// perfectly well-formed tag with a silently truncated value. The
// resolver refuses the leftover words with `flow-tag-invalid` — usually.
// A tail that happens to parse would be accepted, and the caller would
// route on a value the record does not carry.
//
// Every routing fact is safe by construction: an enum, a bool, an int,
// and a set rendered as a COMPACT JSON array (`["0131","0132"]`, no
// space after the comma) are each one shell word. The exception is the
// one fact that carries prose — `profile_raw`, the Profile field's
// rationale tail, which exists for a caveat line and not for routing,
// and which on the reference corpus is a sentence on 91 records of 144.
//
// So a fact the TABLE declares `prose = true` is not rendered here at
// all. That is a declaration, not a discovery: which facts carry prose
// is a property of the fact, and deciding it from the value in hand
// would make `--tags` succeed on one record and fail on the next. The
// table draws the same line one level in — the Status QUALIFIER is prose
// and is deliberately not a fact, while its `form` is. Prose is read
// through `--json`, where it is a JSON string and nothing splits it.
//
// Omitting it is safe here, and only here, because the omission is
// declared rather than conditional. An omitted key is load-bearing on
// the other side — that kernel is three-valued, and absent leaves a rule
// undecided — but a prose fact is one no routing rule can match on
// anyway, so a resolver never had it to lose. What would NOT be safe is
// dropping a fact because this particular record's value looked
// awkward, which is why that case is a refusal rather than a skip.
// withAbsentSentinels fills in the declared sentinel for every fact that
// declares one and evaluated to nothing, returning the facts in the
// TABLE's declaration order so the argv stays stable and diffable.
//
// This is the one place absence is collapsed, and it is collapsed only
// for `--tags`. A resolver's argv cannot spell "absent" — a key is
// present or the guard atom over it refuses — so a fact a routing table
// discriminates on must always arrive. The sentinel says which value
// carries that meaning, and because the table declares it as a domain
// member the model claims the cell positively.
//
// A fact with no `absent` declaration is untouched: it is omitted when
// absent exactly as before, which is right for the facts nothing routes
// on. Adding a sentinel to one is a deliberate act, not a default.
func withAbsentSentinels(tbl *FactTable, facts []Fact, want map[string]bool) []Fact {
	return withSentinels(tbl, facts, want)
}

// skipSentinel says whether a declaration is outside this render: not in
// the filter, or on demand with no filter at all. An on-demand fact was
// never evaluated by an unfiltered call, so its sentinel would say
// "nothing looked" on every record and every routing table would carry a
// key nobody asked for; it renders only when the caller names it.
func skipSentinel(d FactDecl, want map[string]bool) bool {
	if want == nil {
		return d.OnDemand
	}
	return !want[d.Name]
}

func withSentinels(tbl *FactTable, facts []Fact, want map[string]bool) []Fact {
	have := make(map[string]bool, len(facts))
	for _, f := range facts {
		have[f.Name] = true
	}
	missing := false
	for _, d := range tbl.Facts {
		if skipSentinel(d, want) {
			continue
		}
		if d.HasAbsent && !have[d.Name] {
			missing = true
			break
		}
	}
	if !missing {
		return facts
	}
	byName := make(map[string]Fact, len(facts))
	for _, f := range facts {
		byName[f.Name] = f
	}
	out := make([]Fact, 0, len(facts)+1)
	for _, d := range tbl.Facts {
		if skipSentinel(d, want) {
			continue
		}
		switch f, ok := byName[d.Name]; {
		case ok:
			out = append(out, f)
		case d.HasAbsent:
			out = append(out, Fact{Name: d.Name, Kind: d.Kind, Value: d.Absent})
		}
	}
	return out
}

// emitFlat renders the facts as a FLAT JSON object of strings — the wire
// shape a declared command READER returns (intrastate RDR 0025: "read
// results return on stdout as a flat JSON object of strings").
//
// `--json` is the tool's own answer and nests: `{"facts":[{"name":…,
// "value":…}],"record":…}`. That is the right shape for a reader who
// wants the kinds and the record, and the wrong one for an accessor,
// which reads `{"<key>":"<value>"}` and nothing else. Rendering it here
// rather than asking the caller to reshape it is the same rule the rest
// of this file follows: when a consumer parses a projected string, the
// projection is missing a form.
//
// Values are `tagWords`' values, so a fact reads identically whether it
// crosses as argv or as a reader's object — including the absent
// sentinels, which a reader must carry (a declared key the object omits
// is UNREADABLE to the accessor, never established-absent). Prose facts
// are omitted for the reason they are omitted from `--tags`.
func emitFlat(tbl *FactTable, facts []Fact, want map[string]bool, stdout, stderr io.Writer) int {
	words, err := tagWords(tbl, facts, want)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	out := make(map[string]string, len(words))
	for _, text := range words {
		if name, value, ok := strings.Cut(text, "="); ok {
			out[name] = value
		}
	}
	return emit(out, stdout, stderr)
}

func emitTags(tbl *FactTable, facts []Fact, want map[string]bool, stdout, stderr io.Writer) int {
	words, err := tagWords(tbl, facts, want)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	for _, text := range words {
		fmt.Fprintf(stdout, "--tag\n%s\n", text)
	}
	return 0
}

// tagWords renders the facts as the `k=v` words a resolver's argv carries
// — the one rendering `--tags` and `--argv` share, so the two cannot
// disagree about a record.
func tagWords(tbl *FactTable, facts []Fact, want map[string]bool) ([]string, error) {
	prose := map[string]bool{}
	for _, d := range tbl.Facts {
		prose[d.Name] = d.Prose
	}
	facts = withAbsentSentinels(tbl, facts, want)
	var words []string
	for _, f := range facts {
		if prose[f.Name] {
			continue
		}
		text := f.Name + "=" + factValueText(f)
		// A non-prose fact that still would not survive means the table
		// is wrong about itself — an enum whose domain admits a spaced
		// value, a scalar carrying prose nobody declared. Emitting it
		// would truncate silently at the first space, so it stops, and
		// the message names the fact because the fix is in the table.
		if unsafeTagWord(f, text) {
			return nil, fmt.Errorf("stopped:unsafe-tag (%s renders %q, which an unquoted $(…) "+
				"would split or glob; declare it prose or fix its kind)", f.Name, factValueText(f))
		}
		words = append(words, text)
	}
	return words, nil
}

// emitArgv is the line-per-record form a shell loop reads: the record's
// slug, its Status, the `--tag k=v` argv, and the qualifier text, tab
// separated in that order.
//
// The qualifier is LAST and only the qualifier may be empty: `read` with
// a tab IFS collapses a run of tabs, so an empty middle field would shift
// every field after it, while a missing last field just reads empty. It
// is the one prose value the row carries (a parked record's revisit
// condition, a re-entry's scope), and it rides here so the caller never
// pays an `inspect` per row to see it. The argv is already one shell
// word per fact (`tagWords` refuses anything else), so an unquoted
// expansion under `set -f` hands the resolver exactly those words.
func emitArgv(tbl *FactTable, rows []statusRow, want map[string]bool, stdout, stderr io.Writer) int {
	for _, r := range rows {
		words, err := tagWords(tbl, r.Facts, want)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		argv := make([]string, 0, 2*len(words))
		for _, w := range words {
			argv = append(argv, "--tag", w)
		}
		status, qual := "", ""
		if r.Status != nil {
			status, qual = r.Status.Value, r.Status.Qualifier
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", recordSlug(r.Path), status, strings.Join(argv, " "), qual)
	}
	return 0
}

// unsafeShellWord reports whether a rendered tag would be mangled by an
// unquoted `$(…)`. Two hazards, both verified against `sh` and `bash`
// rather than reasoned about:
//
// WHITESPACE splits the word, always. `k=a b` is two arguments in every
// shell, and the first of them is a well-formed tag with a truncated
// value.
//
// GLOB metacharacters rewrite the word when it matches a path. This is
// not theoretical and not zsh's behaviour: under `sh` and `bash` — the
// shells a skill's Bash call actually runs — with files `k=a` and `k=b`
// present, the word `k=[ab]` expands to TWO arguments, `k=a` and `k=b`.
// The value is silently replaced by a filename.
//
// The test is static, and deliberately so. Asking the filesystem would
// be the shell's own rule, but it would make this tool's output depend
// on the caller's working directory — the same input answering
// differently in two places, which is the property every other facet
// here is built to avoid. A value is safe by its own bytes or it is
// refused.
//
// A rendered SET is exempt, and only a set: the tool writes those
// brackets itself, around members it has already quoted, so `[` and `]`
// appear at positions this code controls rather than in an author's
// value. `["0131","0132"]` cannot name a file. A metacharacter INSIDE a
// member is caught, because that came from a record.
func unsafeShellWord(s string) bool {
	return strings.ContainsAny(s, " \t\n\r\v\f") || strings.ContainsAny(s, "*?[")
}

// unsafeTagWord applies the rule to a whole `k=v` word, exempting the
// brackets a set rendering owns.
func unsafeTagWord(f Fact, word string) bool {
	if f.Kind != "set" {
		return unsafeShellWord(word)
	}
	if strings.ContainsAny(word, " \t\n\r\v\f") {
		return true
	}
	for _, m := range f.Members {
		if strings.ContainsAny(m, "*?[]") {
			return true
		}
	}
	return false
}

// jsonArray renders a set the way the seam carries it — the canonical
// compact array, which is also the only array shape that survives an
// unquoted command substitution as a single word.
func jsonArray(members []string) string {
	if len(members) == 0 {
		return "[]"
	}
	sorted := append([]string(nil), members...)
	sort.Strings(sorted)
	var b strings.Builder
	b.WriteByte('[')
	for i, m := range sorted {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%q", m)
	}
	b.WriteByte(']')
	return b.String()
}
