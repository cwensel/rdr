package main

// The fact table: rdr-status's signals, declared as data.
//
// `skills/rdr-status/SKILL.md` decides "what do I run next" from two
// halves — what the record says and what is on disk — and both halves
// were prose in that skill, re-derived by a model on every run. A prose
// signal table drifts from the tree it describes and nothing catches it;
// the corpus already holds a documented path that exists nowhere.
//
// So the signals move into `$RDR_HOME/models/rdr-facts.toml` and this
// file evaluates them. The tool holds the grammar, the skill holds the
// judgment: a fact says `spikes: false`, it does not say "Refine was
// judged done".
//
// Three rules govern everything here.
//
// A probe names ONE path. No globs, no patterns, no first-match. A path
// the tool has to guess is a path it can get wrong, and a lens that ran
// reading as un-run is the same class of error as a skipped check
// reading as a passed one.
//
// Absent is not false. A probe whose root is unbound emits nothing at
// all, exactly as `resolved` goes absent rather than false when no
// `--repo` was given. The consumer is a three-valued kernel: an omitted
// key leaves a rule undecided, where `false` decides it.
//
// The reader never re-parses what the projector already published. A
// Status qualifier's grammar, an assumption's vocabulary tier, a
// contract count — each is a field, because a consumer that has to parse
// a projected string means the projection is missing a field.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cwensel/rdr/tools/rdr/internal/ident"
	"github.com/cwensel/rdr/tools/rdr/internal/model"
	"github.com/cwensel/rdr/tools/rdr/internal/scan"
	"github.com/cwensel/rdr/tools/rdr/internal/toml"
)

// FactTable is a parsed rdr-facts.toml: the roots a probe hangs under
// and the facts themselves, in declaration order so the output is
// stable and diffable.
type FactTable struct {
	Version     int
	Description string
	Roots       map[string]FactRoot
	Facts       []FactDecl
	// Source is the path the table was read from, for error messages
	// that have to say which file disagreed.
	Source string
}

// FactRoot is a tree a probe's path is relative to: a seam var, plus the
// per-record suffix that turns it into this record's own directory.
type FactRoot struct {
	Name string
	// Var is the seam/environment variable naming the tree. Unbound is
	// not an error — every probe under it goes absent.
	Var string
	// Suffix is appended to the var's value, with `{slug}` replaced by
	// the record's slug.
	Suffix string
}

// FactDecl is one declared fact.
type FactDecl struct {
	Name string
	// Kind is the value's type as the downstream tag model spells it:
	// enum, bool, int, set or scalar. It is carried, not interpreted,
	// except that it decides how a value is rendered.
	Kind string
	// Source names the evaluator that answers this fact.
	Source string
	// Path is a projection path for a field, or the relative path for a
	// probe. Paths is the probe-any list.
	Path  string
	Paths []string
	// Root names the FactRoot a probe hangs under.
	Root string
	// Domain is an enum's declared values, checked at load: a fact whose
	// evaluator can produce a value outside its own domain is a bug the
	// table should catch, not export.
	Domain []string
	// Equals turns a field read into a bool: true when the field's value
	// equals this literal.
	Equals string
	// Transform names a normalisation applied to a field's value.
	Transform string
	// Select names which tally a ca-tally fact reports.
	Select string
	// Label is the verdict-line prefix a verdict-line fact looks for.
	Label string
	// Prose marks a fact whose value is free text rather than a token: a
	// sentence, a rationale tail, anything an author wrote for a reader.
	//
	// It exists because the renderings are not equally expressive. A
	// prose value is a JSON string in `--json` and nothing splits it, but
	// as a resolver's argv it crosses through an unquoted `$(…)` — which
	// splits on whitespace and globs the pieces — and arrives truncated
	// at the first space, as a well-formed tag carrying half a value. So
	// `--tags` omits these, and says so, rather than shattering one.
	//
	// The table declares it because that is where the knowledge lives: a
	// fact is prose by the author's design, not by the accident of what
	// one record happened to say. Discovering it at render time would
	// make `--tags` succeed or fail depending on which record was asked
	// about, which is the drift this whole file replaced.
	Prose bool
	// Absent is the value `--tags` renders when this fact evaluates to
	// nothing — the sentinel that carries "the record does not say".
	//
	// It exists because the two consumers of a fact are not equally
	// expressive about absence. `--json` omits an absent fact entirely
	// and a reader sees three values; a resolver's argv has no way to
	// spell "this key is absent" at all. Worse, a routing dimension that
	// is merely *sometimes* supplied cannot be routed on: declaring it
	// optional withholds the coverage proof at lint, and an absent value
	// under a guard atom refuses at resolve. Either way the navigator
	// stops on a record whose Profile field is simply missing — which is
	// a real corpus state, not an error.
	//
	// So a fact that a routing table discriminates on declares the value
	// absence takes, and `--tags` always renders the key. The sentinel is
	// a DECLARED member of the fact's own domain, so the table it feeds
	// claims that cell positively rather than defaulting into it.
	//
	// Only `--tags` substitutes. `--json` still omits, because there the
	// distinction survives and collapsing it would throw away the honest
	// answer this table's header insists on: false says "looked, not
	// there", absent says "nothing looked".
	Absent string
	// HasAbsent records that `absent` was declared, so an empty-string
	// sentinel (a set's `[]`, say) is not read as undeclared.
	HasAbsent bool
	// Min is an int's declared floor, checked at load.
	Min *int
	// Description is prose for the reader of the table.
	Description string
}

// factKinds are the value types the downstream tag model accepts. The
// list is closed on purpose: a kind that side cannot declare is a fact
// nothing can consume.
var factKinds = map[string]bool{
	"enum": true, "bool": true, "int": true, "set": true, "scalar": true,
}

// factTransforms are the normalisations a field may declare. Closed, so
// a misspelling is refused at load rather than quietly doing nothing.
var factTransforms = map[string]bool{
	"leading-word": true, "record-numbers": true, "record-numbers-any": true,
}

// factSources are the evaluators. Closed for the same reason a kind is:
// an unknown source in the table means the table is ahead of the binary,
// and reading it as "no fact" would silently drop a signal.
var factSources = map[string]bool{
	"field": true, "probe": true, "probe-any": true, "ca-tally": true,
	"ca-rollup": true, "verdict-line": true, "capsule-state": true,
	"cluster-member": true, "cluster-key": true,
}

// rootedSource are the sources whose paths hang under a declared root,
// and so must name a real root and an exact path. `cluster-member` reads
// a directory under its root rather than stat-ing one path, but the path
// it is GIVEN is still exact — the pattern ban applies to it for the same
// reason it applies to a probe.
var rootedSource = map[string]bool{
	"probe": true, "probe-any": true, "cluster-member": true,
	"cluster-key": true,
}

// LoadFactTable reads and validates a fact table.
//
// Validation is fail-fast and returns ONE categorized error, never a
// list: a table that does not load is not partially usable, and a caller
// staring at fourteen complaints fixes the first one anyway.
func LoadFactTable(path string) (*FactTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("stopped:no-fact-table (%v)", err)
	}
	doc, err := toml.Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("stopped:malformed-fact-table (%s: %v)", path, err)
	}

	t := &FactTable{Roots: map[string]FactRoot{}, Source: path}
	for _, tbl := range doc {
		switch {
		case tbl.Name == "facts":
			t.Version = tbl.Int("version")
			t.Description = tbl.Str("description")
		case strings.HasPrefix(tbl.Name, "root."):
			name := strings.TrimPrefix(tbl.Name, "root.")
			r := FactRoot{Name: name, Var: tbl.Str("var"), Suffix: tbl.Str("suffix")}
			if r.Var == "" {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: root %q names no var)", path, name)
			}
			t.Roots[name] = r
		case strings.HasPrefix(tbl.Name, "fact."):
			d, err := factFromTable(tbl)
			if err != nil {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: %v)", path, err)
			}
			t.Facts = append(t.Facts, d)
		default:
			return nil, fmt.Errorf("stopped:malformed-fact-table (%s: unknown table [%s])", path, tbl.Name)
		}
	}
	if t.Version != 1 {
		return nil, fmt.Errorf("stopped:unsupported-fact-table (%s: version %d; this binary reads version 1)", path, t.Version)
	}
	if len(t.Facts) == 0 {
		return nil, fmt.Errorf("stopped:malformed-fact-table (%s: declares no facts)", path)
	}
	seen := map[string]bool{}
	for _, f := range t.Facts {
		if seen[f.Name] {
			return nil, fmt.Errorf("stopped:malformed-fact-table (%s: fact %q declared twice)", path, f.Name)
		}
		seen[f.Name] = true
		if rootedSource[f.Source] && f.Root != "" {
			if _, ok := t.Roots[f.Root]; !ok {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: fact %q names undeclared root %q)", path, f.Name, f.Root)
			}
		}
	}
	return t, nil
}

// factFromTable validates one [fact.<name>] table into a declaration.
//
// Every key the table may carry is named here, and an unrecognised one
// is refused rather than ignored. A silently-dropped key is a fact that
// reads as declared and evaluates as something else — the failure this
// whole file exists to avoid.
func factFromTable(tbl toml.Table) (FactDecl, error) {
	name := strings.TrimPrefix(tbl.Name, "fact.")
	d := FactDecl{
		Name:        name,
		Kind:        tbl.Str("kind"),
		Source:      tbl.Str("source"),
		Path:        tbl.Str("path"),
		Paths:       tbl.List("paths"),
		Root:        tbl.Str("root"),
		Domain:      tbl.List("domain"),
		Equals:      tbl.Str("equals"),
		Transform:   tbl.Str("transform"),
		Select:      tbl.Str("select"),
		Label:       tbl.Str("label"),
		Description: tbl.Str("description"),
	}
	if v, ok := tbl.Scalar("min"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return d, fmt.Errorf("fact %q: min %q is not an int", name, v)
		}
		d.Min = &n
	}
	if v, ok := tbl.Scalar("prose"); ok {
		if v != "true" && v != "false" {
			return d, fmt.Errorf("fact %q: prose is true or false, got %q", name, v)
		}
		d.Prose = v == "true"
	}
	if v, ok := tbl.Scalar("absent"); ok {
		d.Absent, d.HasAbsent = v, true
	}
	for _, k := range tbl.Keys() {
		switch k {
		case "kind", "source", "path", "paths", "root", "domain",
			"equals", "transform", "select", "label", "min", "prose", "absent", "description":
		default:
			return d, fmt.Errorf("fact %q: unknown key %q", name, k)
		}
	}
	if !factNameOK(name) {
		return d, fmt.Errorf("fact %q: a fact name is lower-case, digits and underscore", name)
	}
	if !factKinds[d.Kind] {
		return d, fmt.Errorf("fact %q: unknown kind %q", name, d.Kind)
	}
	if !factSources[d.Source] {
		return d, fmt.Errorf("fact %q: unknown source %q", name, d.Source)
	}
	// A transform is named here for the same reason a key is: one this
	// switch does not know would silently apply NOTHING, and the fact
	// would read as declared while evaluating to the raw field — the
	// exact silent-drift this table replaced.
	if d.Transform != "" && !factTransforms[d.Transform] {
		return d, fmt.Errorf("fact %q: unknown transform %q", name, d.Transform)
	}
	switch d.Source {
	case "probe":
		if d.Root == "" || d.Path == "" {
			return d, fmt.Errorf("fact %q: a probe names a root and a path", name)
		}
	case "probe-any":
		if d.Root == "" || len(d.Paths) == 0 {
			return d, fmt.Errorf("fact %q: a probe-any names a root and paths", name)
		}
	case "field":
		if d.Path == "" {
			return d, fmt.Errorf("fact %q: a field names a path", name)
		}
	case "ca-tally":
		if d.Select == "" {
			return d, fmt.Errorf("fact %q: a ca-tally names a select", name)
		}
	case "verdict-line":
		if d.Label == "" {
			return d, fmt.Errorf("fact %q: a verdict-line names a label", name)
		}
	case "cluster-member", "cluster-key":
		if d.Root == "" || d.Path == "" {
			return d, fmt.Errorf("fact %q: a %s names a root and a path", name, d.Source)
		}
	}
	for _, p := range append([]string{d.Path}, d.Paths...) {
		if rootedSource[d.Source] && strings.ContainsAny(p, "*?[") {
			return d, fmt.Errorf("fact %q: %q is a pattern; a probe names one exact path", name, p)
		}
	}
	if d.Kind == "enum" && len(d.Domain) == 0 && d.Source != "capsule-state" {
		return d, fmt.Errorf("fact %q: an enum declares its domain", name)
	}
	// Only a scalar can be prose. An enum is its domain, a bool is two
	// literals, an int is digits and a set is a canonical array — each is
	// a token by construction, so `prose` on one would either be a lie or
	// a sign the kind is wrong.
	if d.Prose && d.Kind != "scalar" {
		return d, fmt.Errorf("fact %q: prose is for a scalar, not a %s", name, d.Kind)
	}
	// A sentinel is only meaningful for a fact something can route on,
	// and prose is the one kind nothing routes on — `--tags` omits it
	// wholesale, so a sentinel there would name a rendering that never
	// happens.
	if d.HasAbsent && d.Prose {
		return d, fmt.Errorf("fact %q: a prose fact is never rendered as a tag, so absent would not apply", name)
	}
	// The sentinel must be a value the fact could otherwise carry, or the
	// table it feeds cannot claim its cell. For an enum that means a
	// DECLARED member: routing on a value outside the domain is exactly
	// the drift this table replaced, and a downstream model declaring the
	// same domain would refuse it.
	if d.HasAbsent && d.Kind == "enum" && !slices.Contains(d.Domain, d.Absent) {
		return d, fmt.Errorf("fact %q: absent %q is not in the declared domain; add it, so a routing table can claim that cell", name, d.Absent)
	}
	if d.HasAbsent && d.Kind == "bool" && d.Absent != "true" && d.Absent != "false" {
		return d, fmt.Errorf("fact %q: absent %q is not a bool", name, d.Absent)
	}
	if d.HasAbsent && d.Kind == "int" {
		if _, err := strconv.Atoi(d.Absent); err != nil {
			return d, fmt.Errorf("fact %q: absent %q is not an int", name, d.Absent)
		}
	}
	return d, nil
}

func factNameOK(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

// Fact is one evaluated fact. A fact that could not be answered is not
// in the output at all, so there is no "unknown" value to render.
type Fact struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
	// Members carries a set's members, already canonical: sorted and
	// duplicate-free, the form the consuming seam compares by bytes.
	Members []string `json:"members,omitempty"`
}

// FactEnv is what the evaluator may look at: the record's projection,
// its slug, and the bound roots.
type FactEnv struct {
	Doc  *scan.Document
	Slug string
	// Roots maps a root name to its resolved directory, absent when the
	// var behind it is unbound.
	Roots map[string]string
	// readFile is the file reader, injectable so a test drives a
	// synthetic tree without reaching for the real corpus.
	readFile func(string) ([]byte, error)
	statPath func(string) (os.FileInfo, error)
	readDir  func(string) ([]os.DirEntry, error)
}

// NewFactEnv binds the roots a table declares from the seam.
//
// An unbound var is not an error and not a zero value: the root is
// simply not in the map, and every probe under it goes absent. Binding
// it to "" would root every probe at the filesystem root and answer
// false for all of them.
//
// A var naming a directory that does not exist is the SAME case, and for
// the reason this table's header gives: a root set to the wrong tree is
// worse than one left unset, because false says "looked, not there"
// while absent says "nothing looked". A stale or typo'd RDR_EVIDENCE
// would otherwise read every lens that ran as un-run — the failure the
// navigator warns about everywhere else.
//
// The BASE is what must exist, never the suffixed path: $RDR_EVIDENCE is
// a tree the consumer keeps, while <slug>/evidence under it is absent
// for any record that has produced no evidence yet, which is a true
// false and must stay one.
func NewFactEnv(t *FactTable, doc *scan.Document, slug string) *FactEnv {
	e := &FactEnv{Doc: doc, Slug: slug, Roots: map[string]string{},
		readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir}
	for name, r := range t.Roots {
		base := strings.TrimSpace(envOrSeam(r.Var))
		if base == "" {
			continue
		}
		if fi, err := os.Stat(base); err != nil || !fi.IsDir() {
			continue
		}
		e.Roots[name] = filepath.Join(base, strings.ReplaceAll(r.Suffix, "{slug}", slug))
	}
	return e
}

// Evaluate answers every fact the table declares, in declaration order.
func (t *FactTable) Evaluate(e *FactEnv) []Fact {
	out := []Fact{}
	for _, d := range t.Facts {
		if f, ok := t.evaluate(d, e); ok {
			out = append(out, f)
		}
	}
	return out
}

func (t *FactTable) evaluate(d FactDecl, e *FactEnv) (Fact, bool) {
	switch d.Source {
	case "probe":
		return e.probe(d, []string{d.Path})
	case "probe-any":
		return e.probe(d, d.Paths)
	case "field":
		return e.field(d)
	case "ca-tally":
		return e.caTally(d)
	case "ca-rollup":
		return e.caRollup(d)
	case "verdict-line":
		return e.verdictLine(d)
	case "capsule-state":
		return e.capsuleState(d)
	case "cluster-member":
		return e.clusterMember(d)
	case "cluster-key":
		return e.clusterKey(d)
	}
	return Fact{}, false
}

// probe answers whether an exact path exists under a bound root.
//
// The root being unbound is the absent case and the whole reason this
// returns two values: "no evidence root is configured" and "the lens did
// not run" are different answers, and only one of them should route.
func (e *FactEnv) probe(d FactDecl, paths []string) (Fact, bool) {
	base, ok := e.Roots[d.Root]
	if !ok {
		return Fact{}, false
	}
	for _, p := range paths {
		// `{slug}` appears only where a path is keyed by the record's slug
		// at the top of a root rather than under the record's own folder.
		p = strings.ReplaceAll(p, "{slug}", e.Slug)
		if _, err := e.statPath(filepath.Join(base, filepath.FromSlash(p))); err == nil {
			return Fact{Name: d.Name, Kind: d.Kind, Value: "true"}, true
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
}

// field reads a value the projector already published.
func (e *FactEnv) field(d FactDecl) (Fact, bool) {
	v, members, ok := e.lookup(d.Path)
	if !ok {
		return Fact{}, false
	}
	if d.Equals != "" {
		return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(v == d.Equals)}, true
	}
	switch d.Transform {
	case "leading-word":
		v = leadingWord(v)
	case "record-numbers":
		members = recordNumbers(v)
		v = ""
	case "record-numbers-any":
		// The set's CARDINALITY as a bool, for a routing dimension that
		// asks "is this record clustered at all" rather than "with
		// whom". A set cannot be a guard dimension — its domain is a
		// power set, not a partition — so the question a routing table
		// can actually decide is this one, and it is derived here rather
		// than re-derived from the members by every consumer.
		return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(len(recordNumbers(v)) > 0)}, true
	}
	if d.Kind == "set" {
		return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(members)}, true
	}
	if v == "" {
		return Fact{}, false
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// lookup walks a projection path. Only the paths the table actually
// declares are supported, and an unknown one is a load-time error rather
// than a silent absent, so this cannot quietly answer nothing.
func (e *FactEnv) lookup(path string) (string, []string, bool) {
	if e.Doc == nil {
		return "", nil, false
	}
	switch {
	case strings.HasPrefix(path, "metadata."):
		rest := strings.TrimPrefix(path, "metadata.")
		label, tail, _ := strings.Cut(rest, ".")
		f := metadataField(e.Doc, label)
		if f == nil {
			return "", nil, false
		}
		switch tail {
		case "value":
			return f.Value, nil, f.Value != ""
		case "status.value":
			if f.Status == nil {
				return "", nil, false
			}
			return f.Status.Value, nil, true
		case "status.form":
			if f.Status == nil {
				return "", nil, false
			}
			// A bare status carries no qualifier and the projector omits
			// the field; the form is "none", which is a real answer.
			if f.Status.Form == "" {
				return model.NoQualifier.String(), nil, true
			}
			return f.Status.Form, nil, true
		case "status.open_joint_decisions":
			if f.Status == nil {
				return "", nil, false
			}
			return "", f.Status.OpenJointDecisions, true
		}
		return "", nil, false
	case strings.HasPrefix(path, "counts.elements."):
		kind := ident.Kind(strings.TrimPrefix(path, "counts.elements."))
		n, ok := e.Doc.Counts.Elements[kind]
		if !ok {
			return "", nil, false
		}
		return strconv.Itoa(n), nil, true
	}
	return "", nil, false
}

func metadataField(doc *scan.Document, label string) *scan.Field {
	for i := range doc.Metadata {
		if doc.Metadata[i].Canonical == label || doc.Metadata[i].Label == label {
			return &doc.Metadata[i]
		}
	}
	return nil
}

// caTally counts Critical Assumptions by their Evidence Record Status.
//
// The projector has already normalised emphasis, trailing punctuation
// and case, and has already placed each value in a vocabulary tier, so
// this counts labels rather than re-reading prose. Placeholders and
// off-vocabulary values get their own buckets: the template legend left
// unfilled is not a Pending assumption, and a typo is not a verdict.
func (e *FactEnv) caTally(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	t := tallyAssumptions(e.Doc)
	n, ok := t[d.Select]
	if !ok {
		return Fact{}, false
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(n)}, true
}

// observedTerminal are the assumption verdicts the corpus writes and the
// template never listed (model.AssumptionStatusVocabulary's observed
// tier). Each one closes an assumption, so they tally as terminal rather
// than falling into neither column.
var observedTerminal = map[string]bool{
	"Refuted": true, "Resolved": true, "Accepted": true, "Downgraded": true,
}

func tallyAssumptions(doc *scan.Document) map[string]int {
	t := map[string]int{
		"total": 0, "Verified": 0, "Pending": 0, "Unverified": 0,
		"observed-terminal": 0, "placeholder": 0, "off-vocabulary": 0,
	}
	for _, el := range doc.Elements {
		if el.Kind != ident.Assumption {
			continue
		}
		for _, f := range el.Fields {
			if f.Canonical != "Status" || f.Status == nil {
				continue
			}
			t["total"]++
			switch {
			case f.Status.Placeholder:
				t["placeholder"]++
			case observedTerminal[f.Status.Value]:
				t["observed-terminal"]++
			case f.Status.Tier == model.OffVocabulary.String():
				t["off-vocabulary"]++
			default:
				if _, known := t[f.Status.Value]; known {
					t[f.Status.Value]++
				} else {
					t["off-vocabulary"]++
				}
			}
		}
	}
	return t
}

// caRollup reduces the tallies to the shape the routing branches on.
//
// `unknown-plan` is the honest answer whenever the list cannot certify a
// stage: no assumptions at all, a legend still unfilled, or a value in
// no vocabulary. An unreadable list is not an empty one, and reading it
// as all-terminal would certify Resolve off a record that never ran it.
func (e *FactEnv) caRollup(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	t := tallyAssumptions(e.Doc)
	if t["total"] == 0 || t["placeholder"] > 0 || t["off-vocabulary"] > 0 {
		return Fact{Name: d.Name, Kind: d.Kind, Value: "unknown-plan"}, true
	}
	terminal := t["Verified"] + t["observed-terminal"]
	open := t["Pending"] + t["Unverified"]
	switch {
	case open == t["total"]:
		return Fact{Name: d.Name, Kind: d.Kind, Value: "all-pending"}, true
	case terminal == t["total"]:
		return Fact{Name: d.Name, Kind: d.Kind, Value: "all-terminal"}, true
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "mixed"}, true
}

// verdictLine reports whether a Stage-2 verdict line is written in
// Decision Rationale.
//
// These lines are prose the projector does not carry — unlike
// `Joint-check:`, which IS projected as a JC element with a parsed
// verdict — so the only way to see them is to read that section's bytes.
// The read is bounded by the section's own line range, never the file.
func (e *FactEnv) verdictLine(d FactDecl) (Fact, bool) {
	if e.Doc == nil || e.Doc.Path == "" {
		return Fact{}, false
	}
	var start, end int
	for _, n := range e.Doc.Outline {
		if n.Canonical == "Decision Rationale" {
			start, end = n.LineStart, n.LineEnd
			break
		}
	}
	if start == 0 {
		return Fact{}, false
	}
	raw, err := e.readFile(e.Doc.Path)
	if err != nil {
		return Fact{}, false
	}
	lines := strings.Split(string(raw), "\n")
	if end > len(lines) {
		end = len(lines)
	}
	for i := start - 1; i < end && i < len(lines); i++ {
		if i < 0 {
			continue
		}
		// The label opens the line; emphasis around it is presentation,
		// the same reading the assumption-status parser applies.
		bare := strings.TrimSpace(strings.NewReplacer("**", "", "*", "", "_", "", "`", "", "- ", "").Replace(lines[i]))
		if strings.HasPrefix(bare, d.Label) {
			return Fact{Name: d.Name, Kind: d.Kind, Value: "true"}, true
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
}

// capsuleState reads the Stage-9 capsule's state word.
//
// Absent when the capsule is not there, and absent when it is there and
// states nothing this understands. The capsule header is the
// authoritative resume read precisely because it is written down;
// inferring a state from the tree instead is what it replaced.
func (e *FactEnv) capsuleState(d FactDecl) (Fact, bool) {
	base, ok := e.Roots["artifacts"]
	if !ok {
		return Fact{}, false
	}
	raw, err := e.readFile(filepath.Join(base, filepath.FromSlash(d.Path)))
	if err != nil {
		return Fact{}, false
	}
	// The header is a capsule at the top of the file; the state word is
	// read from it, not from anywhere the word may later appear in prose.
	lines := strings.Split(string(raw), "\n")
	if len(lines) > capsuleHeaderLines {
		lines = lines[:capsuleHeaderLines]
	}
	for _, ln := range lines {
		up := strings.ToUpper(ln)
		if !strings.Contains(up, "STATE") {
			continue
		}
		for _, want := range d.Domain {
			if strings.Contains(up, want) {
				return Fact{Name: d.Name, Kind: d.Kind, Value: want}, true
			}
		}
	}
	return Fact{}, false
}

// capsuleHeaderLines bounds the capsule read. The header is a fixed
// block at the top of status.md (phase/next/blocker/state in one pass);
// scanning the whole file would let a state word in a later narrative
// paragraph answer for the header.
const capsuleHeaderLines = 20

// numericClusterKey is the current shape of a Stage-7.1 output directory:
// its members' record numbers, joined. `0122-0123-0130-0131-0132`.
//
// It is anchored and total on purpose. The corpus also holds an EARLIER
// shape whose key is topical rather than derived — `dml-purpose`,
// `replay-perf-2026-06-25`, `final-cluster-2026-06-22` — and those are
// excluded here by shape, not read and then filtered. The difference
// matters: `final-cluster-2026-06-22` contains the four-digit run `2026`,
// so a rule that pulled numbers OUT of a name would mint a membership
// claim for a record 2026 that no run ever made. Requiring the whole name
// to be numbers-and-dashes cannot do that.
var numericClusterKey = regexp.MustCompile(`^\d{4}(?:-\d{4})+$`)

// clusterMember answers whether Stage 7.1 reconciled a set containing
// this record.
//
// This reads a directory rather than naming a path, which every other
// probe here refuses to do — so the reason it is allowed is worth
// stating. A probe may not GUESS a path, because a guess that misses
// reports a lens that ran as un-run. Nothing is guessed here: the key of
// a current-shape directory IS its membership, written by the run that
// made it, so the tool reads what the tree says instead of predicting
// what it might have been called. The alternative on offer was a table of
// sixteen hand-authored keys, which makes a DERIVED name into a
// maintained one and drifts the moment a run forgets to update it.
//
// Absent when the evidence root is unbound, like any probe: "no evidence
// root is configured" is not "7.1 has not run".
//
// The pre-2026-06-29 topical epoch answers false here and is declared as
// undeclared in the table. Every record it covers is terminal, so no
// routing reads it; the reason it stays out is that those directories may
// not be the same object at all — `final-cluster-2026-05-28` carries no
// whole-set critique and no pairwise scan, the two artifacts that define
// the stage, and predates `stages/07.1-cluster-reconcile.md` outright.
func (e *FactEnv) clusterMember(d FactDecl) (Fact, bool) {
	base, ok := e.Roots[d.Root]
	if !ok {
		return Fact{}, false
	}
	number := ident.RecordOf(e.Slug)
	if number == "" || e.readDir == nil {
		return Fact{}, false
	}
	entries, err := e.readDir(filepath.Join(base, filepath.FromSlash(d.Path)))
	if err != nil {
		// The tree has no cluster-reconcile directory at all. That is a
		// real answer — no cluster has ever been reconciled here — and
		// not the unbound-root case above.
		return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
	}
	for _, ent := range entries {
		if !ent.IsDir() || !numericClusterKey.MatchString(ent.Name()) {
			continue
		}
		for _, member := range strings.Split(ent.Name(), "-") {
			if member == number {
				return Fact{Name: d.Name, Kind: d.Kind, Value: "true"}, true
			}
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
}

// clusterKey names the Stage-7.1 directory that reconciled a set
// containing this record — the companion to `cluster-member`, which
// answers only WHETHER one did.
//
// The key IS the membership, so this is the one place a reconciled
// cluster's members can be read back without re-deriving them: the run
// that wrote the directory named it for the set it resolved, dropped
// candidates already excluded. `cluster-member` walked these same
// entries and discarded which one matched; this returns it, under the
// identical shape rule (`numericClusterKey`), so the two can never
// disagree about whether a record is covered.
//
// LONGEST MATCH, because a widened re-run leaves both directories
// standing. `0109-0120-0133` is iteration 1 and
// `0109-0120-0133-0134-0135` is iterations 2-3 over a set that grew; the
// longer key is the current membership and the shorter one is history.
// Overlaps on the reference corpus are all nested, so longest-match is
// total there; ties break on the name so the answer is deterministic
// rather than dependent on directory order.
//
// PROSE, and necessarily so: the value is a caveat line for a human and
// a set for a stage to re-enter on, and nothing routes on it —
// `cluster_reconciled` carries the routing half. Declaring it prose also
// keeps it out of `--tags`, where a resolver's argv has no use for it.
func (e *FactEnv) clusterKey(d FactDecl) (Fact, bool) {
	base, ok := e.Roots[d.Root]
	if !ok {
		return Fact{}, false
	}
	number := ident.RecordOf(e.Slug)
	if number == "" || e.readDir == nil {
		return Fact{}, false
	}
	entries, err := e.readDir(filepath.Join(base, filepath.FromSlash(d.Path)))
	if err != nil {
		// No cluster-reconcile tree at all. `cluster-member` calls that
		// false; here there is simply no key to name, and an absent fact
		// is how "nothing to say" is spelled.
		return Fact{}, false
	}
	best := ""
	for _, ent := range entries {
		name := ent.Name()
		if !ent.IsDir() || !numericClusterKey.MatchString(name) {
			continue
		}
		for _, member := range strings.Split(name, "-") {
			if member != number {
				continue
			}
			if len(name) > len(best) || (len(name) == len(best) && name < best) {
				best = name
			}
			break
		}
	}
	if best == "" {
		return Fact{}, false
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: best}, true
}

func boolLiteral(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// leadingWord takes the keyword off a value that carries a rationale
// tail — `mid — one contract plus the metrics surface`. §lens-row is
// explicit that a consumer matches the leading word and never the whole
// string.
func leadingWord(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if r == ' ' || r == '\t' || r == ',' || r == ';' || r == ':' || r == '—' || r == '-' {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// recordNumbers pulls the NNNN out of a comma-separated `NNNN-slug` list.
func recordNumbers(s string) []string {
	out := []string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if n := ident.RecordOf(part); n != "" && n != part {
			out = append(out, n)
			continue
		}
		if ident.RecordNumber.MatchString(part) {
			out = append(out, part)
		}
	}
	return out
}

// canonicalSet renders a set the way the consuming seam compares one:
// sorted and duplicate-free, so read-back equality is byte equality.
func canonicalSet(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// factTablePath finds the table.
//
// $RDR_HOME is the engine root and the marker exports it, so that is the
// first answer. Failing that, the binary installs at $RDR_HOME/bin/rdr,
// so `models/` beside the executable's own directory is the same place
// reached a different way — which is what keeps a `go test` binary and a
// directly-invoked build working with no marker at all.
func factTablePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if home := strings.TrimSpace(envOrSeam("RDR_HOME")); home != "" {
		p := filepath.Join(home, "models", factTableName)
		if fileExists(p) {
			return p, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		p := filepath.Join(filepath.Dir(filepath.Dir(exe)), "models", factTableName)
		if fileExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("stopped:no-fact-table (looked in $RDR_HOME/models and beside the binary; --facts names one)")
}

const factTableName = "rdr-facts.toml"

// bindSchema loads TEMPLATE.md and installs it for this process.
//
// The template is the schema, so a reader cannot proceed without it: the
// failure is a `stopped:` line and exit 2, never a zero schema that would
// classify every value off-vocabulary and report a missing install as a
// corpus-wide defect.
func bindSchema(f *flags) error {
	// A schema already installed stands: a test binary binds one in
	// TestMain, and re-resolving here would look for a marker the test
	// deliberately cleared. An explicit --template still rebinds.
	explicit := ""
	if f != nil && f.template != nil {
		explicit = *f.template
	}
	if explicit == "" && model.Bound() {
		return nil
	}
	tmpl, sidecar, err := templatePaths(explicit)
	if err != nil {
		return err
	}
	// The engine-root README is the authoritative Method vocabulary; it
	// says so itself, so the guidance never ships inside the template.
	readme, err := beside("", "README.md", "")
	if err != nil {
		return err
	}
	s, err := model.Load(tmpl, readme, sidecar)
	if err != nil {
		return err
	}
	model.Bind(s)
	return nil
}

// templatePaths finds TEMPLATE.md and its sidecar.
//
// It is factTablePath's twin, and resolves the same three ways: an
// explicit flag, then $RDR_HOME, then beside the binary. The template
// sits at the ENGINE ROOT and the sidecar under models/, so the two
// differ only in the join — the binary installs at $RDR_HOME/bin/rdr, so
// two Dir calls reach the root either way, which keeps a `go test` binary
// and a directly-invoked build working with no marker at all.
//
// --template names a different template; it does NOT redirect the
// sidecar, which ships with the binary and describes the reader rather
// than the document.
func templatePaths(explicit string) (tmpl, sidecar string, err error) {
	if sidecar, err = beside("models", sidecarName, ""); err != nil {
		return "", "", err
	}
	if tmpl, err = beside("", templateName, explicit); err != nil {
		return "", "", err
	}
	return tmpl, sidecar, nil
}

// beside resolves one engine-relative file: flag, then $RDR_HOME, then
// the executable's own root.
func beside(dir, name, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if home := strings.TrimSpace(envOrSeam("RDR_HOME")); home != "" {
		if p := filepath.Join(home, dir, name); fileExists(p) {
			return p, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if p := filepath.Join(filepath.Dir(filepath.Dir(exe)), dir, name); fileExists(p) {
			return p, nil
		}
	}
	if name == templateName {
		return "", fmt.Errorf("stopped:no-template (looked in $RDR_HOME and beside the binary; --template names one)")
	}
	return "", fmt.Errorf("stopped:no-template-sidecar (looked in $RDR_HOME/models and beside the binary)")
}

const (
	templateName = "TEMPLATE.md"
	sidecarName  = "rdr-template.toml"
)

// emitFacts renders an evaluated set as the neutral JSON vector.
func emitFacts(facts []Fact, record string, stdout, stderr interface{ Write([]byte) (int, error) }) int {
	payload := map[string]any{"schema": schemaVersion, "record": record, "facts": facts}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintf(stderr, "stopped:encode (%v)\n", err)
		return 2
	}
	return 0
}
