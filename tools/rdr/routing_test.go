package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/model"
)

// The routing table (`models/rdr-status.toml`) is the other half of what
// the fact table started: N1 and N2 moved rdr-status's SIGNALS into data,
// and that file moves its ROUTING there. These tests bind the two halves
// the same way `facts_test.go` binds the fact table to the skill's signal
// table — in both directions, so neither file can drift from the other
// without a test going red.
//
// WHY THESE TESTS DO NOT RUN `intrastate`. The coverage proof is that
// binary's job and re-implementing it here would be a second, weaker
// answer to a question already answered — `intrastate lint --model` is
// the acceptance criterion and it runs in the flow, not in `go test`.
// What these tests own is the SEAM: that every tag the model matches on
// is a fact this binary actually renders, that every value it compares
// against is one the fact can actually carry, and that the model stays
// parseable. Those are properties of this repo's data, checkable with no
// second binary installed. `intrastate` is a dependency, not an
// accelerator (rdr-common §intrastate) — the flow stops without it — but
// the seam's own data properties should be checkable without it.

const routingModelName = "rdr-status.toml"

// routingModelNames is every routing model in this repo that matches on
// fact names. Both bind to `rdr-facts.toml` the same way and neither
// binary calls the other, so a check that covered only the first would
// leave the second free to drift — and the write model carries its own
// copy of the status vocabulary plus `readme_status`'s.
var routingModelNames = []string{"rdr-status.toml", "rdr-write.toml", "rdr-cascade.toml", "rdr-launch.toml", "rdr-loop.toml"}

// callerTags names, per model, the observed tags a caller supplies by hand
// (an orchestrator's own packet fields and Ledger, never an `rdr status`
// fact) rather than reading a fact `rdr-facts.toml` declares. The model
// header is the contract for these, not the fact table, so the fact-match
// checks below skip them. `ask_each` is declared now for the sibling
// `posture` group `rdr-cascade.toml` will grow. The write model's six
// are Resolve's two Profile judgements, rdr-status's `emit.floor` passed
// through, the verdict packet's blocker class, and the `ground` group's
// ladder position (`searched`, `found`).
var callerTags = map[string]map[string]bool{
	"rdr-cascade.toml": {"verdict": true, "blocking": true, "retry": true, "action": true, "ask_each": true},
	// The launch run's own observations: source files touched, a suite
	// run over the output cap, context pressure, whether the full suite
	// is green now, the pre-Phase-1 suite baseline, and a Phase 2 leg's
	// commits (git) and elapsed minutes (the clock) since it began. Its
	// seven other tags are facts and stay policed.
	"rdr-launch.toml": {"files": true, "suite": true, "pressure": true, "suite_green": true, "baseline": true, "commits": true, "elapsed": true},
	"rdr-write.toml":  {"user_facing": true, "locks": true, "floor": true, "blocker_class": true, "searched": true, "found": true},
	// The loop caps: every tag is a value the caller holds from a tool
	// call this pass — `rdr paths --next-iter`'s ITER_BUCKET, whether
	// `rdr anchors` and `comm` printed anything, the resolve's fix size,
	// and 7.1's open-entry count. A pass number belongs to one lens or
	// cluster, not the record, so no fact renders it.
	"rdr-loop.toml": {"iter": true, "found": true, "net_new": true, "fix": true, "open": true},
}

// routingModel is the parsed model, reduced to what the seam needs: the
// observed tags it declares and the atoms its rules compare.
type routingModel struct {
	// Tags maps a declared observed tag to its domain, empty for a kind
	// that declares none.
	Tags map[string][]string
	// Kinds maps a declared observed tag to its kind.
	Kinds map[string]string
	// Atoms are every guard comparison, in file order.
	Atoms []routingAtom
	// Outcomes are the declared recognized outcomes.
	Outcomes []string
	// RuleIDs are every rule's id, in file order.
	RuleIDs []string
	// Emits maps a rule id to its emit block.
	Emits map[string]map[string]string
	// EmitDomains maps a declared `[emit.<key>]` to its domain — the
	// union of its partitions, and each partition under `<key>.<part>`.
	EmitDomains map[string][]string
}

// routingAtom is one `[rule.guard.*.<key>]` comparison.
type routingAtom struct {
	Rule     string
	Key      string
	Literals []string
}

// loadRoutingModel parses the shipped model far enough to check the seam.
//
// It does NOT reuse `internal/toml`: that reader
// deliberately refuses `[[array]]` tables, and the model is built of
// them. Widening it to read a file this binary never loads would trade
// away the narrowness that makes it safe for the fact table. So the
// reading here is local, line-oriented, and covers exactly the four
// shapes the model uses — a header, an array-of-tables header, a bare
// scalar, and a bare list. Anything else is a parse error rather than a
// silent skip, because a dropped table would make every check below pass
// vacuously.
func loadRoutingModel(t *testing.T) *routingModel {
	t.Helper()
	return loadRoutingModelNamed(t, routingModelName)
}

// loadRoutingModelNamed is the same parse against any of the routing
// models, so a check can sweep every one of them.
func loadRoutingModelNamed(t *testing.T, name string) *routingModel {
	t.Helper()
	path := repoFile(t, filepath.Join("models", name))
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the shipped routing model is unreadable: %v", err)
	}

	m := &routingModel{
		Tags:        map[string][]string{},
		Kinds:       map[string]string{},
		Emits:       map[string]map[string]string{},
		EmitDomains: map[string][]string{},
	}
	section, rule := "", ""
	pending := map[string]string{} // the open [tags.*] table's keys
	flush := func() {
		if strings.HasPrefix(section, "tags.") && pending["provenance"] == "observed" {
			name := strings.TrimPrefix(section, "tags.")
			m.Tags[name] = splitTOMLList(pending["domain"])
			m.Kinds[name] = pending["kind"]
		}
		pending = map[string]string{}
	}

	for i, raw := range strings.Split(string(src), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[[") {
			flush()
			if !strings.HasSuffix(line, "]]") {
				t.Fatalf("line %d: unterminated array-of-tables header", i+1)
			}
			section = strings.TrimSpace(line[2 : len(line)-2])
			if section != "rule" {
				t.Fatalf("line %d: unexpected array-of-tables %q", i+1, section)
			}
			rule = ""
			continue
		}
		if strings.HasPrefix(line, "[") {
			flush()
			if !strings.HasSuffix(line, "]") {
				t.Fatalf("line %d: unterminated table header", i+1)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("line %d: not a key = value or a header: %q", i+1, line)
		}
		key = strings.TrimSpace(key)
		val := strings.Trim(strings.TrimSpace(rest), `"`)

		switch {
		case section == "" && key == "outcomes":
			m.Outcomes = splitTOMLList(strings.TrimSpace(rest))
		case strings.HasPrefix(section, "tags."):
			pending[key] = strings.TrimSpace(rest)
			if key != "domain" {
				pending[key] = val
			}
		case section == "rule" && key == "id":
			rule = val
			m.RuleIDs = append(m.RuleIDs, rule)
		case strings.HasPrefix(section, "rule.guard."):
			block, gkey, ok := strings.Cut(strings.TrimPrefix(section, "rule.guard."), ".")
			if !ok || (block != "all" && block != "unless") {
				t.Fatalf("line %d: unrecognised guard block %q", i+1, section)
			}
			var lits []string
			switch key {
			case "eq":
				lits = []string{val}
			case "in":
				lits = splitTOMLList(strings.TrimSpace(rest))
			default:
				continue
			}
			m.Atoms = append(m.Atoms, routingAtom{Rule: rule, Key: gkey, Literals: lits})
		case section == "rule.emit":
			if m.Emits[rule] == nil {
				m.Emits[rule] = map[string]string{}
			}
			m.Emits[rule][key] = val
		case strings.HasPrefix(section, "emit."):
			name := strings.TrimPrefix(section, "emit.")
			switch {
			case strings.HasSuffix(name, ".domain"):
				name = strings.TrimSuffix(name, ".domain")
				vals := splitTOMLList(strings.TrimSpace(rest))
				m.EmitDomains[name] = append(m.EmitDomains[name], vals...)
				m.EmitDomains[name+"."+key] = vals
			case key == "domain":
				m.EmitDomains[name] = splitTOMLList(strings.TrimSpace(rest))
			}
		}
	}
	flush()

	if len(m.RuleIDs) == 0 {
		t.Fatal("the routing model declares no rules; the parse moved and this test is blind")
	}
	if len(m.Atoms) == 0 {
		t.Fatal("the routing model declares no guard atoms; without them lint proves nothing")
	}
	return m
}

// splitTOMLList reads a single-line `["a", "b"]` literal. A comma inside
// a quoted member is part of the member — `[emit.row]` spells a lens
// span as one comma-joined word — so the split is on the commas between
// quotes, never on every comma.
func splitTOMLList(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil
	}
	var out []string
	var cur strings.Builder
	quoted := false
	flush := func() {
		if p := strings.Trim(strings.TrimSpace(cur.String()), `"`); p != "" {
			out = append(out, p)
		}
		cur.Reset()
	}
	for _, r := range s[1 : len(s)-1] {
		switch {
		case r == '"':
			quoted = !quoted
			cur.WriteRune(r)
		case r == ',' && !quoted:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// TestRoutingModelMatchesTheFactTable is the seam, checked forward.
//
// Every tag the model declares as observed must be a fact the table
// declares, and every literal a guard atom compares against must be a
// value that fact can actually carry. A model comparing `status` against
// `"Locked"`, or guarding a tag no fact renders, is a rule that can never
// fire — and it would fire nothing SILENTLY, since the resolver has no
// way to know the caller meant a value the corpus never produces.
//
// This is the same claim `TestEveryStageRowIsExpressedAsFacts` makes for
// the signal table, one file further along: the fact names are a contract
// between two repos, and this is the consumer that binds to them.
func TestRoutingModelMatchesTheFactTable(t *testing.T) {
	tbl := loadRealTable(t)
	m := loadRoutingModel(t)

	decl := map[string]FactDecl{}
	for _, f := range tbl.Facts {
		decl[f.Name] = f
	}

	for name := range m.Tags {
		if _, ok := decl[name]; !ok {
			t.Errorf("the model declares observed tag %q, which the fact table does not declare; "+
				"a tag no fact renders can never be supplied, so every rule guarding it is dead", name)
		}
	}

	for _, a := range m.Atoms {
		f, ok := decl[a.Key]
		if !ok {
			t.Errorf("rule %q guards on %q, which the fact table does not declare", a.Rule, a.Key)
			continue
		}
		if _, ok := m.Tags[a.Key]; !ok {
			t.Errorf("rule %q guards on %q, which the model does not declare as an observed tag", a.Rule, a.Key)
		}
		// A prose fact is never rendered as a tag at all, so guarding on
		// one is a rule that cannot fire however the record reads.
		if f.Prose {
			t.Errorf("rule %q guards on %q, which is declared prose and is never rendered as a tag", a.Rule, a.Key)
		}
		for _, lit := range a.Literals {
			if !factCanCarry(f, lit) {
				t.Errorf("rule %q compares %q against %q, which that fact cannot carry (kind %s, domain %v)",
					a.Rule, a.Key, lit, f.Kind, f.Domain)
			}
		}
	}
}

// TestRoutingTagsMatchFactKindAndDomain closes the one direction of the
// seam that nothing watched: a fact whose DOMAIN widens past the routing
// model's copy of it.
//
// The vocabulary is chained from TEMPLATE.md through
// `model.StatusVocabulary()` into the fact table, and
// `TestEnumFactsStayInTheirDomain` holds that link. From the fact table
// to the routing models the chain went dark, and the two failures are
// not symmetric:
//
//   - A model comparing against a value the fact CANNOT carry is already
//     caught — by `TestRoutingModelMatchesTheFactTable` if a rule names
//     it, and by `intrastate lint` if the domain declares it. Both look
//     only at literals some rule mentions.
//   - A fact that GAINS a value no rule claims is caught by neither.
//     `intrastate lint` reads one file and cannot see the fact table;
//     the atom check reads only literals already written down. Adding a
//     status to TEMPLATE.md, then to the fact table, leaves the routing
//     models silently unclaiming a cell — measured, and the whole suite
//     stayed green. The consequence is not a mis-route (the kernel is
//     three-valued and refuses at exit 2) but an unroutable record, found
//     by whoever next runs the navigator instead of at build time.
//
// So the domains are compared as SETS, both ways. Kind is compared too:
// `intrastate lint` rejects a kind/domain pair that is internally
// inconsistent, but a kind that merely disagrees with the fact's is
// internally fine on both sides.
//
// Every routing model is swept, because the write model carries a second
// copy of the same status vocabulary and a `readme_status` domain of its
// own.
func TestRoutingTagsMatchFactKindAndDomain(t *testing.T) {
	tbl := loadRealTable(t)
	decl := map[string]FactDecl{}
	for _, f := range tbl.Facts {
		decl[f.Name] = f
	}

	for _, name := range routingModelNames {
		m := loadRoutingModelNamed(t, name)
		for tag, modelDomain := range m.Tags {
			if callerTags[name][tag] {
				continue
			}
			f, ok := decl[tag]
			if !ok {
				// The name check is TestRoutingModelMatchesTheFactTable's
				// for rdr-status.toml; report it here for the others so a
				// model this test sweeps is never checked vacuously.
				t.Errorf("%s: tag %q names no declared fact", name, tag)
				continue
			}
			if got, want := m.Kinds[tag], f.Kind; got != want {
				t.Errorf("%s: tag %q is kind %q, but fact %q is kind %q; "+
					"the model and the fact table disagree about what the value IS",
					name, tag, got, tag, want)
			}
			// A kind with no declared domain has nothing to compare: a
			// bool's domain is implicit, a set's is a power set, and a
			// scalar has none.
			if f.Kind != "enum" {
				continue
			}
			for _, v := range modelDomain {
				if !containsString(f.Domain, v) {
					t.Errorf("%s: tag %q admits %q, which fact %q cannot carry (fact domain %v); "+
						"the ROUTING MODEL is ahead — either the fact table lost a value or the model invented one",
						name, tag, v, tag, f.Domain)
				}
			}
			for _, v := range f.Domain {
				if !containsString(modelDomain, v) {
					t.Errorf("%s: fact %q can carry %q, which tag %q does not admit (model domain %v); "+
						"the FACT TABLE is ahead — the projector can emit a value no rule claims, "+
						"so a record carrying it is unroutable (flow-guard-unevaluable, exit 2)",
						name, tag, v, tag, modelDomain)
				}
			}
		}
	}
}

// TestRoutingDimensionsAreAlwaysRendered is the absence half of the seam.
//
// A dimension that is merely SOMETIMES supplied cannot be routed on: an
// optional tag withholds the coverage claim at lint and refuses at
// resolve, so a record whose field is simply unwritten would stop the
// navigator rather than route. The fact table's answer is a declared
// `absent` sentinel, and `--tags` then always renders the key.
//
// So: every fact a guard atom reads must either be one that is always
// present, or one that declares a sentinel. This test is what keeps a
// later edit from guarding on a fact that can go missing — the failure
// would otherwise appear only on the one record in the corpus that
// happens to omit that field.
func TestRoutingDimensionsAreAlwaysRendered(t *testing.T) {
	tbl := loadRealTable(t)
	m := loadRoutingModel(t)

	decl := map[string]FactDecl{}
	for _, f := range tbl.Facts {
		decl[f.Name] = f
	}

	// Totality is MEASURED, not assumed from the source kind. Some facts
	// cannot go absent — every record has a Status, and a rollup always
	// answers — while others read a field the template does not require.
	// Which is which is a property of the corpus and the evaluator
	// together, so the check evaluates the real fixtures and asserts
	// that every dimension came out rendered on every one of them.
	//
	// The fixtures are the right subject: they were built to span the
	// shapes the routing cares about, and they include the sparse
	// Draft whose fields are missing. A dimension that survives them all
	// is one `--tags` always emits.
	dims := map[string]bool{}
	for _, a := range m.Atoms {
		dims[a.Key] = true
	}

	_, table := bindStatusFixture(t)
	for _, rec := range routingFixtures {
		rendered := fixtureTags(t, table, rec)
		for key := range dims {
			// An on-demand fact is never evaluated by an unfiltered call
			// (TestStatusTagsRenderAnOnDemandSentinelOnlyWhenAsked) — its
			// group is reached only through `--filter`, the same way the
			// write model's `overlap_uncited` is, so totality here is
			// proved over `--filter`, not the bare vector.
			if decl[key].OnDemand {
				continue
			}
			if _, ok := rendered[key]; ok {
				continue
			}
			f := decl[key]
			t.Errorf("fixture %s does not render dimension %q (source %s, absent declared: %v); "+
				"a guard atom over a missing tag refuses at resolve instead of routing — "+
				"declare the value absence takes on that fact, and claim its cell",
				rec, key, f.Source, f.HasAbsent)
		}
	}
}

// routingFixtures are the `status` fixture records, which span the shapes
// the routing tells apart — including the sparse Draft whose optional
// fields are simply not written.
var routingFixtures = []string{"0020", "0021", "0022", "0023", "0024", "0025", "0030"}

// fixtureTags renders one fixture's `--tags` argv into a key/value map,
// asserting the argv shape on the way through.
func fixtureTags(t *testing.T, table, rec string) map[string]string {
	t.Helper()
	code, out, errb := runCapture(t, "status", "--facts", table, "--tags", rec)
	if code != 0 {
		t.Fatalf("%s: --tags exit %d: %s", rec, code, errb)
	}
	got := map[string]string{}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := 0; i+1 < len(lines); i += 2 {
		if strings.TrimSpace(lines[i]) != "--tag" {
			t.Fatalf("%s: argv is not --tag/value pairs at line %d: %q", rec, i, lines[i])
		}
		k, v, _ := strings.Cut(strings.TrimSpace(lines[i+1]), "=")
		got[k] = v
	}
	return got
}

// TestRoutingSentinelsRenderOnlyAsTags asserts BOTH halves of what a
// sentinel is for, on the one fixture that omits the fields.
//
// `--tags` must render it, because a resolver's argv cannot spell
// absence and a guard atom over a missing tag refuses rather than routes.
// `--json` must still OMIT it, because there the distinction survives —
// and the fact table's whole header rests on it: false says "looked, not
// there", absent says "nothing looked".
//
// Either half alone is misleading, which is why they are asserted
// together. A sentinel that leaked into `--json` would quietly convert
// every "nothing looked" into a claim.
func TestRoutingSentinelsRenderOnlyAsTags(t *testing.T) {
	tbl := loadRealTable(t)
	_, table := bindStatusFixture(t)

	var sentinels []FactDecl
	for _, f := range tbl.Facts {
		// An on-demand fact renders its sentinel only under `--filter`
		// (TestStatusTagsRenderAnOnDemandSentinelOnlyWhenAsked); the
		// unfiltered argv this test reads never carries it.
		if f.HasAbsent && !f.OnDemand {
			sentinels = append(sentinels, f)
		}
	}
	if len(sentinels) == 0 {
		t.Fatal("no fact declares an `absent` sentinel; the routing dimensions cannot be total")
	}

	// 0025 is the sparse Draft: no Profile, no Cluster, no Joint-check:.
	const sparse = "0025"
	tags := fixtureTags(t, table, sparse)

	code, out, errb := runCapture(t, "status", "--json", "--facts", table, sparse)
	if code != 0 {
		t.Fatalf("%s: --json exit %d: %s", sparse, code, errb)
	}
	var env struct {
		Facts []Fact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	inJSON := map[string]bool{}
	for _, f := range env.Facts {
		inJSON[f.Name] = true
	}

	// At least one sentinel must actually be exercised, or this test
	// passes without proving anything.
	exercised := 0
	for _, f := range sentinels {
		if inJSON[f.Name] {
			continue // the fixture carries a real value for this one
		}
		exercised++
		got, ok := tags[f.Name]
		if !ok {
			t.Errorf("fact %q is absent on %s and declares absent=%q, but `--tags` omitted it; "+
				"the sentinel exists precisely so the dimension is always supplied",
				f.Name, sparse, f.Absent)
			continue
		}
		if got != f.Absent {
			t.Errorf("fact %q rendered %q on %s, want the declared sentinel %q", f.Name, got, sparse, f.Absent)
		}
		if fields := strings.Fields(f.Name + "=" + got); len(fields) != 1 {
			t.Errorf("sentinel %q=%q renders %d shell words, not one", f.Name, got, len(fields))
		}
	}
	if exercised == 0 {
		t.Fatalf("fixture %s carries a value for every sentinel fact, so this test proves nothing; "+
			"the absence fixture must omit the fields the sentinels stand in for", sparse)
	}
}

// TestEveryRoutingRuleAnswers checks that no rule can select and then say
// nothing.
//
// `emit` is the whole answer over this model class — there is no write
// block and no owned state — so a rule with no `next` is a row that
// resolves successfully and leaves the caller with nothing to run. That
// is worse than a refusal, which at least names itself.
func TestEveryRoutingRuleAnswers(t *testing.T) {
	m := loadRoutingModel(t)
	for _, id := range m.RuleIDs {
		e, ok := m.Emits[id]
		if !ok || len(e) == 0 {
			t.Errorf("rule %q emits nothing; over a decision table the emit block IS the answer", id)
			continue
		}
		if strings.TrimSpace(e["next"]) == "" {
			t.Errorf("rule %q emits no `next`; every row answers with a command, `none`, or a stopped: token", id)
		}
		if strings.TrimSpace(e["why"]) == "" {
			t.Errorf("rule %q emits no `why`; the reason is what the skill prints beside the answer", id)
		}
	}
}

// TestRoutingStopsAreNamed holds the §stop-packet rule: where the flow's
// answer is a judgement a fact cannot make, the row must SAY so rather
// than route to a plausible stage. A `stopped:` token is how it says it,
// and each one must name what is missing.
func TestRoutingStopsAreNamed(t *testing.T) {
	m := loadRoutingModel(t)
	for _, id := range m.RuleIDs {
		next := m.Emits[id]["next"]
		if !strings.HasPrefix(next, "stopped:") {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(next, "stopped:")) == "" {
			t.Errorf("rule %q emits a bare `stopped:` with no reason token", id)
		}
		if strings.TrimSpace(m.Emits[id]["surface"]) == "" {
			t.Errorf("rule %q stops but surfaces nothing; a stop names what the human must read", id)
		}
	}
}

// TestRoutingModelLints runs the real acceptance criterion when — and
// only when — `intrastate` is installed.
//
// The coverage proof is the whole reason this model is data rather than
// prose, so it is checked here where it can be. But `intrastate` is an
// accelerator for this flow and not a hard dependency: a contributor
// without it must still get a green suite, so an absent binary SKIPS.
// A skip that reads as a pass is exactly the failure the flow's own rules
// warn about, which is why the skip message says what went unchecked.
func TestRoutingModelLints(t *testing.T) {
	// The same resolution order the skill and rdr-doctor 12 use:
	// $RDR_INTRASTATE, else PATH. A test that only looked at PATH would
	// skip on a machine where the flow itself would have run the check.
	bin := os.Getenv("RDR_INTRASTATE")
	if bin == "" {
		found, err := exec.LookPath("intrastate")
		if err != nil {
			t.Skip("intrastate resolves neither from $RDR_INTRASTATE nor on PATH; " +
				"the model's coverage proof went UNCHECKED here " +
				"(run: intrastate lint --model models/rdr-status.toml)")
		}
		bin = found
	}
	// Both models: the write model's refuse cases are cells too, and a
	// guard added to one lock row opens a gap another row has to close.
	for _, name := range routingModelNames {
		model := repoFile(t, filepath.Join("models", name))
		out, err := exec.Command(bin, "lint", "--model", model, "--as", "json").CombinedOutput()
		if err != nil {
			t.Fatalf("intrastate lint refused the shipped model %s: %v\n%s", name, err, out)
		}
		// Exit 0 still carries advisories, and one of them matters: a table
		// closed by a bare escape row is closed, not proved. These models
		// claim every cell positively, so the advisory must be absent.
		if strings.Contains(string(out), "graph-coverage-closed-by-escape") {
			t.Errorf("%s: coverage is closed by an escape row rather than proved over its domains:\n%s", name, out)
		}
	}
}

// TestLockReadsGateStale pins the write model's lock group to the fact
// 52e31a1 added for it: a re-entered Draft keeps the gate.md its Final
// earned, so `gate_written` alone would lock over a gate that never saw
// the rework. The full lock must guard on `gate_stale` and a refusing row
// must claim the stale cell — the seam that the rdr-status signal table's
// `7 Finalize` row already promises.
func TestLockReadsGateStale(t *testing.T) {
	m := loadRoutingModelNamed(t, "rdr-write.toml")
	if _, ok := m.Kinds["gate_stale"]; !ok {
		t.Fatal("rdr-write.toml does not declare gate_stale; lock-draft can resolve over a pre-demote gate.md")
	}
	guards := map[string]map[string][]string{}
	for _, a := range m.Atoms {
		if guards[a.Rule] == nil {
			guards[a.Rule] = map[string][]string{}
		}
		guards[a.Rule][a.Key] = a.Literals
	}
	if got := guards["lock-draft"]["gate_stale"]; len(got) != 1 || got[0] != "false" {
		t.Errorf("lock-draft guards gate_stale on %v, want exactly [false]", got)
	}
	if got := guards["lock-gate-stale"]["gate_stale"]; len(got) != 1 || got[0] != "true" {
		t.Errorf("lock-gate-stale guards gate_stale on %v, want exactly [true]", got)
	}
	if op := m.Emits["lock-gate-stale"]["op"]; !strings.HasPrefix(op, "stopped:") {
		t.Errorf("lock-gate-stale emits op %q; a stale gate must refuse, not lock", op)
	}
}

// factCanCarry reports whether a fact could ever hold this literal.
func factCanCarry(f FactDecl, lit string) bool {
	switch f.Kind {
	case "enum":
		return containsString(f.Domain, lit)
	case "bool":
		return lit == "true" || lit == "false"
	case "int":
		for _, r := range lit {
			if r < '0' || r > '9' {
				return false
			}
		}
		return lit != ""
	}
	// A set is compared by containment rather than equality, and a
	// scalar has no declared domain to check against.
	return true
}

func containsString(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// TestLockPreservesJointDecisionQualifier pins the lock split on the
// qualifier form: a Draft still carrying `[joint decision → …]` must keep
// that qualifier through the status flip, because the navigator's
// `locate-final-joint-decision` row routes the home check on it. The bare
// `lock-draft` edit flattens the Status line to `Final`, so the
// joint-decision cell is carved out into its own row whose sed edit
// captures and re-emits the qualifier — "apply `edit` as handed" then
// preserves it with no caller judgement.
func TestLockPreservesJointDecisionQualifier(t *testing.T) {
	m := loadRoutingModelNamed(t, "rdr-write.toml")
	if kind := m.Kinds["status_form"]; kind != "enum" {
		t.Fatalf("rdr-write.toml declares status_form as %q, want enum; without it the lock cannot tell a joint-decision Draft apart", kind)
	}

	guards := map[string]map[string][]string{}
	for _, a := range m.Atoms {
		if guards[a.Rule] == nil {
			guards[a.Rule] = map[string][]string{}
		}
		guards[a.Rule][a.Key] = a.Literals
	}

	// The carved-out row: joint-decision form, and the same stale-gate
	// refusal boundary the bare lock holds.
	jd := guards["lock-draft-joint-decision"]
	if jd == nil {
		t.Fatal("rdr-write.toml has no lock-draft-joint-decision rule; a joint-decision Draft locks through the flattening edit")
	}
	if got := jd["status_form"]; len(got) != 1 || got[0] != "joint-decision" {
		t.Errorf("lock-draft-joint-decision guards status_form on %v, want exactly [joint-decision]", got)
	}
	if got := jd["gate_stale"]; len(got) != 1 || got[0] != "false" {
		t.Errorf("lock-draft-joint-decision guards gate_stale on %v, want exactly [false]; the stale-gate refusal must still win", got)
	}
	if op := m.Emits["lock-draft-joint-decision"]["op"]; op != "lock" {
		t.Errorf("lock-draft-joint-decision emits op %q, want lock", op)
	}

	// The bare row must carry the carve-out atom (lint proves the cells
	// stay disjoint; this pins that the atom exists at all) and its edit
	// must stay the bare flip.
	if got := guards["lock-draft"]["status_form"]; len(got) != 1 || got[0] != "joint-decision" {
		t.Errorf("lock-draft carries no status_form atom over joint-decision (%v); its edit would flatten the qualifier", got)
	}
	if bare := m.Emits["lock-draft"]["edit"]; strings.Contains(bare, "joint") {
		t.Errorf("lock-draft edit %q mentions the qualifier; the split belongs to the joint-decision row", bare)
	}

	// The parse keeps the TOML escaping (`\\*` for `\*`); the caller's
	// shell sees the unescaped expression, so unescape before judging it.
	edit := strings.ReplaceAll(m.Emits["lock-draft-joint-decision"]["edit"], `\\`, `\`)
	if !strings.Contains(edit, `\(\[joint decision`) || !strings.Contains(edit, `Final \1`) {
		t.Fatalf("lock-draft-joint-decision edit %q does not capture and re-emit the qualifier", edit)
	}

	// The edit is data the caller applies as handed, so its behavior is
	// pinned by running it: the qualifier must survive onto Final.
	if _, err := exec.LookPath("sed"); err != nil {
		t.Skip("sed not on PATH; the edit's capture behavior went UNCHECKED here")
	}
	in := "- **Status**: Draft [joint decision → 0042-frame-grammar § A3: who owns the trailing pad byte]\n"
	cmd := exec.Command("sed", edit)
	cmd.Stdin = strings.NewReader(in)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("sed refused the emitted edit %q: %v", edit, err)
	}
	want := "- **Status**: Final [joint decision → 0042-frame-grammar § A3: who owns the trailing pad byte]\n"
	if string(out) != want {
		t.Errorf("the emitted edit rewrote the Status line to %q, want %q", out, want)
	}
}

// TestCompletionOutcomesReadLensStale pins the status model's critique and
// repeatability groups to the `lens_stale` fact the lens group already
// reads: a re-entered Draft keeps every file its Final earned, so a
// completion row guarding only run/model/file flags would answer "finished"
// over evidence that never saw the rework — while the lens group, one
// outcome over, says the same lens is stale and owed. Each done-claiming
// row must exclude the stale slice, and a dedicated stale row must claim
// it with an answer that routes back to the lens rather than reading as
// done.
//
// The shared parser collapses `all` and `unless` blocks, and the split is
// the whole point here, so this test reads the rule chunks directly.
func TestCompletionOutcomesReadLensStale(t *testing.T) {
	src, err := os.ReadFile(repoFile(t, filepath.Join("models", routingModelName)))
	if err != nil {
		t.Fatal(err)
	}
	chunks := map[string]string{}
	for _, chunk := range strings.Split(string(src), "[[rule]]") {
		for _, line := range strings.Split(chunk, "\n") {
			line = strings.TrimSpace(line)
			if rest, ok := strings.CutPrefix(line, "id = "); ok {
				chunks[strings.Trim(rest, `"`)] = chunk
				break
			}
		}
	}

	// The done-claiming rows: every one whose emit reads as the lens being
	// finished (`none`, or done-with-a-caveat) over on-disk evidence.
	completion := map[string]string{
		"critique-large-complete":                     "critique",
		"critique-large-diffed":                       "critique",
		"critique-foundational-complete":              "critique",
		"critique-foundational-single-model-fallback": "critique",
		"critique-foundational-unstamped":             "critique",
		"repeatability-lite-complete":                 "repeatability",
		"repeatability-lite-variant-mismatch":         "repeatability",
		"repeatability-lite-mismatch-both":            "repeatability",
		"repeatability-lite-mismatch-run3":            "repeatability",
		"repeatability-full-complete":                 "repeatability",
	}
	for id, lens := range completion {
		chunk, ok := chunks[id]
		if !ok {
			t.Errorf("rule %q is gone; the completion rows are pinned to their stale guard", id)
			continue
		}
		if !strings.Contains(chunk, "[rule.guard.unless.lens_stale]\neq = \""+lens+"\"") {
			t.Errorf("rule %q does not exclude lens_stale=%q; it can read a pre-demote %s as finished", id, lens, lens)
		}
	}

	// The stale rows that claim the excluded slice: they must route back to
	// the lens and never read as done.
	stale := map[string]string{
		"critique-stale":      "critique",
		"repeatability-stale": "repeatability",
	}
	for id, lens := range stale {
		chunk, ok := chunks[id]
		if !ok {
			t.Errorf("no rule %q claims the lens_stale=%q slice the completion rows exclude", id, lens)
			continue
		}
		if !strings.Contains(chunk, "[rule.guard.all.lens_stale]\neq = \""+lens+"\"") {
			t.Errorf("rule %q does not guard all.lens_stale=%q", id, lens)
		}
		_, emit, ok := strings.Cut(chunk, "[rule.emit]")
		if !ok {
			t.Errorf("rule %q has no emit block", id)
			continue
		}
		if !strings.Contains(emit, "/rdr-prelock "+lens) {
			t.Errorf("rule %q must route back to the lens (`/rdr-prelock %s`); its emit is:\n%s", id, lens, emit)
		}
		if strings.Contains(strings.ToLower(emit), "complete") {
			t.Errorf("rule %q answers over a stale lens and must not say complete; its emit is:\n%s", id, emit)
		}
	}
}

// launchModelName is the Stage-8 launch table: the size gate's route and
// the completion gate's verdict.
const launchModelName = "rdr-launch.toml"

// emitNextDomain reads a model's `[emit.next.domain]` partitions —
// disposition name to members — with the same local, line-oriented
// reading loadRoutingModelNamed uses, and for the same reason.
func emitNextDomain(t *testing.T, name string) map[string][]string {
	t.Helper()
	src, err := os.ReadFile(repoFile(t, filepath.Join("models", name)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	in := false
	for _, raw := range strings.Split(string(src), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = line == "[emit.next.domain]"
			continue
		}
		if !in {
			continue
		}
		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("[emit.next.domain] line is not key = list: %q", line)
		}
		out[strings.TrimSpace(key)] = splitTOMLList(strings.TrimSpace(rest))
	}
	if len(out) == 0 {
		t.Fatalf("%s declares no [emit.next.domain]; the dispositions are the caller's contract", name)
	}
	return out
}

// TestLaunchEmitsAreDeclaredDispositions pins the launch table's answer
// surface both ways. Forward: every row's `next` is a member of a declared
// partition, so the caller branches on a proven disposition — `route` is
// applied as inline-or-delegated, `done` is the capsule's state word, and
// `stop` is the INCOMPLETE blocker — never on a string it has to
// recognise. Backward: every declared member is emitted by some row, so
// the domain carries no stop token nothing can ever produce. And each
// group keeps its own vocabulary: a size row never stops (the prose gate
// routed, it did not halt), a completion row never routes, a shard row
// names a worklist or stops on the unread projection, and a budget row
// only ever says what the leg does next (`loop`).
func TestLaunchEmitsAreDeclaredDispositions(t *testing.T) {
	m := loadRoutingModelNamed(t, launchModelName)
	domain := emitNextDomain(t, launchModelName)
	for _, part := range []string{"route", "done", "go", "loop", "stop"} {
		if len(domain[part]) == 0 {
			t.Errorf("[emit.next.domain] declares no %q partition", part)
		}
	}
	member := map[string]string{}
	for part, vals := range domain {
		for _, v := range vals {
			member[v] = part
		}
	}

	// Which group a rule belongs to is its `recognized` match atom; the
	// parser keeps guard atoms only, so read the id prefix the file uses.
	emitted := map[string]bool{}
	for _, id := range m.RuleIDs {
		next := m.Emits[id]["next"]
		part, ok := member[next]
		if !ok {
			t.Errorf("rule %q emits next %q, which no [emit.next.domain] partition declares", id, next)
			continue
		}
		emitted[next] = true
		switch {
		case strings.HasPrefix(id, "size"):
			if part != "route" {
				t.Errorf("size row %q emits %q (%s); the size gate routes and never stops or completes", id, next, part)
			}
		case strings.HasPrefix(id, "complete"):
			if part == "route" {
				t.Errorf("completion row %q emits %q; the completion gate verdicts, it does not route", id, next)
			}
		case strings.HasPrefix(id, "precheck"):
			if part != "go" && part != "stop" {
				t.Errorf("precheck row %q emits %q (%s); the precheck proceeds or stops, it neither routes nor completes", id, next, part)
			}
		case strings.HasPrefix(id, "shard"):
			if part != "route" && part != "stop" {
				t.Errorf("shard row %q emits %q (%s); the shard route names a worklist or stops on an unread impact.md", id, next, part)
			}
		case strings.HasPrefix(id, "budget"):
			if part != "loop" {
				t.Errorf("budget row %q emits %q (%s); a budget row says what the leg does next and nothing else", id, next, part)
			}
		default:
			t.Errorf("rule %q belongs to none of the size, complete, precheck, shard or budget groups by id", id)
		}
	}
	for v := range member {
		if !emitted[v] {
			t.Errorf("[emit.next.domain] declares %q and no row emits it; a member nothing produces is a caller branch nothing reaches", v)
		}
	}
	for _, id := range m.RuleIDs {
		if strings.TrimSpace(m.Emits[id]["why"]) == "" {
			t.Errorf("rule %q emits no why; the ROUTE and INCOMPLETE lines print it", id)
		}
	}
}

// intrastateBinary resolves the routing binary the way the skills and
// rdr-doctor 12 do — $RDR_INTRASTATE, else PATH — or skips, saying what
// went unchecked. A skip that read as a pass is the failure the flow's
// own rules warn about.
func intrastateBinary(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("RDR_INTRASTATE"); bin != "" {
		return bin
	}
	found, err := exec.LookPath("intrastate")
	if err != nil {
		t.Skip("intrastate resolves neither from $RDR_INTRASTATE nor on PATH; the live resolve went UNCHECKED here")
	}
	return found
}

// filteredTagArgv renders one fixture's `--tags --filter` argv as the
// resolver receives it, asserting the pair shape on the way through.
func filteredTagArgv(t *testing.T, table, rec, filter string) []string {
	t.Helper()
	code, out, errb := runCapture(t, "status", "--facts", table, "--tags", "--filter", filter, rec)
	if code != 0 {
		t.Fatalf("%s: --tags exit %d: %s", rec, code, errb)
	}
	var argv []string
	for _, ln := range strings.Split(strings.TrimSpace(out), "\n") {
		argv = append(argv, strings.TrimSpace(ln))
	}
	if len(argv)%2 != 0 {
		t.Fatalf("%s: --tags argv is not --tag/value pairs: %q", rec, argv)
	}
	return argv
}

// TestLaunchModelResolvesTheFixture is the seam end to end, once per
// gate: `rdr status --tags --filter …` renders the facts, the launch
// prompt adds what it observed, and `intrastate flow resolve --plan-only`
// selects one row. Fixture 0030 is the table's one COMPLETE cell on disk
// and, being small and short, its inline cell at PRECHECKS — where the
// REQ list is already written, so the counted inline row fires rather
// than the uncounted one. Neither binary calls the other; this test is
// the composition the prompt performs in one Bash call.
func TestLaunchModelResolvesTheFixture(t *testing.T) {
	bin := intrastateBinary(t)
	_, table := bindStatusFixture(t)
	model := repoFile(t, filepath.Join("models", launchModelName))

	resolve := func(outcome string, argv []string, extra ...string) (rule, next string) {
		t.Helper()
		args := append([]string{"flow", "resolve", "--model", model, "--outcome", outcome, "--plan-only", "--as", "json"}, argv...)
		for i := 0; i+1 < len(extra); i += 2 {
			args = append(args, "--tag", extra[i]+"="+extra[i+1])
		}
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("intrastate refused the %s resolve: %v\n%s", outcome, err, out)
		}
		var env struct {
			Data struct {
				Rule string            `json:"rule"`
				Emit map[string]string `json:"emit"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &env); err != nil {
			t.Fatalf("%s: unreadable plan: %v\n%s", outcome, err, out)
		}
		return env.Data.Rule, env.Data.Emit["next"]
	}

	argv := filteredTagArgv(t, table, "0030", "profile,lines,req_count")
	if rule, next := resolve("size", argv, "files", "0-3", "suite", "quick", "pressure", "false"); rule != "size-inline" || next != "inline" {
		t.Errorf("size gate on 0030 at PRECHECKS: rule %q next %q, want size-inline/inline", rule, next)
	}
	// The same record with a 4th file touched mid-run is the fallback:
	// the same row re-asked, answering delegated.
	if rule, next := resolve("size", argv, "files", "4+", "suite", "quick", "pressure", "false"); rule != "size-files" || next != "delegated" {
		t.Errorf("size gate on 0030 with a 4th file: rule %q next %q, want size-files/delegated", rule, next)
	}

	argv = filteredTagArgv(t, table, "0030", "impl_orphans,impl_open_decisions,impl_mvv_recorded")
	if rule, next := resolve("complete", argv, "suite_green", "true"); rule != "complete" || next != "COMPLETE" {
		t.Errorf("completion gate on 0030: rule %q next %q, want complete/COMPLETE", rule, next)
	}
	// 0021 has no ledger at all: every impl fact renders its sentinel and
	// the gate names the unread file rather than passing a check it never ran.
	argv = filteredTagArgv(t, table, "0021", "impl_orphans,impl_open_decisions,impl_mvv_recorded")
	if rule, next := resolve("complete", argv, "suite_green", "true"); rule != "complete-coverage-unread" || next != "stopped:coverage-unread" {
		t.Errorf("completion gate on 0021 (no ledger): rule %q next %q, want complete-coverage-unread", rule, next)
	}

	// The precheck: status x predecessors_state x baseline, every row
	// reachable from a fixture. 0020 is Draft; 0032 names a record the dir
	// lacks; 0031 names one with no capsule; 0030 names none, so the
	// baseline decides: unrun re-asks, red stops, green proceeds.
	for _, c := range []struct{ rec, baseline, rule, next string }{
		{"0020", "none", "precheck-not-final", "stopped:not-final"},
		{"0032", "none", "precheck-unresolved", "stopped:predecessor-unresolved"},
		{"0031", "none", "precheck-incomplete", "stopped:predecessor-incomplete"},
		{"0030", "none", "precheck-baseline-unrun", "run-baseline"},
		{"0030", "red", "precheck-baseline-red", "stopped:baseline-red"},
		{"0030", "green", "precheck-ok", "proceed"},
	} {
		argv = filteredTagArgv(t, table, c.rec, "status,predecessors_state")
		if rule, next := resolve("precheck", argv, "baseline", c.baseline); rule != c.rule || next != c.next {
			t.Errorf("precheck on %s: rule %q next %q, want %s/%s", c.rec, rule, next, c.rule, c.next)
		}
	}

	// The shard route reads one fact: 0030's artifacts carry an impact.md
	// in `rdr impact`'s shape with `families: 2`, so the worklist is
	// sharded; 0021 has no artifacts dir, so the fact renders its sentinel
	// and the route stops on the unread projection rather than reading
	// nothing as an empty radius.
	argv = filteredTagArgv(t, table, "0030", "impact_families")
	if rule, next := resolve("shard", argv); rule != "shard-sharded" || next != "sharded" {
		t.Errorf("shard on 0030: rule %q next %q, want shard-sharded/sharded", rule, next)
	}
	argv = filteredTagArgv(t, table, "0021", "impact_families")
	if rule, next := resolve("shard", argv); rule != "shard-impact-unread" || next != "stopped:impact-unread" {
		t.Errorf("shard on 0021 (no impact.md): rule %q next %q, want shard-impact-unread/stopped:impact-unread", rule, next)
	}

	// The budget is caller tags only — the leg observes git, the clock
	// and its last suite exit — so every one of its 8 cells resolves
	// with no fixture: green returns whatever the caps say, either cap
	// over cuts the leg, under both continues.
	for _, c := range []struct{ green, elapsed, commits, rule, next string }{
		{"true", "0-30", "0-5", "budget-green", "return-green"},
		{"true", "0-30", "6+", "budget-green", "return-green"},
		{"true", "31+", "0-5", "budget-green", "return-green"},
		{"true", "31+", "6+", "budget-green", "return-green"},
		{"false", "31+", "0-5", "budget-elapsed", "return-partial"},
		{"false", "31+", "6+", "budget-elapsed", "return-partial"},
		{"false", "0-30", "6+", "budget-commits", "return-partial"},
		{"false", "0-30", "0-5", "budget-continue", "continue"},
	} {
		if rule, next := resolve("budget", nil, "suite_green", c.green, "elapsed", c.elapsed, "commits", c.commits); rule != c.rule || next != c.next {
			t.Errorf("budget suite_green=%s elapsed=%s commits=%s: rule %q next %q, want %s/%s", c.green, c.elapsed, c.commits, rule, next, c.rule, c.next)
		}
	}
}

// TestWriteDemoteWritesTheQualifierTarget: every `demote` edit carries the
// `@<stage>` token the row's `stage` names, and that token is one the
// qualifier grammar accepts (model.ReentryTargets). The class -> stage
// mapping is the row's; a token the grammar cannot read would degrade
// the qualifier to a free-text note and blind every re-entry rule.
func TestWriteDemoteWritesTheQualifierTarget(t *testing.T) {
	m := loadRoutingModelNamed(t, "rdr-write.toml")
	seen := 0
	for _, id := range m.RuleIDs {
		em := m.Emits[id]
		if !strings.HasPrefix(id, "demote-") || em["op"] != "demote" {
			continue
		}
		seen++
		stage := em["stage"]
		if !containsString(model.ReentryTargets, stage) {
			t.Errorf("rule %q emits stage %q, which the qualifier grammar does not accept (%v)", id, stage, model.ReentryTargets)
		}
		if !strings.Contains(em["edit"], "@"+stage+" ") {
			t.Errorf("rule %q emits stage %q but its edit does not write `@%s `:\n%s", id, stage, stage, em["edit"])
		}
	}
	if seen == 0 {
		t.Fatal("rdr-write.toml has no demote row emitting op=demote; the parse moved and this test is blind")
	}
	for _, want := range model.ReentryTargets {
		if !containsString(m.EmitDomains["stage"], want) {
			t.Errorf("[emit.stage] does not declare %q, which the qualifier grammar accepts", want)
		}
	}
}

// TestWriteReturnStaysInTheStatusRouteDomain polices the one duplication
// the `return` group carries: its `next_action` values are commands the
// navigator (rdr-status.toml) already declares as `route`. A return
// stage the navigator cannot route to is a packet the cascade relays to
// nowhere.
func TestWriteReturnStaysInTheStatusRouteDomain(t *testing.T) {
	w := loadRoutingModelNamed(t, "rdr-write.toml")
	st := loadRoutingModelNamed(t, "rdr-status.toml")
	route := st.EmitDomains["next.route"]
	if len(route) == 0 {
		t.Fatal("rdr-status.toml declares no [emit.next.domain] route partition; the parse moved")
	}
	for _, v := range w.EmitDomains["next_action"] {
		if !containsString(route, v) {
			t.Errorf("[emit.next_action] admits %q, which rdr-status.toml's route domain does not (%v)", v, route)
		}
	}
	seen := 0
	for _, id := range w.RuleIDs {
		em := w.Emits[id]
		if !strings.HasPrefix(id, "return-") || em["op"] != "return" {
			continue
		}
		seen++
		if !containsString(w.EmitDomains["next_action"], em["next_action"]) {
			t.Errorf("rule %q emits next_action %q outside [emit.next_action]", id, em["next_action"])
		}
		if !containsString(w.EmitDomains["stage"], em["stage"]) {
			t.Errorf("rule %q emits stage %q outside [emit.stage]", id, em["stage"])
		}
	}
	if seen == 0 {
		t.Fatal("rdr-write.toml has no return row emitting op=return; the parse moved and this test is blind")
	}
}

// TestWriteProfileEmitMatchesTheProfileFact: what the `profile` group can
// write is exactly what the `profile` fact can read back, minus the
// absent sentinel — a value the table emits and the projector cannot
// render would route nothing at the next stage.
func TestWriteProfileEmitMatchesTheProfileFact(t *testing.T) {
	w := loadRoutingModelNamed(t, "rdr-write.toml")
	tbl := loadRealTable(t)
	var fact []string
	for _, f := range tbl.Facts {
		if f.Name == "profile" {
			for _, v := range f.Domain {
				if v != f.Absent {
					fact = append(fact, v)
				}
			}
		}
	}
	if len(fact) == 0 {
		t.Fatal("the fact table declares no enum `profile`")
	}
	emit := w.EmitDomains["profile"]
	for _, v := range fact {
		if !containsString(emit, v) {
			t.Errorf("fact profile can carry %q, which [emit.profile] does not declare (%v)", v, emit)
		}
	}
	for _, v := range emit {
		if !containsString(fact, v) {
			t.Errorf("[emit.profile] declares %q, which fact profile cannot carry (%v)", v, fact)
		}
	}
	for _, id := range w.RuleIDs {
		if em := w.Emits[id]; em["op"] == "profile" && !containsString(emit, em["profile"]) {
			t.Errorf("rule %q emits profile %q outside [emit.profile]", id, em["profile"])
		}
	}
}

// TestLensRowHasOneHome pins the Stage-5 lens row to its one home, the
// `lens` group of the status model, both ways.
//
// Forward: no stage, skill, or prompt spells the row out — a
// `grounding → 3amigo → critique` chain in prose is a second table that
// drifts from the model and invites a reader to walk it instead of
// reading `emit.row`. The sweep covers every prose tree; the model is the
// home, fixtures and testdata are pinned copies, and README.md carries
// the matrix under an explicit non-normative note.
//
// Backward: every `[emit.row]` member is emitted by some lens rule, so the
// declared domain carries no span nothing can ever produce, and every
// lens rule that routes (not a stop) emits one — draft-to-lock writes its
// `lenses:` line from that value as handed.
func TestLensRowHasOneHome(t *testing.T) {
	re := regexp.MustCompile(`(grounding|cove)[[:space:]]*(→|->|\+)[[:space:]]*3amigo|3amigo[[:space:]]*(→|->|\+)[[:space:]]*critique|critique[[:space:]]*(→|->|\+)[[:space:]]*repeatability`)
	skip := []string{"models", filepath.Join("skills", "rdr-doctor", "fixtures"), filepath.Join("tools", "rdr", "testdata")}
	allow := map[string]bool{"README.md": true}
	for _, top := range []string{"README.md", "stages", "skills", "prompts"} {
		root := repoFile(t, top)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repoFile(t, "."), path)
			for _, s := range skip {
				if rel == s || strings.HasPrefix(rel, s+string(filepath.Separator)) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if d.IsDir() || allow[rel] || !strings.HasSuffix(path, ".md") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(src), "\n") {
				if re.MatchString(line) {
					t.Errorf("%s:%d spells the lens row in prose; the row is the `lens` group's `emit.row`:\n%s", rel, i+1, line)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	m := loadRoutingModel(t)
	domain := m.EmitDomains["row"]
	if len(domain) == 0 {
		t.Fatal("the status model declares no [emit.row]; the lens span has no home")
	}
	emitted := map[string]string{}
	for _, id := range m.RuleIDs {
		if !strings.HasPrefix(id, "lens-") {
			continue
		}
		em := m.Emits[id]
		row, ok := em["row"]
		if strings.HasPrefix(em["next"], "stopped:") {
			if ok {
				t.Errorf("rule %q stops and must emit no row; it emits %q", id, row)
			}
			continue
		}
		if !ok {
			t.Errorf("rule %q routes and must emit `row`", id)
			continue
		}
		if !containsString(domain, row) {
			t.Errorf("rule %q emits row %q, not a member of [emit.row]", id, row)
		}
		if strings.Contains(row, " ") {
			t.Errorf("rule %q emits row %q with a space; a span is one shell word", id, row)
		}
		emitted[row] = id
	}
	for _, v := range domain {
		if emitted[v] == "" {
			t.Errorf("[emit.row] declares %q, which no lens rule emits", v)
		}
	}
}

// TestDeterminacyGroupRoutesTheWrittenLine is the Stage-5 Determinacy
// seam, resolved live. The judgement — is a locked contract algorithmic?
// — is a reading no fact makes; what the table routes on is the
// `Determinacy:` line that reading leaves in Normative Contracts, read as
// the `determinacy` fact. The mid/large `repeatability` row with no run-1
// CHAINS to the `determinacy` group (its own group is at 1440, and a
// third value would overflow it), and that group answers once instead of
// the ladder being re-walked at Stages 5, 6 and 7.
//
// The goldens the prose ladder gave, recorded before it was deleted:
//
//	cue judged | run-1+diff | n/a line | before                | after
//	no cue     | -          | -        | READY, unrecorded     | NOT READY until `na` is written
//	fired      | yes        | -        | READY                 | `fired` -> lite rows -> none
//	fired      | no         | yes      | READY                 | `na` -> none
//	fired      | no         | no       | NOT READY             | `fired` -> route run 1
//	unjudged   | -          | -        | re-walked at 5, 6, 7  | stop, once
func TestDeterminacyGroupRoutesTheWrittenLine(t *testing.T) {
	bin := intrastateBinary(t)
	_, table := bindStatusFixture(t)
	model := repoFile(t, filepath.Join("models", routingModelName))

	tags := func(rec string) []string {
		t.Helper()
		code, out, errb := runCapture(t, "status", "--facts", table, "--tags", rec)
		if code != 0 {
			t.Fatalf("%s: --tags exit %d: %s", rec, code, errb)
		}
		return strings.Split(strings.TrimSpace(out), "\n")
	}
	// with is the shell's `${V/key=old/key=new}`: one tag rewritten, the
	// rest of the record's vector untouched.
	with := func(argv []string, key, value string) []string {
		out := append([]string(nil), argv...)
		for i, a := range out {
			if strings.HasPrefix(a, key+"=") {
				out[i] = key + "=" + value
				return out
			}
		}
		t.Fatalf("no %s= tag in %q", key, argv)
		return nil
	}
	resolve := func(outcome string, argv []string) (rule, next string) {
		t.Helper()
		args := append([]string{"flow", "resolve", "--model", model, "--outcome", outcome, "--plan-only", "--as", "json"}, argv...)
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("intrastate refused the %s resolve: %v\n%s", outcome, err, out)
		}
		var env struct {
			Data struct {
				Rule string            `json:"rule"`
				Emit map[string]string `json:"emit"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &env); err != nil {
			t.Fatalf("%s: unreadable plan: %v\n%s", outcome, err, out)
		}
		return env.Data.Rule, env.Data.Emit["next"]
	}
	expect := func(what, outcome string, argv []string, rule, next string) {
		t.Helper()
		if r, n := resolve(outcome, argv); r != rule || n != next {
			t.Errorf("%s: --outcome %s selected %q -> %q, want %q -> %q", what, outcome, r, n, rule, next)
		}
	}

	// 0020: mid, no run-1, no line — the unjudged sentinel, stopped once.
	v20 := tags("0020")
	expect("0020", "repeatability", v20, "repeatability-mid-large-unrun", "resolve:determinacy")
	expect("0020", "determinacy", v20, "determinacy-unjudged", "stopped:determinacy-trigger-unjudged")
	expect("0020 with na written", "determinacy", with(v20, "determinacy", "na"), "determinacy-na", "none")
	expect("0020 with fired written", "determinacy", with(v20, "determinacy", "fired"), "determinacy-fired-mid", "/rdr-prelock repeatability 1")
	expect("0020 as large, fired", "determinacy", with(with(v20, "determinacy", "fired"), "profile", "large"), "determinacy-fired-large", "/rdr-prelock repeatability 1")
	expect("0020 with no Profile", "determinacy", with(v20, "profile", "none"), "determinacy-no-profile", "stopped:no-profile")

	// 0024: mid with the line written n/a — the record that owes no run,
	// end to end through the chain.
	v24 := tags("0024")
	expect("0024", "repeatability", v24, "repeatability-mid-large-unrun", "resolve:determinacy")
	expect("0024", "determinacy", v24, "determinacy-na", "none")

	// 0026: foundational carries a fired line, and its route is unchanged
	// — the full lens is on the row, so the trigger adds nothing.
	v26 := tags("0026")
	expect("0026", "determinacy", v26, "determinacy-foundational", "none")
	if _, next := resolve("repeatability", v26); next != "/rdr-prelock repeatability 2" {
		t.Errorf("0026: --outcome repeatability answers %q; a foundational route must not read the line", next)
	}

	// 0030: small has no Stage 5.
	expect("0030", "determinacy", tags("0030"), "determinacy-small", "none")
}

// TestLockRefusesAnOpenJointDecision pins the write model's joint-
// decision fence. The two locking rows read `joint_check_home` and lock
// over none/clear/homed/unhomed (a positive atom: intrastate's `unless`
// is a block-level conjunction, so a second `unless` key would have
// widened the row rather than narrowed it); the open cell is claimed by
// a refusing row that keeps the gate refusals' precedence — unhomed
// locks, because a home the tool cannot resolve (a register outside the
// records dir) is not a home it may refuse; and the `fence` group is
// exactly the three overlap rows, every stop a declared disposition.
func TestLockRefusesAnOpenJointDecision(t *testing.T) {
	m := loadRoutingModelNamed(t, "rdr-write.toml")
	guards := map[string]map[string][]string{}
	for _, a := range m.Atoms {
		if guards[a.Rule] == nil {
			guards[a.Rule] = map[string][]string{}
		}
		guards[a.Rule][a.Key] = a.Literals
	}
	for _, id := range []string{"lock-draft", "lock-draft-joint-decision"} {
		got := guards[id]["joint_check_home"]
		if len(got) != 4 || !containsString(got, "none") || !containsString(got, "clear") || !containsString(got, "homed") || !containsString(got, "unhomed") {
			t.Errorf("%s guards joint_check_home on %v, want exactly [none clear homed unhomed]", id, got)
		}
	}
	for id, want := range map[string][2]string{
		"lock-joint-open": {"open", "stopped:joint-decision-open"},
	} {
		g := guards[id]
		if g == nil {
			t.Errorf("no rule %q; the %s cell locks through the bare row", id, want[0])
			continue
		}
		if got := g["joint_check_home"]; len(got) != 1 || got[0] != want[0] {
			t.Errorf("%s guards joint_check_home on %v, want [%s]", id, got, want[0])
		}
		if got := g["gate_stale"]; len(got) != 1 || got[0] != "false" {
			t.Errorf("%s guards gate_stale on %v; the stale-gate refusal must keep precedence", id, got)
		}
		if got := g["gate_written"]; len(got) != 1 || got[0] != "true" {
			t.Errorf("%s guards gate_written on %v; the no-gate refusal must keep precedence", id, got)
		}
		if op := m.Emits[id]["op"]; op != want[1] {
			t.Errorf("%s emits op %q, want %s", id, op, want[1])
		}
		if strings.TrimSpace(m.Emits[id]["surface"]) == "" {
			t.Errorf("%s stops but surfaces nothing", id)
		}
	}

	if !containsString(m.Outcomes, "fence") {
		t.Fatalf("outcomes %v carry no fence", m.Outcomes)
	}
	fence := map[string]string{
		"fence-clear":             "none",
		"fence-uncited":           "stopped:overlap-uncited",
		"fence-unchecked":         "stopped:overlap-unchecked",
		"fence-rulings-open":      "stopped:rulings-open",
		"fence-rulings-unchecked": "stopped:rulings-unchecked",
	}
	// The two rulings rows guard on rulings_open alone (any overlap_uncited);
	// the three overlap rows now carry both dimensions.
	fenceOverlapDims := map[string]bool{"fence-clear": true, "fence-uncited": true, "fence-unchecked": true}
	for _, id := range m.RuleIDs {
		if !strings.HasPrefix(id, "fence-") {
			continue
		}
		want, ok := fence[id]
		if !ok {
			t.Errorf("unexpected fence row %q", id)
			continue
		}
		delete(fence, id)
		if op := m.Emits[id]["op"]; op != want {
			t.Errorf("%s emits op %q, want %s", id, op, want)
		}
		if fenceOverlapDims[id] {
			if g := guards[id]; len(g["overlap_uncited"]) != 1 || len(g["rulings_open"]) != 1 {
				t.Errorf("%s guards %v, want exactly one overlap_uncited literal and one rulings_open literal", id, g)
			}
		} else if g := guards[id]; len(g["rulings_open"]) != 1 {
			t.Errorf("%s guards %v, want exactly one rulings_open literal", id, g)
		}
	}
	for id := range fence {
		t.Errorf("fence row %q is missing", id)
	}
	stops := m.EmitDomains["op.stop"]
	for _, tok := range []string{"stopped:joint-decision-open", "stopped:overlap-uncited", "stopped:overlap-unchecked", "stopped:rulings-open", "stopped:rulings-unchecked"} {
		if !containsString(stops, tok) {
			t.Errorf("%s is emitted but not a declared stop disposition (%v)", tok, stops)
		}
	}
}

// TestLockAndFenceResolveTheFixtures runs the seam live: the fixture's
// `--tags --filter` argv, the gate facts the finalize prompt asserts, and
// `intrastate flow resolve` selecting one row — the open fire refuses,
// the homed one locks, the unhomed one locks with its home visible on
// the vector, and the fence answers over the fixture's uncited pair.
func TestLockAndFenceResolveTheFixtures(t *testing.T) {
	bin := intrastateBinary(t)
	_, table := bindStatusFixture(t)
	model := repoFile(t, filepath.Join("models", "rdr-write.toml"))
	resolve := func(outcome string, argv []string, extra ...string) (rule, op string) {
		t.Helper()
		args := append([]string{"flow", "resolve", "--model", model, "--outcome", outcome, "--plan-only", "--as", "json"}, argv...)
		for i := 0; i+1 < len(extra); i += 2 {
			args = append(args, "--tag", extra[i]+"="+extra[i+1])
		}
		out, err := exec.Command(bin, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("intrastate refused the %s resolve: %v\n%s", outcome, err, out)
		}
		var env struct {
			Data struct {
				Rule string            `json:"rule"`
				Emit map[string]string `json:"emit"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &env); err != nil {
			t.Fatalf("%s: unreadable plan: %v\n%s", outcome, err, out)
		}
		return env.Data.Rule, env.Data.Emit["op"]
	}
	for _, c := range []struct{ rec, rule, op string }{
		{"0033", "lock-joint-open", "stopped:joint-decision-open"},
		// A home the tool could not resolve is not the lock's refusal.
		{"0022", "lock-draft", "lock"},
		{"0026", "lock-draft", "lock"},
		{"0020", "lock-draft", "lock"},
		// No line at all is not the lock's refusal: `joint_checks` surfaces it.
		{"0025", "lock-draft", "lock"},
	} {
		argv := filteredTagArgv(t, table, c.rec, "status,status_form,joint_check_home")
		if rule, op := resolve("lock", argv, "gate_written", "true", "gate_stale", "false"); rule != c.rule || op != c.op {
			t.Errorf("lock on %s: rule %q op %q, want %s/%s", c.rec, rule, op, c.rule, c.op)
		}
	}
	for _, c := range []struct{ rec, rule, op string }{
		{"0025", "fence-uncited", "stopped:overlap-uncited"},
		{"0033", "fence-clear", "none"},
	} {
		argv := filteredTagArgv(t, table, c.rec, "overlap_uncited,rulings_open")
		if rule, op := resolve("fence", argv); rule != c.rule || op != c.op {
			t.Errorf("fence on %s: rule %q op %q, want %s/%s", c.rec, rule, op, c.rule, c.op)
		}
	}
	if rule, op := resolve("fence", nil, "overlap_uncited", "unchecked", "rulings_open", "0"); rule != "fence-unchecked" || op != "stopped:overlap-unchecked" {
		t.Errorf("fence unchecked: rule %q op %q", rule, op)
	}
	if rule, op := resolve("fence", nil, "overlap_uncited", "0", "rulings_open", "1+"); rule != "fence-rulings-open" || op != "stopped:rulings-open" {
		t.Errorf("fence rulings open: rule %q op %q", rule, op)
	}
	if rule, op := resolve("fence", nil, "overlap_uncited", "0", "rulings_open", "unchecked"); rule != "fence-rulings-unchecked" || op != "stopped:rulings-unchecked" {
		t.Errorf("fence rulings unchecked: rule %q op %q", rule, op)
	}
}
