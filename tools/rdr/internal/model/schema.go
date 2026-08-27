package model

// Loading the schema.
//
// One call assembles what the reader needs: TEMPLATE.md parsed for
// everything it states about itself, README.md for the Method vocabulary
// it declares itself authoritative for, and the sidecar for the three
// things neither document can carry.
//
// Failure is a single categorised `stopped:` error, never a list. The
// caller cannot proceed without a schema, so the first thing wrong is
// the only thing worth reporting.

import (
	"fmt"
	"os"
	"path/filepath"
)

// Schema is the current TEMPLATE.md plus what it cannot say about itself.
type Schema struct {
	Table    TemplateTable
	Sidecar  *Sidecar
	Source   string // the template path, for error messages
	Readme   string // the README path
	Sections []templateSection
	// TemplateSource is the template's bytes, kept for the readers that
	// re-scan it rather than a projected column.
	TemplateSource string
}

// Bind installs the loaded schema for this process, and current is every
// reader's one door to it.
//
// The alternative — threading a *Schema through every call — was measured
// rather than assumed: the readers are 13 free functions across four
// packages (scan, lint, edge and this one), none with a schema in scope,
// reached from 124 test call sites. Changing all of them to carry a
// parameter, in a change whose acceptance is byte-identical output, is a
// large surface for no behavioural gain. So the readers keep their
// signatures and the table behind them stops being a literal.
//
// An unbound schema PANICS rather than answering. A zero schema does not
// fail — it succeeds wrongly, classifying every value off-vocabulary and
// every heading unknown-to-template, so `lint` would report a missing
// install as a corpus-wide defect. The flow's own rule is that a skipped
// check must never read as a passed one, and this is that rule turned on
// the reader itself.
var bound *Schema

func Bind(s *Schema) { bound = s }

// Bound reports whether a schema has been installed.
func Bound() bool { return bound != nil }

func current() *Schema {
	if bound == nil {
		panic("model: schema not bound — call model.Bind(model.Load(...)) before reading a record")
	}
	return bound
}

// MustBindFrom loads the schema from an engine root and installs it.
//
// It exists for tests: a test binary runs with no seam marker bound, and
// every package that reads a record needs a schema before it can. Each
// such package calls this from TestMain.
func MustBindFrom(root string) error {
	s, err := Load(
		filepath.Join(root, "TEMPLATE.md"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "models", "rdr-template.toml"),
	)
	if err != nil {
		return err
	}
	Bind(s)
	return nil
}

// Load reads the schema from the three files that state it.
func Load(templatePath, readmePath, sidecarPath string) (*Schema, error) {
	tmpl, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("stopped:no-template (%s: %v)", templatePath, err)
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, fmt.Errorf("stopped:no-readme (%s: %v)", readmePath, err)
	}
	side, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, fmt.Errorf("stopped:no-template-sidecar (%s: %v)", sidecarPath, err)
	}

	sc, err := ParseSidecar(string(side), sidecarPath)
	if err != nil {
		return nil, err
	}
	s, err := parse(string(tmpl), string(readme), templatePath, readmePath)
	if err != nil {
		return nil, err
	}
	s.Sidecar = sc
	if err := sc.CheckAgainst(s.Table.Sections); err != nil {
		return nil, err
	}
	return s, nil
}

// parse builds the schema from the two documents that state it.
func parse(tmpl, readme, templatePath, readmePath string) (*Schema, error) {
	fail := func(format string, a ...any) error {
		return fmt.Errorf("stopped:malformed-template (%s: %s)", templatePath, fmt.Sprintf(format, a...))
	}

	parsed, err := parseSections(tmpl)
	if err != nil {
		return nil, fail("%v", err)
	}
	parents := parentOf(parsed)

	sections := make([]Section, len(parsed))
	for i, p := range parsed {
		sections[i] = Section{
			Name:   p.Name,
			Level:  p.Level,
			Class:  p.Class,
			Parent: parents[i],
			Keys:   p.Keys,
		}
	}

	labels := templateLabels(tmpl)
	metadata, evidence := labels["Metadata"], labels["Critical Assumptions"]
	if len(metadata) == 0 {
		return nil, fail("the Metadata block declares no fields")
	}
	if len(evidence) == 0 {
		return nil, fail("the Evidence Record declares no fields")
	}

	status, err := pipeValues(tmpl, "Status")
	if err != nil {
		return nil, fail("%v", err)
	}
	typ, err := pipeValues(tmpl, "Type")
	if err != nil {
		return nil, fail("%v", err)
	}
	profile, err := pipeValues(tmpl, "Profile")
	if err != nil {
		return nil, fail("%v", err)
	}
	methods, err := methodLabels(readme)
	if err != nil {
		return nil, fmt.Errorf("stopped:malformed-readme (%s: %v)", readmePath, err)
	}

	return &Schema{
		Table: TemplateTable{
			Sections:       sections,
			MetadataFields: metadata,
			EvidenceFields: evidence,
			Vocabularies: []Vocabulary{
				{Field: "Status", Canonical: status},
				{Field: "Type", Canonical: typ},
				{Field: "Profile", Canonical: profile},
				{Field: "Method", Canonical: methods},
			},
		},
		Source:         templatePath,
		Readme:         readmePath,
		Sections:       parsed,
		TemplateSource: tmpl,
	}, nil
}
