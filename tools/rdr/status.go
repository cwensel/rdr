package main

// `rdr status` — the navigator's read, in one call.
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
// Three renderings of one evaluation, because three different consumers
// need it: text for a person, `--json` for anything structured, `--tags`
// for a resolver's argv. The evaluation itself is identical in all three
// — a rendering that could disagree with another rendering of the same
// facts would be a second source of truth.
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

	"github.com/cwensel/rdr/tools/rdr/internal/scan"
)

// statusCmd is `rdr status [NNNN]`: one record's fact vector, or the
// in-flight worklist with each record's facts.
func statusCmd(args []string, f *flags, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "stopped:usage (status takes at most one NNNN, slug or path)")
		return 2
	}
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
	if len(args) == 0 {
		return statusWorklist(tbl, f, stdout, stderr)
	}
	return statusOne(tbl, args[0], f, stdout, stderr)
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
	facts := tbl.Evaluate(NewFactEnv(tbl, doc, recordSlug(path)))

	switch {
	case *f.tags:
		return emitTags(tbl, facts, stdout, stderr)
	case *f.json:
		return emitFacts(facts, doc.Record, stdout, stderr)
	}
	return emitFactLines(facts, stdout)
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
	docs, skipped, code := records(f, stderr)
	if code != 0 {
		return code
	}
	rows := []statusRow{}
	for _, d := range docs {
		s := scan.Summarize(d)
		if !s.InFlight {
			continue
		}
		rows = append(rows, statusRow{
			Summary: s,
			Facts:   tbl.Evaluate(NewFactEnv(tbl, d, recordSlug(s.Path))),
		})
	}
	if *f.json {
		return emit(map[string]any{
			"schema": schemaVersion, "records": rows, "skipped": skipped,
		}, stdout, stderr)
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
// The consuming call is `intrastate flow resolve --model … $(rdr status
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
func withAbsentSentinels(tbl *FactTable, facts []Fact) []Fact {
	have := make(map[string]bool, len(facts))
	for _, f := range facts {
		have[f.Name] = true
	}
	missing := false
	for _, d := range tbl.Facts {
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
		switch f, ok := byName[d.Name]; {
		case ok:
			out = append(out, f)
		case d.HasAbsent:
			out = append(out, Fact{Name: d.Name, Kind: d.Kind, Value: d.Absent})
		}
	}
	return out
}

func emitTags(tbl *FactTable, facts []Fact, stdout, stderr io.Writer) int {
	prose := map[string]bool{}
	for _, d := range tbl.Facts {
		prose[d.Name] = d.Prose
	}
	facts = withAbsentSentinels(tbl, facts)
	var lines []string
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
			fmt.Fprintf(stderr, "stopped:unsafe-tag (%s renders %q, which an unquoted $(…) "+
				"would split or glob; declare it prose or fix its kind)\n", f.Name, factValueText(f))
			return 2
		}
		lines = append(lines, text)
	}
	for _, text := range lines {
		fmt.Fprintf(stdout, "--tag\n%s\n", text)
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
