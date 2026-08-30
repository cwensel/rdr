package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
// second binary installed — which matters because `intrastate` is an
// accelerator here, never a hard dependency.

const routingModelName = "rdr-status.toml"

// routingModelNames is every routing model in this repo that matches on
// fact names. Both bind to `rdr-facts.toml` the same way and neither
// binary calls the other, so a check that covered only the first would
// leave the second free to drift — and the write model carries its own
// copy of the status vocabulary plus `readme_status`'s.
var routingModelNames = []string{"rdr-status.toml", "rdr-write.toml"}

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
		Tags:  map[string][]string{},
		Kinds: map[string]string{},
		Emits: map[string]map[string]string{},
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

// splitTOMLList reads a single-line `["a", "b"]` literal.
func splitTOMLList(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s[1:len(s)-1], ",") {
		if p := strings.Trim(strings.TrimSpace(part), `"`); p != "" {
			out = append(out, p)
		}
	}
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
var routingFixtures = []string{"0020", "0021", "0022", "0023", "0024", "0025"}

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
		if f.HasAbsent {
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
