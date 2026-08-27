package model

// Reading TEMPLATE.md.
//
// TEMPLATE.md is the only schema an RDR has, and it is prose. Every
// consumer that needed its section names, levels, classes, vocabularies
// and label sets used to restate them by hand, and they drifted — four
// template generations became tribal knowledge because nothing checked.
// The answer was a Go table bound to the file by a test that parsed the
// file at TEST time. This is that parser, promoted: the binary now reads
// the template itself, so the table it used to check no longer exists to
// disagree with.
//
// What the parser reads is everything TEMPLATE.md states about itself —
// the heading tree with its levels and nesting, the class markers, which
// sections key their items, the metadata and Evidence Record field sets,
// the closed vocabularies, the decision classes, and the gate items with
// their keys and retention. What it cannot read has no home here: see
// the sidecar (schema.go).

import (
	"fmt"
	"regexp"
	"strings"
)

// A section's class is declared by the bracket text in its own body.
// delegatedMarker is the exception: a `[Conditional scaffold — …]` clause
// declares a CHILD block's class rather than the section's own —
// Alternatives Considered carries one on behalf of `Alternative 1` — so
// reading it as a self-declaration would make a Required spine section
// read as Conditional.
var (
	requiredMarker    = regexp.MustCompile(`\[Required\b`)
	conditionalMarker = regexp.MustCompile(`\[Conditional\b`)
	delegatedMarker   = regexp.MustCompile(`\[Conditional scaffold\b`)
	retainedMarker    = regexp.MustCompile(`\[Retained at lock\b`)
	// gateKeyMarker declares a gate sub-section's citation key. The key
	// is not derived from the heading — `Contradiction Check` keys
	// `contradiction`, not `contradiction-check` — because peers cite
	// gate responses as `cli/NNNN:G-<key>` and a heading may be reworded
	// without moving the id.
	gateKeyMarker = regexp.MustCompile(`\[Gate key:\s*([a-z][a-z0-9-]*)\b`)
)

// numberedItem is a list item the author numbers: `1. **Scenario**:`.
var numberedItem = regexp.MustCompile(`^\s*\d+[.)]\s`)

// boldKeyLead is a bold lead that names a key rather than a placeholder:
// `- **A1 [Statement]**`, `**C1**`, `- **Identity** — …`. A lead whose
// bold text OPENS with `[` is the template's placeholder for the author's
// own words (`- **[Alternative N]**:`) and keys nothing.
var boldKeyLead = regexp.MustCompile(`^\s*(?:[-*]\s+)?\*\*([^*]+)\*\*`)

// decisionBullet is a Load-Bearing Decisions class bullet.
var decisionBullet = regexp.MustCompile(`^- \*\*([^*]+)\*\*`)

// methodLabel is a Method definition in README.md's authoritative list.
var methodLabel = regexp.MustCompile(`(?m)^- \*\*([^*]+)\*\* —`)

// templateSection is one heading of TEMPLATE.md with what its own body
// declares about it.
type templateSection struct {
	Name  string
	Level int
	Class Class
	// ClassExplicit records whether the class came from a bracket marker
	// or from the unmarked-means-Required default. Only an unmarked
	// section may be re-classed by a parent's scaffold clause.
	ClassExplicit bool
	Keys          bool
	Retained      bool
	GateKey       string
	Body          []string
}

// parseSections walks TEMPLATE.md's heading tree, reading each section's
// level, class, key-ness and markers out of its own body.
//
// Fenced code and HTML comments are skipped so that a marker quoted in
// guidance is not mistaken for a declaration, and a level-1 heading is
// the record title rather than a section.
func parseSections(src string) ([]templateSection, error) {
	var out []templateSection
	var body []string

	flush := func() {
		if len(out) == 0 {
			return
		}
		s := &out[len(out)-1]
		s.Body = body
		s.Keys = sectionKeys(body)
		s.Retained = retainedMarker.MatchString(strings.Join(body, "\n"))
		if m := gateKeyMarker.FindStringSubmatch(strings.Join(body, "\n")); m != nil {
			s.GateKey = m[1]
		}
		// A `[Conditional scaffold` clause in a section's own body
		// declares THAT section a per-instance slot. The same clause in a
		// PARENT's body speaks on behalf of a child it names in prose —
		// Alternatives Considered carries one for `Alternative 1` — and
		// reading that as a self-declaration would make a Required spine
		// section read as Conditional. The two are told apart by whether
		// the clause is the section's own opening marker: a scaffold
		// declares itself first, a delegating parent says it mid-body
		// after its own guidance.
		joined := strings.Join(body, "\n")
		text := joined
		if !declaresItselfScaffold(body) {
			text = delegatedMarker.ReplaceAllString(joined, "")
		}
		switch {
		case conditionalMarker.MatchString(text):
			s.Class, s.ClassExplicit = Conditional, true
		case requiredMarker.MatchString(text):
			s.Class, s.ClassExplicit = Required, true
		default:
			s.Class, s.ClassExplicit = Required, false
		}
		body = nil
	}

	inFence, inComment := false, false
	for _, line := range strings.Split(src, "\n") {
		if FenceDelimiter.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if !inComment && strings.Contains(line, "<!--") && !strings.Contains(line, "-->") {
			inComment = true
			continue
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		if strings.Contains(line, "<!--") && strings.Contains(line, "-->") {
			continue
		}

		if m := Heading.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			if level == 1 {
				continue // the record title, not a section
			}
			flush()
			out = append(out, templateSection{Name: m[2], Level: level})
			continue
		}
		body = append(body, line)
	}
	flush()

	if len(out) == 0 {
		return nil, fmt.Errorf("no sections parsed — the heading grammar or the file layout changed")
	}
	return out, nil
}

// declaresItselfScaffold reports whether the section OPENS with a
// scaffold clause, which is how a slot declares its own class rather
// than a parent declaring a child's.
func declaresItselfScaffold(body []string) bool {
	for _, l := range body {
		if strings.TrimSpace(l) == "" {
			continue
		}
		return delegatedMarker.MatchString(l)
	}
	return false
}

// sectionKeys reports whether a section's template body shows its items
// carrying a key the projector can read back as an id.
//
// Grammar cannot answer this — Testing Strategy, Briefly Rejected and
// Failure Modes all carry prose, and only the first numbers its items —
// which is why it is read from the body rather than inferred.
func sectionKeys(body []string) bool {
	for _, l := range body {
		if numberedItem.MatchString(l) {
			return true
		}
		if m := boldKeyLead.FindStringSubmatch(l); m != nil {
			if !strings.HasPrefix(strings.TrimSpace(m[1]), "[") {
				return true
			}
		}
	}
	return false
}

// parentOf assigns each section the nearest preceding heading one level
// shallower. The template states nesting structurally, by heading level;
// this is that nesting named.
func parentOf(sections []templateSection) []string {
	parents := make([]string, len(sections))
	var stack [7]string
	for i, s := range sections {
		if s.Level > 2 {
			parents[i] = stack[s.Level-1]
		}
		if s.Level < len(stack) {
			stack[s.Level] = s.Name
		}
	}
	return parents
}

// templateLabels reads every `- **Label**:` bullet, at any indent, keyed
// by the nearest heading above it. This is the labelled-bullet half of
// the schema: the Metadata block, the Evidence Record, and each section's
// canonical label set.
func templateLabels(src string) map[string][]string {
	out := map[string][]string{}
	section, fenced := "", false
	for _, line := range strings.Split(src, "\n") {
		if FenceDelimiter.MatchString(line) {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := Heading.FindStringSubmatch(line); m != nil {
			section = m[2]
			continue
		}
		if m := EvidenceFieldBullet.FindStringSubmatch(line); m != nil {
			out[section] = append(out[section], strings.TrimSpace(m[1]))
		}
	}
	return out
}

// pipeValues reads a `Label | Label | Label` vocabulary line.
//
// The three metadata vocabulary lines each end differently: Status ends
// at the HTML comment that documents it, Type at the next bullet, and
// Profile mid-line at an em-dash introducing prose ("foundational — value
// only, plus one clause…"). Taking whichever terminator comes first
// handles all three, so there is one parser rather than three that can
// drift apart.
func pipeValues(src, label string) ([]string, error) {
	i := strings.Index(src, "- **"+label+"**:")
	if i < 0 {
		return nil, fmt.Errorf("no %s metadata field", label)
	}
	rest := src[i+len("- **"+label+"**:"):]
	for _, end := range []string{"<!--", "\n- **", "—"} {
		if j := strings.Index(rest, end); j > 0 {
			rest = rest[:j]
		}
	}
	var out []string
	for _, part := range strings.Split(strings.Join(strings.Fields(rest), " "), "|") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s declares no values", label)
	}
	return out, nil
}

// assumptionStatuses reads the Evidence Record's own Status line, which
// is an INDENTED bullet under an assumption rather than a metadata field.
func assumptionStatuses(src string) ([]string, error) {
	for _, line := range strings.Split(src, "\n") {
		m := EvidenceFieldBullet.FindStringSubmatch(line)
		if m == nil || strings.TrimSpace(m[1]) != "Status" || !strings.HasPrefix(line, " ") {
			continue
		}
		var out []string
		for _, p := range strings.Split(m[2], "|") {
			if v := strings.TrimSpace(p); v != "" {
				out = append(out, v)
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("no indented Evidence Record Status bullet")
}

// bodyOf returns the lines of the named section, by exact heading text.
func bodyOf(sections []templateSection, name string) []string {
	for _, s := range sections {
		if s.Name == name {
			return s.Body
		}
	}
	return nil
}

// decisionClasses reads the Load-Bearing Decisions class bullets. A
// decision whose bold label opens with one of these is keyed by the class
// (`D-identity`); any other label is the author's own and gets a derived
// key.
func decisionClasses(sections []templateSection) []string {
	var out []string
	for _, l := range bodyOf(sections, "Load-Bearing Decisions") {
		if m := decisionBullet.FindStringSubmatch(l); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

// gateItems reads the Finalization Gate's sub-sections with the key and
// retention each declares.
//
// Which item survives the lock, and what each is cited as, are the
// template's to say: a rule the reader spelled out for itself would be a
// second source for one fact, and the two would drift.
func gateItems(sections []templateSection) ([]GateItem, error) {
	parents := parentOf(sections)
	var out []GateItem
	for i, s := range sections {
		if parents[i] != "Finalization Gate" {
			continue
		}
		if s.GateKey == "" {
			return nil, fmt.Errorf("gate sub-section %q declares no [Gate key: …] marker", s.Name)
		}
		out = append(out, GateItem{Section: s.Name, Key: s.GateKey, Retained: s.Retained})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the Finalization Gate declares no sub-sections")
	}
	return out, nil
}

// methodLabels reads README.md's "Verifying load-bearing claims", which
// declares itself authoritative for the Method vocabulary precisely so
// the guidance does not ship inside the template body.
func methodLabels(readme string) ([]string, error) {
	start := strings.Index(readme, "### Verifying load-bearing claims")
	if start < 0 {
		return nil, fmt.Errorf("README.md has no 'Verifying load-bearing claims' section; the Method vocabulary lost its home")
	}
	section := readme[start:]
	if end := strings.Index(section, "\n## "); end > 0 {
		section = section[:end]
	}
	var out []string
	for _, m := range methodLabel.FindAllStringSubmatch(section, -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("README.md's Method section declares no labels")
	}
	return out, nil
}
