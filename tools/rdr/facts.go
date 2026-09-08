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

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
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
	// Iter is the re-entry convention `rdr paths` reads. Optional: a
	// table that declares none is still a valid fact table.
	Iter Iteration
	// Source is the path the table was read from, for error messages
	// that have to say which file disagreed.
	Source string
}

// Iteration is the re-entry convention, declared rather than compiled in.
// `rdr paths` reads it to answer where a lens writes and which pass is
// next; nothing else does, so an absent block simply leaves `paths`
// without trees to offer rather than breaking a fact.
type Iteration struct {
	// Segment is the directory a re-entry pass appends, with `{n}` the
	// iteration number. It is the whole vocabulary: a tree spelling it
	// otherwise is declared as its own tree, never pattern-matched.
	Segment string
	// First is how iteration 1 is spelled. "loose" means the files sit
	// directly under the base and no `iter-1` exists — which is what the
	// corpus does, and what makes "next" 2 rather than 1 on a base that
	// already holds a first pass.
	First string
	// Trees are the bases that take iterations, by name (`lens`,
	// `cluster`, …). The name is what `rdr paths` selects with.
	Trees map[string]IterTree
}

// IterTree is one base an iteration segment hangs under: a declared root,
// plus the path beneath it. `{lens}` / `{key}` are filled by the caller's
// operand, `{slug}` by the record — the same substitution a probe uses.
type IterTree struct {
	Name  string
	Root  string
	Under string
	// Cap is the iteration past which the tree's loop is flapping;
	// `rdr paths` buckets ITER against it. 0 means the tree declares
	// none, and then no bucket is emitted.
	Cap int
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
	// Legacy is a second suffix read only where a path is missing under
	// Suffix — a retained tolerance for records laid out before the
	// current shape, dropped when they migrate. Empty means none.
	Legacy string
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
	// Path is a projection path for a field, the relative path for a
	// probe, or the section a verdict-line reads. Paths is the probe-any
	// list.
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
	// Entry is a ledger-tally's line-matcher: a line matching it is one
	// ledger entry. Open and Closed are mutually exclusive: exactly one
	// names the regex that decides an entry is OPEN — Open matching it
	// directly, Closed by its ABSENCE.
	Entry  string
	Open   string
	Closed string
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
	// OnDemand marks a fact whose evaluation has a cost the unfiltered
	// render must not pay — the edge tallies resolve edges (a corpus scan
	// plus a repo walk, ~1s on a large record) for a navigator that calls
	// `--tags` several times a stage. Like `prose`, it is DECLARED: the
	// fact is evaluated only when `--filter` names it, and omitted from
	// every unfiltered render; a caller that wants it asks by name.
	OnDemand bool
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
	"cluster-member": true, "cluster-key": true, "header-field": true,
	"model-compare": true, "section-prose": true, "readme-row": true,
	"stale-lens": true, "stale-path": true, "seam-lineage": true, "contracts": true,
	"impl-artifact": true, "record-lines": true, "spike-diff": true,
	// the gate facts, evaluated in the block at the end of this file
	"assumption-ids": true, "reverify-ids": true, "edge-tally": true,
	"joint-check-home": true, "overlap-uncited": true,
	// the rollups: facts about the record's peers, read through FactEnv.Peer
	"predecessor-rollup": true, "cluster-proposed": true, "related-rollup": true,
	// the ledger readers: a lens's dispositions.md or evidence/rulings.md,
	// tallied by an entry/open-or-closed regex pair; and the deepest
	// iter-N segment reached under a tree.
	"ledger-tally": true, "iter-max": true,
}

// rootedSource are the sources whose paths hang under a declared root,
// and so must name a real root and an exact path. `cluster-member` reads
// a directory under its root rather than stat-ing one path, but the path
// it is GIVEN is still exact — the pattern ban applies to it for the same
// reason it applies to a probe.
var rootedSource = map[string]bool{
	"probe": true, "probe-any": true, "cluster-member": true,
	"cluster-key": true, "header-field": true, "readme-row": true,
	"stale-lens": true, "stale-path": true, "impl-artifact": true,
	"spike-diff": true, "ledger-tally": true, "iter-max": true,
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
			r := FactRoot{Name: name, Var: tbl.Str("var"), Suffix: tbl.Str("suffix"), Legacy: tbl.Str("legacy")}
			if r.Var == "" {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: root %q names no var)", path, name)
			}
			t.Roots[name] = r
		case tbl.Name == "iteration":
			t.Iter.Segment = tbl.Str("segment")
			t.Iter.First = tbl.Str("first")
		case strings.HasPrefix(tbl.Name, "iteration.tree."):
			name := strings.TrimPrefix(tbl.Name, "iteration.tree.")
			tr := IterTree{Name: name, Root: tbl.Str("root"), Under: tbl.Str("under"), Cap: tbl.Int("cap")}
			if c, ok := tbl.Scalar("cap"); ok && tr.Cap < 1 {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: iteration tree %q cap %q is not a positive integer)", path, name, c)
			}
			if tr.Root == "" {
				return nil, fmt.Errorf("stopped:malformed-fact-table (%s: iteration tree %q names no root)", path, name)
			}
			if t.Iter.Trees == nil {
				t.Iter.Trees = map[string]IterTree{}
			}
			t.Iter.Trees[name] = tr
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
	// An iteration tree is checked against the same roots, for the same
	// reason: a tree hanging off a root nobody declared would resolve to
	// a path under the filesystem root and report a first pass on every
	// record. Checked at LOAD so the table is wrong here, not in a
	// consumer's evidence dir.
	for _, tr := range t.Iter.Trees {
		if _, ok := t.Roots[tr.Root]; !ok {
			return nil, fmt.Errorf("stopped:malformed-fact-table (%s: iteration tree %q names undeclared root %q)", path, tr.Name, tr.Root)
		}
	}
	if len(t.Iter.Trees) > 0 && !strings.Contains(t.Iter.Segment, "{n}") {
		return nil, fmt.Errorf("stopped:malformed-fact-table (%s: iteration declares trees but segment %q carries no {n})", path, t.Iter.Segment)
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
		Entry:       tbl.Str("entry"),
		Open:        tbl.Str("open"),
		Closed:      tbl.Str("closed"),
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
	if v, ok := tbl.Scalar("on_demand"); ok {
		if v != "true" && v != "false" {
			return d, fmt.Errorf("fact %q: on_demand is true or false, got %q", name, v)
		}
		d.OnDemand = v == "true"
	}
	if v, ok := tbl.Scalar("absent"); ok {
		d.Absent, d.HasAbsent = v, true
	}
	for _, k := range tbl.Keys() {
		switch k {
		case "kind", "source", "path", "paths", "root", "domain",
			"equals", "transform", "select", "label", "min", "prose", "on_demand", "absent", "description",
			"entry", "open", "closed":
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
	case "header-field":
		if d.Root == "" || d.Path == "" || d.Label == "" {
			return d, fmt.Errorf("fact %q: a header-field names a root, a path and a label", name)
		}
	case "model-compare":
		if d.Root == "" || len(d.Paths) != 2 || d.Label == "" {
			return d, fmt.Errorf("fact %q: a model-compare names a root, a label and exactly two paths", name)
		}
	case "section-prose":
		if d.Path == "" {
			return d, fmt.Errorf("fact %q: a section-prose names the section it reads", name)
		}
	case "stale-lens":
		// The paths ARE the answer's vocabulary: the fact names the first
		// stale one, so every path must be a declared member and `none`
		// must be there for a row to claim the fresh cell.
		if d.Root == "" || len(d.Paths) == 0 || d.Kind != "enum" {
			return d, fmt.Errorf("fact %q: a stale-lens is an enum naming a root and paths", name)
		}
		if !slices.Contains(d.Domain, "none") {
			return d, fmt.Errorf("fact %q: a stale-lens domain declares none, the fresh answer", name)
		}
		for _, p := range d.Paths {
			if !slices.Contains(d.Domain, p) {
				return d, fmt.Errorf("fact %q: path %q is not in the domain it answers with", name, p)
			}
		}
	case "stale-path":
		if d.Root == "" || d.Path == "" || d.Kind != "bool" {
			return d, fmt.Errorf("fact %q: a stale-path is a bool naming a root and a path", name)
		}
	case "assumption-ids", "edge-tally":
		if d.Select == "" {
			return d, fmt.Errorf("fact %q: a %s names a select", name, d.Source)
		}
	case "spike-diff":
		if d.Root == "" || d.Kind != "set" {
			return d, fmt.Errorf("fact %q: a spike-diff is a set naming a root", name)
		}
	case "impl-artifact":
		// The six selects are the six ledger reads; an unknown one
		// would evaluate to nothing while reading as declared, the same
		// failure seam-lineage's select guards against.
		switch d.Select {
		case "req-count", "orphans", "open-decisions", "mvv-recorded", "verification-recorded", "impact-families", "findings-open":
		default:
			return d, fmt.Errorf("fact %q: an impl-artifact selects req-count, orphans, open-decisions, mvv-recorded, verification-recorded, impact-families or findings-open, got %q", name, d.Select)
		}
		if d.Root == "" {
			return d, fmt.Errorf("fact %q: an impl-artifact names a root", name)
		}
		switch d.Select {
		case "orphans":
			if len(d.Paths) != 2 {
				return d, fmt.Errorf("fact %q: orphans names exactly two paths (req-list, coverage)", name)
			}
		default:
			if len(d.Paths) != 1 {
				return d, fmt.Errorf("fact %q: a %s impl-artifact names one path", name, d.Select)
			}
		}
		if d.Select == "open-decisions" || d.Select == "mvv-recorded" || d.Select == "findings-open" {
			if d.Label == "" {
				return d, fmt.Errorf("fact %q: a %s impl-artifact names a label", name, d.Select)
			}
		}
		// verification-recorded carries no label: its ids (FAIL-N/ADV-N)
		// and its Verdict line are fixed by the launch prompt, so the
		// code owns them rather than the declaration.
		if d.Select == "mvv-recorded" || d.Select == "verification-recorded" {
			if d.Kind != "bool" {
				return d, fmt.Errorf("fact %q: %s is a bool", name, d.Select)
			}
		} else {
			if d.Kind != "enum" {
				return d, fmt.Errorf("fact %q: %s is an enum", name, d.Select)
			}
			for _, m := range implArtifactMembers[d.Select] {
				if !slices.Contains(d.Domain, m) {
					return d, fmt.Errorf("fact %q: domain does not declare %q, which %s can emit", name, m, d.Select)
				}
			}
		}
	case "record-lines":
		// The bucket's members are fixed by the evaluator, so the domain
		// must declare them — the same closed-domain check impl-artifact
		// makes, for the same reason: a member the domain does not name
		// is a value no routing row can claim.
		if d.Kind != "enum" {
			return d, fmt.Errorf("fact %q: a record-lines is an enum", name)
		}
		for _, m := range recordLinesMembers {
			if !slices.Contains(d.Domain, m) {
				return d, fmt.Errorf("fact %q: domain does not declare %q, which record-lines can emit", name, m)
			}
		}
	case "predecessor-rollup":
		// Three selects: the set that owes work, the set closed without
		// implementing, and the state word the launch precheck routes on.
		// The enum's members are fixed by the evaluator, so the domain
		// must declare them.
		switch d.Select {
		case "incomplete", "retired":
			if d.Kind != "set" {
				return d, fmt.Errorf("fact %q: %s is a set", name, d.Select)
			}
		case "state":
			if d.Kind != "enum" {
				return d, fmt.Errorf("fact %q: state is an enum", name)
			}
			for _, m := range predecessorStates {
				if !slices.Contains(d.Domain, m) {
					return d, fmt.Errorf("fact %q: domain does not declare %q, which state can emit", name, m)
				}
			}
		default:
			return d, fmt.Errorf("fact %q: a predecessor-rollup selects incomplete, retired or state, got %q", name, d.Select)
		}
	case "cluster-proposed":
		if d.Path == "" || d.Kind != "enum" {
			return d, fmt.Errorf("fact %q: a cluster-proposed is an enum naming the section it reads", name)
		}
		for _, m := range clusterProposedMembers {
			if !slices.Contains(d.Domain, m) {
				return d, fmt.Errorf("fact %q: domain does not declare %q, which cluster-proposed can emit", name, m)
			}
		}
	case "related-rollup":
		if d.Kind != "enum" {
			return d, fmt.Errorf("fact %q: a related-rollup is an enum", name)
		}
		switch d.Select {
		case "", "final-unimplemented", "draft":
		default:
			return d, fmt.Errorf("fact %q: a related-rollup selects final-unimplemented or draft, got %q", name, d.Select)
		}
		for _, m := range relatedRollupMembers {
			if !slices.Contains(d.Domain, m) {
				return d, fmt.Errorf("fact %q: domain does not declare %q, which related-rollup can emit", name, m)
			}
		}
	case "ledger-tally":
		if d.Root == "" || d.Path == "" || d.Entry == "" {
			return d, fmt.Errorf("fact %q: a ledger-tally names a root, a path and an entry", name)
		}
		if (d.Open == "") == (d.Closed == "") {
			return d, fmt.Errorf("fact %q: a ledger-tally names exactly one of open or closed", name)
		}
		if _, err := regexp.Compile(d.Entry); err != nil {
			return d, fmt.Errorf("fact %q: entry %q does not compile: %v", name, d.Entry, err)
		}
		if d.Open != "" {
			if _, err := regexp.Compile(d.Open); err != nil {
				return d, fmt.Errorf("fact %q: open %q does not compile: %v", name, d.Open, err)
			}
		}
		if d.Closed != "" {
			if _, err := regexp.Compile(d.Closed); err != nil {
				return d, fmt.Errorf("fact %q: closed %q does not compile: %v", name, d.Closed, err)
			}
		}
		switch d.Kind {
		case "int":
		case "enum":
			if !slices.Contains(d.Domain, "0") || !slices.Contains(d.Domain, "1+") {
				return d, fmt.Errorf("fact %q: an enum ledger-tally declares domain 0 and 1+", name)
			}
		default:
			return d, fmt.Errorf("fact %q: a ledger-tally is an int or an enum, got %q", name, d.Kind)
		}
	case "iter-max":
		if d.Root == "" {
			return d, fmt.Errorf("fact %q: an iter-max names a root", name)
		}
		if d.Kind != "int" {
			return d, fmt.Errorf("fact %q: an iter-max is an int", name)
		}
	case "seam-lineage":
		// The three selects are the three shapes the floor reads, and an
		// unknown one would evaluate to nothing while reading as declared.
		switch d.Select {
		case "count", "bucket", "disposition":
		default:
			return d, fmt.Errorf("fact %q: a seam-lineage selects count, bucket or disposition, got %q", name, d.Select)
		}
	case "contracts":
		// Two selects: the Transient-marked count, and the durable bucket
		// the `profile` rows read. The subtraction happens here, once.
		switch d.Select {
		case "transient", "durable":
		default:
			return d, fmt.Errorf("fact %q: a contracts selects transient or durable, got %q", name, d.Select)
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
	// An on-demand fact MAY declare one: it is rendered only under
	// `--filter`, and a filtered `--tags` renders the sentinels of the
	// facts asked for (withAbsentSentinels reads the same want set), so a
	// routing group that guards on an on-demand fact still receives the
	// key when nothing was looked.
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
	// Legacy maps a root name to its second, pre-migration directory,
	// present only for a root that declares one. `under` reads it.
	Legacy map[string]string
	// readFile is the file reader, injectable so a test drives a
	// synthetic tree without reaching for the real corpus.
	readFile func(string) ([]byte, error)
	statPath func(string) (os.FileInfo, error)
	readDir  func(string) ([]os.DirEntry, error)
	// Want, when set, names the facts to evaluate and the rest are
	// skipped before they cost anything: `--filter` applied at
	// evaluation rather than after it, so a call asking only for the
	// cheap facts never pays the edge resolution the tallies need.
	Want map[string]bool
	// ResolveEdges decides the record's edges on first need (edgeTally),
	// or is nil when the caller bound no corpus — then every anchor is
	// unlooked, which is the honest answer.
	ResolveEdges  func()
	edgesResolved bool
	// Peer answers another record's env by number — the rollup facts
	// (predecessors, cluster siblings, related Finals) read a peer's
	// capsule or section through it. Nil, or false, is "nothing looked":
	// the peer is reported unresolved, never as not-COMPLETE.
	Peer func(record string) (*FactEnv, bool)
	// Corpus is every record in the dir, on first need, for the one fact
	// that walks the edge graph (related-rollup). Nil leaves it absent.
	Corpus func() []*scan.Document
	// table is the declaring table, so a rollup can evaluate a peer's
	// capsule state through the same declaration this record's uses.
	table *FactTable
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
		readFile: os.ReadFile, statPath: os.Stat, readDir: os.ReadDir, table: t}
	for _, r := range t.Roots {
		base := strings.TrimSpace(envOrSeam(r.Var))
		if base == "" {
			continue
		}
		if fi, err := os.Stat(base); err != nil || !fi.IsDir() {
			continue
		}
		e.bindRoot(r, base)
	}
	return e
}

// bindRoot resolves one root's directories for this record. Tests bind
// through it too, so a test can never bind a root differently from the
// seam.
func (e *FactEnv) bindRoot(r FactRoot, base string) {
	e.Roots[r.Name] = filepath.Join(base, strings.ReplaceAll(r.Suffix, "{slug}", e.Slug))
	if r.Legacy != "" {
		if e.Legacy == nil {
			e.Legacy = map[string]string{}
		}
		e.Legacy[r.Name] = filepath.Join(base, strings.ReplaceAll(r.Legacy, "{slug}", e.Slug))
	}
}

// under resolves a relative path beneath a bound root, PER PATH: the
// canonical directory when the path is there or the root declares no
// legacy leg; the legacy directory when the path is missing under the
// canonical one and present under the legacy one; the canonical path
// otherwise, so a file missing from both keeps reading false or absent
// exactly as it did with one root. Canonical first is the point — a
// record carrying both reads the current one — and the legacy leg is
// looked at only for the path in hand, because the two layouts can
// share a record (a gate written under the current tree beside a
// capsule written under the old one). False means the root is unbound.
func (e *FactEnv) under(root, rel string) (string, bool) {
	base, ok := e.Roots[root]
	if !ok {
		return "", false
	}
	p := filepath.Join(base, filepath.FromSlash(rel))
	if old, ok := e.Legacy[root]; ok {
		if _, err := e.statPath(p); err != nil {
			q := filepath.Join(old, filepath.FromSlash(rel))
			if _, err := e.statPath(q); err == nil {
				return q, true
			}
		}
	}
	return p, true
}

// Evaluate answers every fact the table declares, in declaration order.
func (t *FactTable) Evaluate(e *FactEnv) []Fact {
	out := []Fact{}
	for _, d := range t.Facts {
		if e.Want != nil && !e.Want[d.Name] || e.Want == nil && d.OnDemand {
			continue
		}
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
	case "header-field":
		return e.headerField(d)
	case "model-compare":
		return e.modelCompare(d)
	case "section-prose":
		return e.sectionProse(d)
	case "readme-row":
		return e.readmeRow(d)
	case "seam-lineage":
		return e.seamLineage(d)
	case "contracts":
		return e.contractCounts(d)
	case "stale-lens":
		return e.staleLens(d)
	case "stale-path":
		return e.stalePath(d)
	case "assumption-ids":
		return e.assumptionIDs(d)
	case "reverify-ids":
		return e.reverifyIDs(d)
	case "edge-tally":
		return e.edgeTally(d)
	case "joint-check-home":
		return e.jointCheckHome(d)
	case "overlap-uncited":
		return e.overlapUncited(d)
	case "impl-artifact":
		return e.implArtifact(d)
	case "record-lines":
		return e.recordLines(d)
	case "spike-diff":
		return e.spikeDiff(d)

	case "predecessor-rollup":
		return e.predecessorRollup(d)
	case "cluster-proposed":
		return e.clusterProposed(d)
	case "related-rollup":
		return e.relatedRollup(d)
	case "ledger-tally":
		return e.ledgerTally(d)
	case "iter-max":
		return e.iterMax(d)
	}
	return Fact{}, false
}

// probe answers whether an exact path exists under a bound root.
//
// The root being unbound is the absent case and the whole reason this
// returns two values: "no evidence root is configured" and "the lens did
// not run" are different answers, and only one of them should route.
func (e *FactEnv) probe(d FactDecl, paths []string) (Fact, bool) {
	if _, ok := e.Roots[d.Root]; !ok {
		return Fact{}, false
	}
	for _, p := range paths {
		// `{slug}` appears only where a path is keyed by the record's slug
		// at the top of a root rather than under the record's own folder.
		p = strings.ReplaceAll(p, "{slug}", e.Slug)
		full, _ := e.under(d.Root, p)
		if _, err := e.statPath(full); err == nil {
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
		case "status.reentry_target":
			// Empty is ABSENT, not a value: a re-entry that names no
			// target, or any other form, has no answer here. `--tags`
			// renders the declared sentinel so the routing can claim it.
			if f.Status == nil || f.Status.ReentryTarget == "" {
				return "", nil, false
			}
			return f.Status.ReentryTarget, nil, true
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

// seamLineage answers the accretion floor's inputs off the projected
// Seam Lineage field (scan.SeamLineage): the count, the count bucketed
// for routing, or whether the written disposition is present. The floor
// itself — count >= 2 with no disposition — is a ROW in rdr-status.toml,
// not a comparison made here; this only publishes what the row reads.
//
//	count        the int, absent when there is no field or no readable
//	             count — like `contracts`, nothing routes on it directly
//	bucket       0 | 1 | 2+ | unread; absent when there is no field, so
//	             the table's `none` sentinel claims that cell. `unread`
//	             is a REAL value: the field is written and its count is
//	             in no declared form, which is a stop, not a zero.
//	disposition  true when an `Accretion disposition:` line is written;
//	             false when the field is present without one; absent
//	             when there is no field (the table's sentinel is false).
func (e *FactEnv) seamLineage(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	f := metadataField(e.Doc, "Seam Lineage")
	if f == nil || f.Seam == nil {
		return Fact{}, false
	}
	sl := f.Seam
	switch d.Select {
	case "count":
		if sl.Count == nil {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(*sl.Count)}, true
	case "bucket":
		if sl.Form == "placeholder" {
			return Fact{}, false // an unfilled seed has no field to read
		}
		if sl.Count == nil {
			return Fact{Name: d.Name, Kind: d.Kind, Value: "unread"}, true
		}
		v := strconv.Itoa(*sl.Count)
		if *sl.Count >= 2 {
			v = "2+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	case "disposition":
		if sl.Form == "placeholder" {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(sl.Disposition)}, true
	}
	return Fact{}, false
}

// contractCounts answers the Profile sizing's contract axis off the
// projected contract elements: how many carry the Transient marker
// (scan.Element.Transient), and the DURABLE count — labelled minus
// Transient — bucketed for routing. TEMPLATE.md's Normative Contracts
// says a Transient-marked contract "counts toward neither the Profile
// contract axis nor the split signal"; that subtraction is made here,
// once, so the `profile` rows in rdr-write.toml never do arithmetic.
//
//	transient  the int; absent only when there is no document
//	durable    0 | 1 | 2+ over labelled − transient. A record whose
//	           contracts are prose has no labelled elements and reads
//	           0 — `contracts_prose` is what tells that apart from an
//	           empty section, and the rows read both.
func (e *FactEnv) contractCounts(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	total, transient := 0, 0
	for _, el := range e.Doc.Elements {
		if el.Kind != ident.Contract {
			continue
		}
		total++
		if el.Transient {
			transient++
		}
	}
	switch d.Select {
	case "transient":
		return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(transient)}, true
	case "durable":
		v := strconv.Itoa(total - transient)
		if total-transient >= 2 {
			v = "2+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	return Fact{}, false
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

// verdictLine reads a labelled verdict line from one section — the
// Stage-2 `Premortem:`/`Ground-sweep:` lines in Decision Rationale, and
// the Stage-5 `Determinacy:` line in Normative Contracts.
//
// These lines are prose the projector does not carry — unlike
// `Joint-check:`, which IS projected as a JC element with a parsed
// verdict — so the only way to see them is to read that section's bytes.
// The read is bounded by the section's own line range, never the file.
// `path` names the section (Decision Rationale when unset). A bool fact
// answers whether the line is there; an enum fact answers its leading
// word, normalised (`n/a` reads `na`), and goes ABSENT when there is no
// line or the word is off the declared domain — the honest answer, and
// the one the table's sentinel names, rather than a member invented for
// it (the header-field rule).
func (e *FactEnv) verdictLine(d FactDecl) (Fact, bool) {
	if e.Doc == nil || e.Doc.Path == "" {
		return Fact{}, false
	}
	section := d.Path
	if section == "" {
		section = "Decision Rationale"
	}
	var start, end int
	for _, n := range e.Doc.Outline {
		if n.Canonical == section {
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
	inMarker := false
	for i := start - 1; i < end && i < len(lines); i++ {
		if i < 0 {
			continue
		}
		// Template guidance (`[Required — …]`) is not content, the same
		// reading sectionProse applies: a line inside a marker block
		// cannot be the verdict, however it is worded.
		t := strings.TrimSpace(lines[i])
		if inMarker {
			inMarker = !strings.Contains(t, "]")
			continue
		}
		if strings.HasPrefix(t, "[") && !strings.Contains(t, "]") {
			inMarker = true
			continue
		}
		// The label opens the line; emphasis around it is presentation,
		// the same reading the assumption-status parser applies.
		bare := strings.TrimSpace(strings.NewReplacer("**", "", "*", "", "_", "", "`", "", "- ", "").Replace(lines[i]))
		if !strings.HasPrefix(bare, d.Label) {
			continue
		}
		if d.Kind != "enum" {
			return Fact{Name: d.Name, Kind: d.Kind, Value: "true"}, true
		}
		v := verdictWord(strings.TrimSpace(bare[len(d.Label):]))
		if v == "" || !slices.Contains(d.Domain, v) {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	if d.Kind == "enum" {
		return Fact{}, false
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
}

// verdictWord is the enum reading of a verdict line's value: the leading
// whitespace-delimited token (headerValue's rule — what follows is a note
// to a human), lower-cased, a trailing stop dropped, and `n/a` folded to
// `na`, the spelling a tag can carry.
func verdictWord(s string) string {
	w := strings.ToLower(strings.TrimRight(headerValue(s), ".,;:"))
	if w == "n/a" {
		return "na"
	}
	return w
}

// capsuleState reads the Stage-9 capsule's state word.
//
// Absent when the capsule is not there, and absent when it is there and
// states nothing this understands. The capsule header is the
// authoritative resume read precisely because it is written down;
// inferring a state from the tree instead is what it replaced.
func (e *FactEnv) capsuleState(d FactDecl) (Fact, bool) {
	full, ok := e.under("artifacts", d.Path)
	if !ok {
		return Fact{}, false
	}
	raw, err := e.readFile(full)
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

// implArtifactMembers are the values each impl-artifact select can emit,
// checked against the declared domain at load — the same closed-domain
// discipline stale-lens applies to its own paths.
var implArtifactMembers = map[string][]string{
	"req-count":       {"0-10", "11+"},
	"orphans":         {"0", "1+"},
	"open-decisions":  {"0", "1+", "midline"},
	"impact-families": {"0", "1+"},
	"findings-open":   {"0", "1+"},
}

// implArtifact answers the Stage-8 launch-gate signals, read from the
// artifact ledger a launch prompt writes beside the capsule.
//
// Absent for the reason every other artifact-root fact is absent: an
// unbound root or a missing file means nothing was read, and `none` (or
// `false` for the bool) is the --tags sentinel a launch table's row can
// claim positively — never a guess this file exists.
func (e *FactEnv) implArtifact(d FactDecl) (Fact, bool) {
	if _, ok := e.Roots[d.Root]; !ok {
		return Fact{}, false
	}
	read := func(i int) ([]byte, bool) {
		full, _ := e.under(d.Root, d.Paths[i])
		raw, err := e.readFile(full)
		if err != nil {
			return nil, false
		}
		return raw, true
	}
	switch d.Select {
	case "req-count":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		n := len(reqIDs(raw))
		v := "0-10"
		if n > 10 {
			v = "11+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	case "orphans":
		reqRaw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		covRaw, ok := read(1)
		if !ok {
			return Fact{}, false
		}
		reqs := reqIDs(reqRaw)
		covered, emptyCell := coverageRows(covRaw)
		n := 0
		for id := range reqs {
			if !covered[id] {
				n++
			}
		}
		for id := range covered {
			if !reqs[id] {
				n++
			}
		}
		for id := range emptyCell {
			if reqs[id] && covered[id] {
				n++
			}
		}
		v := "0"
		if n > 0 {
			v = "1+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	case "open-decisions":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		n, midline := openDecisions(raw, d.Label)
		v := "0"
		switch {
		case midline:
			// Precedence over a count: a mid-line Status key means the
			// count below cannot be trusted, whatever it says.
			v = "midline"
		case n > 0:
			v = "1+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	case "mvv-recorded":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(labelledLine(raw, d.Label))}, true
	case "verification-recorded":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(verificationRun(raw))}, true
	case "findings-open":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		v := "0"
		if openFindings(raw, d.Label) > 0 {
			v = "1+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	case "impact-families":
		raw, ok := read(0)
		if !ok {
			return Fact{}, false
		}
		n, ok := impactFamilies(raw)
		if !ok {
			// A file with no readable `families:` line is not the
			// projection's output: unread, so the shard route stops on
			// it rather than reading a stray file as an empty radius.
			return Fact{}, false
		}
		v := "0"
		if n > 0 {
			v = "1+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	return Fact{}, false
}

// impactFamilies reads the `families: N` header line `rdr impact` writes
// (emitImpact); the first such line decides, and a file without one, or
// with a non-integer after the key, is unreadable.
func impactFamilies(raw []byte) (int, bool) {
	for _, ln := range strings.Split(string(raw), "\n") {
		rest, ok := strings.CutPrefix(ln, "families:")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil || n < 0 {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// recordLinesCap is the size gate's line cap: a record at or under it is
// `0-400`, above it `401+`. The cap lives here beside req-count's 10 for
// the same reason — the bucket IS the fact, and a routing row compares
// against a member, never against a number it would have to bucket
// itself.
const recordLinesCap = 400

// recordLinesMembers are the two values record-lines can emit, checked
// against the declared domain at load.
var recordLinesMembers = []string{"0-400", "401+"}

// recordLines buckets the record's own line count — the same number
// `inspect` reports as top-level `lines` — so the size gate reads its cap
// as a tag rather than comparing a count in prose. Absent only when there
// is no document at all; every record has a length.
func (e *FactEnv) recordLines(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	v := recordLinesMembers[0]
	if e.Doc.Lines > recordLinesCap {
		v = recordLinesMembers[1]
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// stripLead strips the markup an author's list item or header carries in
// front of the content the parsers below match on: bullet markers, bold,
// backticks and heading hashes, repeatedly — `- **[REQ-1]**` and
// `#### **REQ-MVV output**` both reduce to their bare content.
func stripLead(s string) string {
	s = strings.TrimSpace(s)
	for {
		trimmed := s
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimPrefix(trimmed, "**")
		trimmed = strings.TrimPrefix(trimmed, "`")
		trimmed = strings.TrimLeft(trimmed, "#")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == s {
			return s
		}
		s = trimmed
	}
}

// reqIDBody is the one shape of a REQ id both parsers share: dot-joined
// segments, optionally hyphen-joined — `REQ-1`, `REQ-7.a-iv`, `REQ-OVR-1`.
// A single definition, because the two files are compared by id: a
// prefix-only reading in one and an anchored one in the other turns
// `REQ-OVR-1` into a phantom `REQ-OVR` orphan.
const reqIDBody = `REQ-[A-Za-z0-9.]+(?:-[A-Za-z0-9.]+)*`

// reqLineID matches a leading `[REQ-<id>]` once the line's markup is
// stripped — the ledger's one authoring form, never guessed from prose
// elsewhere on the line.
var reqLineID = regexp.MustCompile(`^\[(` + reqIDBody + `)\]`)

// reqIDs collects the DISTINCT [REQ-<id>] ids a req-list.md declares,
// excluding REQ-MVV — the size gate counts requirements, not the MVV
// acceptance line that travels beside them.
func reqIDs(raw []byte) map[string]bool {
	out := map[string]bool{}
	for _, ln := range strings.Split(string(raw), "\n") {
		bare := stripLead(ln)
		m := reqLineID.FindStringSubmatch(bare)
		if m == nil {
			continue
		}
		id := m[1]
		if strings.HasPrefix(id, "REQ-MVV") {
			continue
		}
		out[id] = true
	}
	return out
}

// reqCellID matches a markdown table cell's leading REQ id, emphasis and
// backticks stripped — coverage.md's first column.
var reqCellID = regexp.MustCompile(`^(` + reqIDBody + `)`)

// coverageRows reads coverage.md's table: the REQ ids its rows cover, and
// which of those rows carry an empty second (test) cell — a row present
// without a test is an orphan by this table's own reading, not a parse
// failure.
//
// The header row, a literal `REQ-N` placeholder row and `---` separator
// rows are skipped by shape: `REQ-N` is the template's own example id,
// never a real one the ledger would emit, and neither carries a real
// cell to read.
func coverageRows(raw []byte) (ids map[string]bool, emptyCell map[string]bool) {
	ids = map[string]bool{}
	emptyCell = map[string]bool{}
	for _, ln := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(t, "|"), "|")
		if len(cells) == 0 {
			continue
		}
		first := strings.TrimSpace(stripLead(strings.TrimSpace(cells[0])))
		first = strings.Trim(first, "*`")
		if strings.Trim(first, "-: ") == "" {
			continue // the --- separator row
		}
		m := reqCellID.FindStringSubmatch(first)
		if m == nil {
			continue // the header row, or any non-REQ lead
		}
		id := m[1]
		if id == "REQ-N" {
			continue // the template's own placeholder row
		}
		if strings.HasPrefix(id, "REQ-MVV") {
			continue
		}
		ids[id] = true
		if len(cells) < 2 || strings.TrimSpace(cells[1]) == "" {
			emptyCell[id] = true
		}
	}
	return ids, emptyCell
}

// openFindings counts triage.md's still-open rows: an IN-SCOPE verdict
// (the label, matched by prefix so a qualified form like `IN-SCOPE (C2-4:
// …)` still counts) whose Outcome cell has not settled to `fixed:<sha>`
// or `held:<reason>`. Everything else in that state — a bare `open`, an
// empty cell, or a shipped-as-kata `kata ab12` — is still open work, not
// a closed one wearing different words, because only those two prefixes
// are the ledger's own spelling of "done".
//
// The table is read by its header rather than by position: the ledger's
// writer owns the column order, so a reordered column must not move the
// reader, and a later table in the same file (a second `Verdict | … |
// Outcome` header) re-binds which columns those words mean from that
// point on. So every `|` row is checked for being a header before it is
// read as data; a file with no such header at all has nothing this fact
// can count, and reads as 0 rather than absent — a ledger that never grew
// a verdict table has no open findings in it.
func openFindings(raw []byte, label string) int {
	label = strings.ToUpper(strings.TrimSpace(label))
	verdictCol, outcomeCol := -1, -1
	haveHeader := false
	n := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(t, "|"), "|")
		if len(cells) == 0 {
			continue
		}
		norm := make([]string, len(cells))
		for i, c := range cells {
			norm[i] = strings.ToLower(strings.Trim(strings.TrimSpace(c), "*`"))
		}
		if vi, oi := indexOf(norm, "verdict"), indexOf(norm, "outcome"); vi >= 0 && oi >= 0 {
			verdictCol, outcomeCol = vi, oi
			haveHeader = true
			continue
		}
		if !haveHeader {
			continue
		}
		first := strings.Trim(strings.TrimSpace(cells[0]), "*`")
		if strings.Trim(first, "-: ") == "" {
			continue // the --- separator row
		}
		if len(cells) <= verdictCol || len(cells) <= outcomeCol {
			continue
		}
		verdict := strings.ToUpper(strings.Trim(strings.TrimSpace(cells[verdictCol]), "*`_"))
		if !strings.HasPrefix(verdict, label) {
			continue
		}
		outcome := strings.ToLower(strings.Trim(strings.TrimSpace(cells[outcomeCol]), "*`_"))
		if strings.HasPrefix(outcome, "fixed:") || strings.HasPrefix(outcome, "held:") {
			continue
		}
		n++
	}
	return n
}

// indexOf returns the index of the first element equal to want, or -1.
func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}

// decisionCut are the marks that close a Status line's leading phrase —
// never `→`, because `needs author decision → RESOLVED (…)` is a
// rewritten, CLOSED line and the arrow is what carries that rewrite; a
// cut there would read the closed line as still open.
var decisionCut = []string{"(", ".", ";", "—"}

// emphasisMarks are the markdown emphasis/code delimiters a field's key
// or value may be wrapped in — bold, italic, double- and single-score,
// and backtick — in the order fieldValue tries stripping a closing one.
var emphasisMarks = []string{"**", "__", "*", "_", "`"}

// fieldValue reads a `key: value` field from a line already reduced by
// stripLead, treating the key as emphasis-insensitive: a bare key, one
// wrapped in matching bold, italic, double- or single-score, or
// backticks — `Status:`, `**Status**:`, `**Status:**`, and so on — are
// all the same key. The corpus wraps a labelled field in whichever of
// these an author's editor produced; the field's identity is the word,
// not the markup around it.
//
// ok is false when bare does not open with the key at all — the caller
// decides what a mid-line key (one that sits after other text) means,
// because that is a different, uncountable case, not a fieldValue one.
func fieldValue(bare, key string) (string, bool) {
	if len(bare) < len(key) || !strings.EqualFold(bare[:len(key)], key) {
		return "", false
	}
	rest := bare[len(key):]
	for _, mark := range emphasisMarks {
		if strings.HasPrefix(rest, mark) {
			rest = rest[len(mark):]
			break
		}
	}
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	return strings.Trim(rest[1:], "* `"), true
}

// midlineStatus matches a `Status:` key (any of the emphasis wrappings
// fieldValue accepts) that sits after a CLOSED prior field on the same
// line — preceded by a sentence-ending mark (`.`, `;`, `,`, `)`, itself
// optionally followed by whitespace) — so a shared line like `**Type**:
// SPEC-DEFECT. **Status**: needs author decision.` is recognized even
// though the line does not OPEN with the key. Bare whitespace alone is
// deliberately not a boundary: `Original status:` is a different field
// (a history note) whose name merely ends in the same word, one space
// away, not a Status key sitting after other content.
var midlineStatus = regexp.MustCompile(`(?i)([.;,)]\s*|^)(\*\*|\*|__|_|` + "`" + `)?status(\*\*|\*|__|_|` + "`" + `)?\s*:`)

// openDecisions counts deviations.md's `Status:` lines whose leading
// phrase still reads the open label, case-insensitively, and reports
// whether any line carries a Status key mid-line — after other text,
// where the reader cannot trust a count.
//
// A line carrying `→` anywhere is a rewritten line — the launch prompt
// closes an entry only by rewriting it to `… → RESOLVED (…)` — so it is
// closed whatever sits before the arrow. Otherwise the phrase is open
// when it IS the label or starts with the label and a space: `needs
// author decision on ordering (…)` is a qualified open line, not a
// closed one.
func openDecisions(raw []byte, label string) (n int, midline bool) {
	want := strings.ToLower(label)
	for _, ln := range strings.Split(string(raw), "\n") {
		bare := stripLead(ln)
		val, ok := fieldValue(bare, "status")
		if !ok {
			if midlineStatus.MatchString(bare) {
				midline = true
			}
			continue
		}
		if strings.Contains(val, "→") {
			continue
		}
		cut := len(val)
		for _, mark := range decisionCut {
			if i := strings.Index(val, mark); i >= 0 && i < cut {
				cut = i
			}
		}
		phrase := strings.ToLower(strings.TrimSpace(val[:cut]))
		if phrase == want || strings.HasPrefix(phrase, want+" ") {
			n++
		}
	}
	return n, midline
}

// labelledLine reports whether any line's stripped lead starts with
// label, case-insensitively — coverage.md's `REQ-MVV output` heading or
// bullet, never its `REQ-MVV runner` sibling, which does not share the
// prefix.
func labelledLine(raw []byte, label string) bool {
	for _, ln := range strings.Split(string(raw), "\n") {
		bare := stripLead(ln)
		if len(bare) < len(label) {
			continue
		}
		if strings.EqualFold(bare[:len(label)], label) {
			return true
		}
	}
	return false
}

// verificationFinding matches a Phase 3 finding id at the head of a
// stripped line — `### FAIL-1 — …`, `- **ADV-1** — x` and a bare
// `FAIL-2 …` all reduce to the id. stripLead does not remove the
// CLOSING `**`, so this anchors on the id prefix only.
var verificationFinding = regexp.MustCompile(`^(FAIL|ADV)-[0-9]+`)

// verificationRun reports whether verification.md is the record of a run
// rather than a stub: either Phase 3 found something (a FAIL-N/ADV-N id)
// or it found nothing and said so (`## Verdict — clean`). A file with
// headings and neither is a template nobody filled in, which must not
// read as a pass.
func verificationRun(raw []byte) bool {
	for _, ln := range strings.Split(string(raw), "\n") {
		if verificationFinding.MatchString(stripLead(ln)) {
			return true
		}
	}
	return labelledLine(raw, "Verdict")
}

// headerField reads a labelled header line from a file under a root.
//
// The lens evidence files open with a small stamp block — `model:` on
// every lens element file (rdr-common §model-stamp), `variant:` on a
// repeatability run (§repeatability-variant) — written at generation
// precisely so the answer survives the session that produced it. The
// projector does not read these files: they are evidence, not records,
// so a fact is the only way the routing can see them.
//
// It is a STRICT LABEL, unlike capsuleState's loose scan, and the
// difference is the corpus rather than taste. A capsule states its word
// three ways (a bare `COMPLETE`, `# Status - COMPLETE`, and a labelled
// `state:` inside a pipe-delimited line), so that reader must search.
// These stamps are written by a prompt to a fixed form, so this one
// matches `<label> <value>` at the head of a line and takes the LEADING
// WORD of what follows — which is what makes `full (escalated: …)` and
// `claude-opus-5[1m]   (fresh-context run A; …)` answer `full` and
// `claude-opus-5[1m]` rather than carrying a sentence into a tag.
//
// ABSENT, never false, when the file is missing or carries no such
// label. A missing stamp is `model-unknown` by §model-stamp's own rule —
// "no stamp = unknown, so don't assume a match" — and false would say
// the opposite. The completion PROBES already answer whether the file
// exists; this answers only what it says.
//
// The read is bounded by headerLines for capsuleState's reason: a word
// appearing in a later narrative paragraph must not answer for a header
// the author wrote at the top.
func (e *FactEnv) headerField(d FactDecl) (Fact, bool) {
	full, ok := e.under(d.Root, d.Path)
	if !ok {
		return Fact{}, false
	}
	raw, err := e.readFile(full)
	if err != nil {
		return Fact{}, false
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) > headerLines {
		lines = lines[:headerLines]
	}
	for _, ln := range lines {
		bare := strings.TrimSpace(strings.NewReplacer("**", "", "*", "", "_", "", "`", "", "- ", "").Replace(ln))
		if len(bare) < len(d.Label) || !strings.EqualFold(bare[:len(d.Label)], d.Label) {
			continue
		}
		v := headerValue(strings.TrimSpace(bare[len(d.Label):]))
		if v == "" {
			return Fact{}, false
		}
		// An enum answers only inside its declared domain. A stamp the
		// domain does not name is not a new member to be exported — the
		// table's own rule is that an evaluator producing a value off
		// its domain is a bug — so it reads as absent, which is the
		// same answer a missing stamp gives and the honest one.
		if d.Kind == "enum" && len(d.Domain) > 0 && !slices.Contains(d.Domain, v) {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	return Fact{}, false
}

// modelCompare answers whether critique's two passes were written by
// different base models.
//
// It reads the two stamps rather than exporting them, because the ids
// are an OPEN vocabulary — `claude-opus-5[1m]`, `glm-5.2:cloud`,
// `Claude Sonnet 5` are all in the corpus — and an enum cannot declare a
// domain it does not know. The QUESTION the flow asks is closed over any
// pair of ids, so that is what is published: a routing table can
// enumerate four outcomes, and could never enumerate the model roster.
//
// The four are all real states, measured (24 differ, 8 same, 10
// unknown). `same` is the recorded single-model fallback, which the
// critique lens explicitly sanctions — "do not let the absence of one
// skip the anti-sycophancy step" — so it is a completion story, not a
// defect. `unknown` is §model-stamp's own rule turned into a value: a
// missing stamp must not read as a match.
//
// Comparison is on the LEADING WORD of each stamp, which headerField
// already took, so a trailing note like `(pass A — single-model
// fallback, fresh context)` does not make two identical models differ.
func (e *FactEnv) modelCompare(d FactDecl) (Fact, bool) {
	if _, ok := e.Roots[d.Root]; !ok {
		return Fact{}, false
	}
	read := func(path string) (string, bool) {
		f, ok := e.headerField(FactDecl{
			Kind: "scalar", Source: "header-field",
			Root: d.Root, Path: path, Label: d.Label})
		return f.Value, ok
	}
	a, aok := read(d.Paths[0])
	b, bok := read(d.Paths[1])
	switch {
	case !bok && !e.exists(d.Root, d.Paths[1]):
		// No second pass on disk at all. Distinct from a second pass
		// that ran and left no stamp, which is `unknown`.
		return Fact{Name: d.Name, Kind: d.Kind, Value: "absent"}, true
	case !aok || !bok:
		return Fact{Name: d.Name, Kind: d.Kind, Value: "unknown"}, true
	case strings.EqualFold(a, b):
		return Fact{Name: d.Name, Kind: d.Kind, Value: "same"}, true
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "differ"}, true
}

// exists reports whether a path under a bound root is on disk, for the
// one case that must tell "no second pass" from "a second pass with no
// stamp": both leave headerField absent, and they are different states.
func (e *FactEnv) exists(root, path string) bool {
	full, ok := e.under(root, path)
	if !ok {
		return false
	}
	_, err := e.statPath(full)
	return err == nil
}

// readmeRow answers what the records dir's index table says about THIS
// record: whether it has a row at all, and the Status that row carries.
//
// It is the one write-side read that no other source can give. A probe
// names a path and a field reads this record's own projection, but the
// index row lives in a SIBLING document — `$RDR_RECORDS/README.md` — and
// is keyed by the record number rather than by a path. `index --readme`
// already computes the whole table's drift; this is the per-record slice
// of the same read, so the two cannot disagree about what a row says.
//
// Three-valued like every other fact, and the distinction is the point:
//
//	absent  the root is unbound, or there is no README, or it holds no
//	        index table — nothing looked
//	"none"  the table was read and this record has NO row (a pre-seed
//	        record, or one seed never indexed)
//	<status> the row's Status cell, verbatim
//
// "looked and found no row" is a real answer a write op must act on —
// `readme --add` is exactly the op for it, and `--flip` must refuse
// there rather than inventing a row — so it is a declared value and not
// an absence. An unreadable README is the opposite: nothing looked, so
// nothing is claimed.
//
// `d.Label` selects which cell is reported, defaulting to the Status.
// The row's own presence is reported as the `none` member of the
// declared domain, which is why the domain must name it.
func (e *FactEnv) readmeRow(d FactDecl) (Fact, bool) {
	full, ok := e.under(d.Root, d.Path)
	if !ok {
		return Fact{}, false
	}
	raw, err := e.readFile(full)
	if err != nil {
		return Fact{}, false
	}
	rows := scan.ParseReadmeIndex(strings.Split(string(raw), "\n"))
	if len(rows) == 0 {
		// No index table is "nothing looked", not "no row": a README that
		// is prose-only says nothing about whether this record is indexed.
		return Fact{}, false
	}
	// The record number is the key the table is written on, and a row is
	// matched on it exactly — never on the title, which drifts.
	want := e.Doc.Record
	for _, r := range rows {
		if r.Record != want {
			continue
		}
		v := strings.TrimSpace(r.Status)
		if d.Label == "title" {
			v = strings.TrimSpace(r.Title)
		} else if d.Label == "priority" {
			v = strings.TrimSpace(r.Priority)
		}
		if v == "" {
			// A row whose cell is empty is a malformed row, not a missing
			// one. Reporting "none" would send a caller to `--add` over a
			// row that already exists; absent says the table cannot answer.
			return Fact{}, false
		}
		if d.Kind == "enum" && len(d.Domain) > 0 && !slices.Contains(d.Domain, v) {
			return Fact{}, false
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "none"}, true
}

// sectionProse reports whether a section carries text the AUTHOR wrote.
//
// It exists for the Normative Contracts zero. `counts.elements.C` counts
// LABELLED contracts, and a zero there has two very different meanings:
// the section is empty, or it holds contracts written as prose, which are
// unaddressable but real. The Stage-5 Determinacy trigger reads the
// section either way, so the skills were told to go and read it — the
// override this fact retires.
//
// Authored means: a non-blank line that is not a heading, not the
// template's own guidance block, and not a draft placeholder. Those three
// exclusions are what makes the answer honest — a section holding only
// `[Load-bearing — implementers must match exactly. …]` and a
// `_Draft placeholder._` is EMPTY, however many lines it spans. Measured
// on the corpus: that is the difference between cli/0069 (52 lines, all
// template) and cli/0053 (27 lines, real prose contracts).
//
// The guidance block is recognised by the same AuthoringMarker the lint's
// placeholder rule uses, so the two agree by construction rather than by
// two lists kept in step.
func (e *FactEnv) sectionProse(d FactDecl) (Fact, bool) {
	if e.Doc == nil || e.Doc.Path == "" {
		return Fact{}, false
	}
	var start, end int
	for _, n := range e.Doc.Outline {
		if n.Canonical == d.Path {
			start, end = n.LineStart, n.LineEnd
			break
		}
	}
	if start == 0 {
		return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
	}
	raw, err := e.readFile(e.Doc.Path)
	if err != nil {
		return Fact{}, false
	}
	lines := strings.Split(string(raw), "\n")
	if end > len(lines) {
		end = len(lines)
	}
	inMarker, inPlaceholder := false, false
	for i := start; i < end && i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case t == "":
			continue
		case inMarker:
			if strings.Contains(t, "]") {
				inMarker = false
			}
			continue
		case inPlaceholder:
			// The stand-in is one italic run; it ends where the emphasis
			// closes, however many lines the author wrapped it over.
			if strings.HasSuffix(t, "_") {
				inPlaceholder = false
			}
			continue
		case model.AuthoringMarker.MatchString(t):
			// A guidance block runs to its closing bracket, over as many
			// lines as the template wraps it across.
			inMarker = !strings.Contains(t, "]")
			continue
		case model.FenceDelimiter.MatchString(t):
			// A fence's delimiter is structure. Its CONTENTS are content
			// and are judged on their own lines; the ``` itself says
			// nothing about who wrote them.
			continue
		case len(t) < proseFloor:
			// A bare `**C1**` label, an `e.g.:`. Too short to carry a
			// claim, and each appears in the template and in authored
			// sections alike — so neither this nor TemplateLine can say
			// who wrote it, and a section made only of these has no
			// content either way.
			continue
		case strings.HasPrefix(t, "#"):
			continue
		case model.TemplateLine(t):
			// The template writes this line itself, so a record carrying
			// it has copied rather than authored — the same reading the
			// guidance-block rule applies, generalised past the bracket.
			continue
		case isDraftPlaceholder(t):
			inPlaceholder = !strings.HasSuffix(t, "_")
			continue
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: "true"}, true
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
}

// proseFloor is the shortest line that can carry an authored claim. It
// matches the template comparison's own floor for the same reason: below
// it a line is punctuation, a fence or a label, and it is written
// identically whether or not the section was authored.
const proseFloor = 12

// isDraftPlaceholder recognises the seeded stand-in a stage writes before
// a section is authored — `_Draft placeholder._`, and the longer forms
// that name the stage owing it.
func isDraftPlaceholder(t string) bool {
	l := strings.ToLower(strings.Trim(t, "_*> "))
	return strings.HasPrefix(l, "draft placeholder")
}

// headerValue takes a stamp's value: everything up to the first
// whitespace.
//
// It is NOT leadingWord, and the difference is the point. leadingWord
// breaks on `-` and `:` too, which is right for the qualifier grammars
// it serves and wrong for every id written here: `claude-opus-5` would
// become `claude` and `glm-5.2:cloud` would become `glm-5.2`, so two
// different models would compare equal. A stamp's value is one
// whitespace-delimited token — `full`, `lite`, `claude-opus-5[1m]` —
// and the prose that may follow it (`(escalated: …)`, `(pass A —
// single-model fallback)`) is a note to a human, never part of the id.
func headerValue(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

// headerLines bounds a header-field read, for capsuleHeaderLines'
// reason. The stamps sit in the first lines of an evidence file: a
// `model:`/`variant:` pair, sometimes under a title or a comment, and
// the corpus's deepest is line 3.
const headerLines = 12

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
	full, ok := e.under(d.Root, d.Path)
	if !ok {
		return Fact{}, false
	}
	number := ident.RecordOf(e.Slug)
	if number == "" || e.readDir == nil {
		return Fact{}, false
	}
	entries, err := e.readDir(full)
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
	full, ok := e.under(d.Root, d.Path)
	if !ok {
		return Fact{}, false
	}
	number := ident.RecordOf(e.Slug)
	if number == "" || e.readDir == nil {
		return Fact{}, false
	}
	entries, err := e.readDir(full)
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

// --- freshness: is the evidence older than the re-entry? -------------------
//
// A lens folder says the lens RAN and its product file says it FINISHED,
// and on a first pass those two answer the row. On a RE-ENTERED record
// they do not: a `Draft [revised from Final <date>; …]` keeps every
// folder its Final earned, so the folder-and-stamp reading answers
// "row complete → /rdr-reconcile" over a battery that never saw the
// rework. Measured on a live run: three prose overrides in one pass,
// each re-litigating "this evidence predates the demote" by hand.
//
// The reference point is the DEMOTE DATE the qualifier already carries.
// It is content, so it survives a fresh checkout, and it is the one date
// that means "the record changed after this" by construction. Only a
// re-entered record has one, which is what makes a Final or a first-pass
// Draft incapable of reading stale: with no demote date there is
// nothing to be older than, and both facts answer their fresh value.
//
// The evidence side is content-first and mtime last. Each file under
// the lens is dated by a `Date:` stamp in its header where one is
// written (gate.md, reconcile and tooling-pass reports carry one; the
// lens element files carry a `Model:` stamp and, mostly, no date), and
// by its modification time otherwise. The mtime fallback is the one
// non-deterministic read in this table and it is deliberate: a checkout
// resets mtimes to NOW, which reads as FRESH — the answer this fact gave
// before it existed — so the failure mode of the fallback is the status
// quo, never a spurious re-verify. A lens is dated by its NEWEST file,
// loose or under any iter-N, so a re-verify pass that wrote iter-3 today
// makes the lens current however old iter-1 is.
//
// Same-day is fresh. A demote and its re-verify routinely land on one
// date, and a strict "older than" is what lets the re-run count.

// demoteDate is the `revised from Final YYYY-MM-DD` date off the Status
// qualifier, or "" for any record that is not a scoped re-entry.
//
// This is the one place a fact re-reads a qualifier the projector
// already classified, and it does so with the projector's OWN grammar:
// the date is not a projected field yet, and `RevisedFromGrammar` is
// exported precisely so nothing spells the form twice. The form check
// comes first, so a bracketed near-miss (which lint already names)
// stays "not a re-entry" here as it does everywhere else.
func (e *FactEnv) demoteDate() string {
	if e.Doc == nil {
		return ""
	}
	f := metadataField(e.Doc, "Status")
	if f == nil || f.Status == nil || f.Status.Form != model.QualifierRevisedFrom.String() {
		return ""
	}
	m := model.RevisedFromGrammar.FindStringSubmatch(strings.TrimSpace(f.Status.Qualifier))
	if m == nil {
		return ""
	}
	return m[1]
}

// staleLens names the first declared lens, in the table's order, whose
// newest evidence predates the demote date; `none` otherwise.
//
// The order the table lists the paths in is the row order, and the
// answer is the FIRST stale one because that is how the lens group
// walks a row — one lens per answer, the next on the next call. Absent
// when the evidence root is unbound, like any probe: "no evidence root"
// is not "nothing is stale".
func (e *FactEnv) staleLens(d FactDecl) (Fact, bool) {
	if _, ok := e.Roots[d.Root]; !ok {
		return Fact{}, false
	}
	demoted := e.demoteDate()
	if demoted == "" {
		return Fact{Name: d.Name, Kind: d.Kind, Value: "none"}, true
	}
	for _, p := range d.Paths {
		full, _ := e.under(d.Root, p)
		newest := e.newestDate(full)
		if newest != "" && newest < demoted {
			return Fact{Name: d.Name, Kind: d.Kind, Value: p}, true
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: "none"}, true
}

// stalePath answers whether one file predates the demote date. A
// missing file is not stale — the existence probe beside it says
// "unwritten", and this must not say "old" about nothing.
func (e *FactEnv) stalePath(d FactDecl) (Fact, bool) {
	full, ok := e.under(d.Root, d.Path)
	if !ok {
		return Fact{}, false
	}
	demoted := e.demoteDate()
	if demoted == "" {
		return Fact{Name: d.Name, Kind: d.Kind, Value: "false"}, true
	}
	dated := e.fileDate(full)
	return Fact{Name: d.Name, Kind: d.Kind, Value: boolLiteral(dated != "" && dated < demoted)}, true
}

// newestDate is the latest fileDate under a directory, recursively, or
// "" when there is no file to date. Dotfiles are skipped as nextIteration
// skips them: an editor's swap file is not evidence.
func (e *FactEnv) newestDate(dir string) string {
	if e.readDir == nil {
		return ""
	}
	entries, err := e.readDir(dir)
	if err != nil {
		return ""
	}
	newest := ""
	for _, ent := range entries {
		if strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		full := filepath.Join(dir, ent.Name())
		var d string
		if ent.IsDir() {
			d = e.newestDate(full)
		} else {
			d = e.fileDate(full)
		}
		if d > newest {
			newest = d
		}
	}
	return newest
}

// dateStamp is the YYYY-MM-DD a `Date:` header opens with. Anchored so
// `Date: 2026-08-22. Verdict: …` yields the date and nothing after it.
var dateStamp = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)

// fileDate dates one file: its `Date:` header if it writes one, else its
// mtime as a calendar day in local time. "" when the file is unreadable.
func (e *FactEnv) fileDate(path string) string {
	raw, err := e.readFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) > headerLines {
		lines = lines[:headerLines]
	}
	for _, ln := range lines {
		bare := strings.TrimSpace(strings.NewReplacer("**", "", "*", "", "_", "", "`", "", "- ", "").Replace(ln))
		if len(bare) < len("date:") || !strings.EqualFold(bare[:len("date:")], "date:") {
			continue
		}
		if m := dateStamp.FindStringSubmatch(strings.TrimSpace(bare[len("date:"):])); m != nil {
			return m[1]
		}
	}
	if e.statPath == nil {
		return ""
	}
	fi, err := e.statPath(path)
	if err != nil {
		return ""
	}
	return fi.ModTime().Format("2006-01-02")
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

// --- gate facts ------------------------------------------------------------
//
// The questions a stage asks at its exit gate — WHICH assumptions are
// still Pending, WHICH ones a re-entry reopened, how many anchors were
// looked for and not found — used to be answerable only by pulling the
// elements or edges facet (100KB+ on a large record) and walking it by
// hand. The tallies above say how many; these say which, and count the
// edges the tallies never read.

// assumptionIDs names the assumptions in one tally bucket, as the ids the
// projector cites (`NNNN:A3`). The buckets are tallyAssumptions's, read
// through the same switch, so `ca_pending_ids` has `ca_pending` members.
func (e *FactEnv) assumptionIDs(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	var ids []string
	for _, el := range e.Doc.Elements {
		if el.Kind != ident.Assumption {
			continue
		}
		for _, f := range el.Fields {
			if f.Canonical == "Status" && f.Status != nil && assumptionBucket(f.Status) == d.Select {
				ids = append(ids, el.ID)
			}
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(ids)}, true
}

// assumptionBucket is tallyAssumptions's classification, so the ids and
// the counts cannot disagree about where a status falls.
func assumptionBucket(s *scan.Status) string {
	switch {
	case s.Placeholder:
		return "placeholder"
	case observedTerminal[s.Value]:
		return "observed-terminal"
	case s.Tier == model.OffVocabulary.String():
		return "off-vocabulary"
	case s.Value == "Verified" || s.Value == "Pending" || s.Value == "Unverified":
		return s.Value
	}
	return "off-vocabulary"
}

// reverifyIDs is the re-verify set a `Draft [revised from Final …;
// re-verify A2,A4 — …]` qualifier names, read off the projection's
// `reverify` self-edges — the list a scoped Resolve works through.
// ABSENT unless the Status is in that form: a record with no re-entry
// has no re-verify set, which is not an empty one (`re-verify none` is a
// real answer and projects `[]`).
func (e *FactEnv) reverifyIDs(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	f := metadataField(e.Doc, "Status")
	if f == nil || f.Status == nil || f.Status.Form != model.QualifierRevisedFrom.String() {
		return Fact{}, false
	}
	var ids []string
	for _, ed := range e.Doc.Edges {
		if ed.Kind == edge.Reverify {
			ids = append(ids, ed.To)
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(ids)}, true
}

// resolveEdgesOnce asks the caller's hook the first time a fact needs
// the edges decided, and never again: every edge fact shares the one
// resolution, and a call that filters them all out never pays for it.
func (e *FactEnv) resolveEdgesOnce() {
	if e.ResolveEdges != nil && !e.edgesResolved {
		e.edgesResolved = true
		e.ResolveEdges()
	}
}

// jointCheckHome is the lock's reading of the record's `Joint-check:`
// lines, folded worst-first: clear < homed < unhomed < open. A fire is
// homed only when every `|` segment of its `(home: …)` minted a
// joint-decision-home edge (scan.jointCheckEdges) and every one resolved
// TRUE — the propose gate's own rule, "false or absent is not a pass", so
// a home nothing looked for (no corpus bound) reads unhomed, never homed.
// A `re-run` line mints no home and reads unhomed for the same reason.
// `open` is a legal propose-time value the lock refuses. With no line at
// all the fact is absent; `joint_checks` already surfaces that.
func (e *FactEnv) jointCheckHome(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	const (
		clear = iota + 1
		homed
		unhomed
		open
	)
	worst := 0
	for _, el := range e.Doc.Elements {
		if el.Kind != ident.JointCheck || el.Joint == nil {
			continue
		}
		r := unhomed
		switch {
		case el.Joint.Verdict == "clear":
			r = clear
		case el.Joint.Open:
			r = open
		case e.jointHomed(el):
			r = homed
		}
		if r > worst {
			worst = r
		}
	}
	if worst == 0 {
		return Fact{}, false
	}
	v := map[int]string{clear: "clear", homed: "homed", unhomed: "unhomed", open: "open"}[worst]
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// jointHomed decides one fired line: every `|` segment named a reference
// (Joint.Homes, filled by the edge pass — a segment that did not, prose
// naming no authority or a trailing OPEN, is a home nothing can resolve),
// at least one home edge was minted, and every one resolved true. Homes
// is counted rather than the edges because two segments naming one
// section are one edge and two homes.
func (e *FactEnv) jointHomed(el scan.Element) bool {
	if len(el.Joint.Homes) != len(strings.Split(el.Joint.Home, "|")) {
		return false
	}
	e.resolveEdgesOnce()
	n := 0
	for _, ed := range e.Doc.Edges {
		if ed.From != el.ID || ed.Kind != edge.JointDecisionHome {
			continue
		}
		if ed.Resolved == nil || !*ed.Resolved {
			return false
		}
		n++
	}
	return n > 0
}

// overlapUncited re-runs propose's two query arms at lock, scoped to this
// record: the in-flight pairs sharing a source anchor (AnchorIntersect) or
// a contract literal (LiteralIntersect) where neither record cites the
// other — a joint decision nobody fired on, or a propose-time `clear` a
// later peer made stale. On demand: it walks the corpus. Absent with no
// corpus bound, so the table's `unchecked` sentinel carries it.
func (e *FactEnv) overlapUncited(d FactDecl) (Fact, bool) {
	if e.Doc == nil || e.Corpus == nil {
		return Fact{}, false
	}
	docs := e.Corpus()
	if docs == nil {
		return Fact{}, false
	}
	n := 0
	for _, o := range append(scan.AnchorIntersect(docs, true), scan.LiteralIntersect(docs, true)...) {
		if !o.Cited && (o.Records[0] == e.Doc.Record || o.Records[1] == e.Doc.Record) {
			n++
		}
	}
	v := "0"
	if n > 0 {
		v = "1+"
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// edgeTally counts the record's typed edges by verdict — the numbers
// §mechanical-gate used to pull the whole edges facet for.
//
// `resolved` is three-valued and the tallies keep it so: `anchors-total`
// is every source anchor, `anchors-unresolved` those looked for and not
// found, `anchors-unlooked` those nothing looked for (no source root, or
// no corpus to decide against). The three travel together, so unlooked
// never reads as passed. `peer-evidence-unresolved` has no unlooked
// twin, so it goes ABSENT while any peer-evidence edge is undecided
// rather than reporting a zero nothing checked.
//
// Resolution is on demand: the first tally asks the caller's hook, so a
// call that filters these facts out never pays the corpus scan and repo
// walk they cost.
func (e *FactEnv) edgeTally(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	e.resolveEdgesOnce()
	var total, unresolved, unlooked, peerUnresolved int
	peerUnlooked := false
	for _, ed := range e.Doc.Edges {
		switch ed.Kind {
		case edge.SourceAnchor:
			total++
			switch {
			case ed.Resolved == nil:
				unlooked++
			case !*ed.Resolved:
				unresolved++
			}
		case edge.PeerEvidence:
			switch {
			case ed.Resolved == nil:
				peerUnlooked = true
			case !*ed.Resolved:
				peerUnresolved++
			}
		}
	}
	var n int
	switch d.Select {
	case "anchors-total":
		n = total
	case "anchors-unresolved":
		n = unresolved
	case "anchors-unlooked":
		n = unlooked
	case "peer-evidence-unresolved":
		if peerUnlooked {
			return Fact{}, false
		}
		n = peerUnresolved
	default:
		return Fact{}, false
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(n)}, true
}

// spikeDiff is the spikes the record NAMES minus the ones on disk: the
// first path segment after `{SPIKE_DIR}/` in each artifact edge, less
// the entries of the root's `spikes/` dir, matched by name with or
// without an extension (`warmup-order/` and `warmup-order.md` both count
// as a run of `warmup-order`). Stage 6 built this set by hand from the
// record and a directory listing; a set fact is the same subtraction,
// executed.
//
// Three-valued like every rooted fact: absent when the root is unbound.
// A missing spikes dir is NOT absent — the root was looked at and holds
// no runs, so every named spike is unrun — and a record naming no spike
// answers `[]` whatever the dir holds.
func (e *FactEnv) spikeDiff(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	dir, ok := e.under(d.Root, "spikes")
	if !ok {
		return Fact{}, false
	}
	const prefix = "{SPIKE_DIR}/"
	var named []string
	for _, ed := range e.Doc.Edges {
		if ed.Kind != edge.Artifact || !strings.HasPrefix(ed.To, prefix) {
			continue
		}
		rest := strings.TrimPrefix(ed.To, prefix)
		if i := strings.Index(rest, "/"); i >= 0 {
			rest = rest[:i]
		}
		if rest != "" {
			named = append(named, rest)
		}
	}
	run := map[string]bool{}
	if entries, err := e.readDir(dir); err == nil {
		for _, ent := range entries {
			n := ent.Name()
			if strings.HasPrefix(n, ".") {
				continue
			}
			run[n] = true
			run[strings.TrimSuffix(n, filepath.Ext(n))] = true
		}
	}
	var unrun []string
	for _, n := range named {
		if !run[n] {
			unrun = append(unrun, n)
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(unrun)}, true
}

// --- the rollups: facts about a record's peers -----------------------------
//
// Each reads another record through FactEnv.Peer, and each folds a set
// the flow used to fold by hand — Stage 8's "every predecessor COMPLETE",
// the Stage-2 tandem barrier's "every sibling proposed", Finalize's "a
// related Final is still unimplemented". The fold is here, once, and the
// routing table reads a word.

// predecessorStates are the values `state` can emit, checked against the
// declared domain at load. `none` is the declared sentinel for a record
// with no Predecessors field; the evaluator itself goes absent there.
var predecessorStates = []string{"complete", "incomplete", "retired", "unresolved"}

var clusterProposedMembers = []string{"all", "some", "none"}

var relatedRollupMembers = []string{"0", "1+"}

// capsuleStateOf evaluates the table's capsule-state fact over an env —
// the peer's own `impl_state`, through the declaration this record's
// uses, so the two cannot read the capsule differently. Empty when no
// such fact is declared or the capsule is absent.
func capsuleStateOf(t *FactTable, env *FactEnv) string {
	if t == nil || env == nil {
		return ""
	}
	for _, d := range t.Facts {
		if d.Source != "capsule-state" {
			continue
		}
		if f, ok := env.capsuleState(d); ok {
			return f.Value
		}
		return ""
	}
	return ""
}

// predecessorRollup folds the Predecessors field over the peers'
// capsules.
//
//	incomplete  the predecessors not COMPLETE — an unresolvable record
//	            and an absent capsule are members, because neither is
//	            "looked and COMPLETE"
//	state       unresolved when any predecessor did not resolve (the
//	            launch precheck halts on that first), else retired when
//	            every not-COMPLETE one is terminal, else incomplete
//	            when any is not COMPLETE, else complete
//
// A RETIRED predecessor is one the flow has closed without implementing:
// Superseded, Abandoned, Rejected. Its capsule is absent and always will
// be, so asking it to read COMPLETE asks the record to be built in order
// to be replaced — cli/0107 declares the record that superseded it as a
// predecessor, and the gate demanded the superseded one be implemented
// first. That is unsatisfiable by construction, and folding it into
// `incomplete` told a caller "implement those first" about a record that
// never can be.
//
// It is a distinct state rather than a silent pass because the two are
// different questions for a human: `incomplete` names work to do, and
// `retired` names a lineage to confirm. The launch table routes it; this
// only refuses to call them the same thing. Demoted and Deferred are NOT
// retired — a Demoted record re-enters the flow and a Deferred one is
// parked, so both may still reach COMPLETE.
//
// Absent when the record declares no Predecessors: `none` is the
// declared sentinel, not a state the evaluator invents.
func (e *FactEnv) predecessorRollup(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	f := metadataField(e.Doc, "Predecessors")
	if f == nil {
		return Fact{}, false
	}
	preds := recordNumbers(f.Value)
	if len(preds) == 0 {
		return Fact{}, false
	}
	var incomplete, retired []string
	unresolved := false
	for _, p := range preds {
		var peer *FactEnv
		ok := false
		if e.Peer != nil {
			peer, ok = e.Peer(p)
		}
		if !ok {
			unresolved = true
			incomplete = append(incomplete, p)
			continue
		}
		if capsuleStateOf(e.table, peer) == "COMPLETE" {
			continue
		}
		if retiredStatus(peer) {
			retired = append(retired, p)
			continue
		}
		incomplete = append(incomplete, p)
	}
	switch d.Select {
	case "incomplete":
		return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(incomplete)}, true
	case "retired":
		return Fact{Name: d.Name, Kind: d.Kind, Members: canonicalSet(retired)}, true
	case "state":
		// Order is the launch prompt's: a dangling name first (nothing was
		// looked at), then real work owed, then a lineage to confirm. A
		// record with both an unbuilt and a retired predecessor is
		// `incomplete` — the work outranks the confirmation.
		state := "complete"
		switch {
		case unresolved:
			state = "unresolved"
		case len(incomplete) > 0:
			state = "incomplete"
		case len(retired) > 0:
			state = "retired"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: state}, true
	}
	return Fact{}, false
}

// retiredStatus reports whether a record is closed without implementing:
// its capsule is absent and no future run will write one.
//
// Only the three the flow never returns from. `Demoted` re-enters at a
// named stage and `Deferred` is parked, so both may still reach COMPLETE
// and a predecessor in either is honestly `incomplete`.
func retiredStatus(env *FactEnv) bool {
	if env == nil || env.Doc == nil {
		return false
	}
	f := metadataField(env.Doc, "Status")
	if f == nil || f.Status == nil {
		return false
	}
	switch f.Status.Value {
	case "Superseded", "Abandoned", "Rejected":
		return true
	}
	return false
}

// clusterProposed folds the Cluster field over the siblings' plans:
// `all` when every sibling's section (d.Path, the Implementation Plan)
// carries authored text by sectionProse's rule, `some`, or `none`. A
// sibling that does not resolve reads not-proposed, never proposed.
// Absent when the record declares no Cluster: `solo` is the sentinel.
func (e *FactEnv) clusterProposed(d FactDecl) (Fact, bool) {
	if e.Doc == nil {
		return Fact{}, false
	}
	f := metadataField(e.Doc, "Cluster")
	if f == nil {
		return Fact{}, false
	}
	var siblings []string
	for _, m := range recordNumbers(f.Value) {
		if m != e.Doc.Record {
			siblings = append(siblings, m)
		}
	}
	if len(siblings) == 0 {
		return Fact{}, false
	}
	proposed := 0
	for _, m := range siblings {
		if e.Peer == nil {
			break
		}
		peer, ok := e.Peer(m)
		if !ok {
			continue
		}
		if p, ok := peer.sectionProse(d); ok && p.Value == "true" {
			proposed++
		}
	}
	v := "some"
	switch proposed {
	case 0:
		v = "none"
	case len(siblings):
		v = "all"
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// relatedRollup counts the record's asserted cluster members (ClusterOf,
// candidates excluded), by `select`:
//
//	final-unimplemented  Final and not COMPLETE — Finalize's "reconcile
//	                      before implement" question
//	draft                still Status Draft — a 7.1 pass run now would
//	                      cover a partial set
//
// On demand: it walks the corpus. Absent with no corpus bound.
func (e *FactEnv) relatedRollup(d FactDecl) (Fact, bool) {
	if e.Doc == nil || e.Corpus == nil {
		return Fact{}, false
	}
	docs := e.Corpus()
	if docs == nil {
		return Fact{}, false
	}
	n := 0
	for _, m := range scan.ClusterOf(docs, e.Doc.Record) {
		if m.Record == e.Doc.Record || m.Candidate {
			continue
		}
		switch d.Select {
		case "draft":
			if m.Status == "Draft" {
				n++
			}
		default: // "final-unimplemented", and "" for a table predating the key
			if m.Status != "Final" {
				continue
			}
			state := ""
			if e.Peer != nil {
				if peer, ok := e.Peer(m.Record); ok {
					state = capsuleStateOf(e.table, peer)
				}
			}
			if state != "COMPLETE" {
				n++
			}
		}
	}
	v := "0"
	if n > 0 {
		v = "1+"
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
}

// --- the ledger readers -----------------------------------------------------
//
// Two facts about a written ledger: how many of its rows are still open
// (ledgerTally), and the deepest iter-N pass any of a set of trees
// reached (iterMax). Both read the CURRENT file only — the one under the
// highest iter-N segment, or the loose one — never merge history across
// passes, for the same reason a probe names one path: a signal that
// silently folds an old pass into a new one is wrong in the direction
// nothing catches.

// iterOf is e.table.Iter, safe over a nil table — a test env built by
// hand rather than NewFactEnv. A nil Iter's empty Segment makes
// iterSegmentRE return nil, which both readers treat as "no segments to
// find", the same answer a table declaring no [iteration] gives.
func (e *FactEnv) iterOf() Iteration {
	if e.table == nil {
		return Iteration{}
	}
	return e.table.Iter
}

// currentLedgerFile finds the ledger `name` under `dir`: the file under
// the highest iter-N segment that contains it, else the loose file
// directly under dir. "" when neither exists. `dir` may be "" (the root
// itself, ledger-tally's no-`paths` case).
func (e *FactEnv) currentLedgerFile(base, dir, name string, it Iteration) string {
	root := base
	if dir != "" {
		root = filepath.Join(base, dir)
	}
	re := iterSegmentRE(it.Segment)
	best, bestN := "", -1
	if re != nil && e.readDir != nil {
		if entries, err := e.readDir(root); err == nil {
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				m := re.FindStringSubmatch(ent.Name())
				if m == nil {
					continue
				}
				n, err := strconv.Atoi(m[1])
				if err != nil {
					continue
				}
				candidate := filepath.Join(root, ent.Name(), name)
				if _, err := e.statPath(candidate); err != nil {
					continue
				}
				if n > bestN {
					bestN, best = n, candidate
				}
			}
		}
	}
	if best != "" {
		return best
	}
	loose := filepath.Join(root, name)
	if _, err := e.statPath(loose); err == nil {
		return loose
	}
	return ""
}

// ledgerTally counts a ledger's entries and its open ones, summed over
// `paths` (or the single root itself when `paths` is absent), reading
// only the CURRENT file per dir. A bound root with no ledger anywhere
// answers 0 — looked, none — not absent; only an unbound root is absent.
func (e *FactEnv) ledgerTally(d FactDecl) (Fact, bool) {
	base, ok := e.Roots[d.Root]
	if !ok {
		return Fact{}, false
	}
	entryRE := regexp.MustCompile(d.Entry)
	var openRE *regexp.Regexp
	if d.Open != "" {
		openRE = regexp.MustCompile(d.Open)
	} else {
		openRE = regexp.MustCompile(d.Closed)
	}
	dirs := d.Paths
	if len(dirs) == 0 {
		dirs = []string{""}
	}
	total, open := 0, 0
	for _, dir := range dirs {
		full := e.currentLedgerFile(base, dir, d.Path, e.iterOf())
		if full == "" {
			continue
		}
		raw, err := e.readFile(full)
		if err != nil {
			continue
		}
		for _, ln := range strings.Split(string(raw), "\n") {
			if !entryRE.MatchString(ln) {
				continue
			}
			total++
			isOpen := openRE.MatchString(ln)
			if d.Closed != "" {
				isOpen = !isOpen
			}
			if isOpen {
				open++
			}
		}
	}
	if d.Kind == "enum" {
		v := "0"
		if open > 0 {
			v = "1+"
		}
		return Fact{Name: d.Name, Kind: d.Kind, Value: v}, true
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(open)}, true
}

// iterMax is the deepest iter-N segment found under any of `paths`, or 1
// (the loose first pass, [iteration]'s `first = "loose"`) when none of
// them has a segment. Absent when the root is unbound.
func (e *FactEnv) iterMax(d FactDecl) (Fact, bool) {
	base, ok := e.Roots[d.Root]
	if !ok {
		return Fact{}, false
	}
	re := iterSegmentRE(e.iterOf().Segment)
	max := 1
	if re != nil && e.readDir != nil {
		for _, dir := range d.Paths {
			root := filepath.Join(base, dir)
			entries, err := e.readDir(root)
			if err != nil {
				continue
			}
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				m := re.FindStringSubmatch(ent.Name())
				if m == nil {
					continue
				}
				n, err := strconv.Atoi(m[1])
				if err == nil && n > max {
					max = n
				}
			}
		}
	}
	return Fact{Name: d.Name, Kind: d.Kind, Value: strconv.Itoa(max)}, true
}
