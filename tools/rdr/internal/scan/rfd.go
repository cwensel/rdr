package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RFD is a capability document as the projector reads it: the anchors a
// record may cite. Like a Registry it is deliberately NOT a Document —
// an RFD is human-authored prose whose sections are numbered by hand, and
// running it through the record scanner would mint element ids for a
// document that has none.
type RFD struct {
	// Number is the four-digit RFD number.
	Number string
	// Path is the file read.
	Path string
	// Anchors holds every citable anchor, lowercased: section numbers
	// (`3c`), principle ids (`p-2`), and the legacy decision rows
	// (`dx-13`, `decision-1`).
	Anchors map[string]bool
}

// rfdSection matches a numbered heading. The corpus writes the separator
// four ways — `### 3c. How the mechanisms compose`, `## 1 Getting data
// in`, `### 4b — What carries the structure`, `### 4e` alone — so the
// separator is optional and the em-dash counts. Requiring a period
// missed `4b`, which 42 citations name and which exists.
//
// The NUMBER is the anchor; the prose after it is free to change, which
// is the whole point of the tier — prose is mutable, the citable id is
// not.
var rfdSection = regexp.MustCompile(`(?m)^#{2,4}\s+([0-9]+[a-z]?)\s*(?:[.)]|[—–-]|\s|$)`)

// rfdPrinciple matches a principle: `- **P-2** MUST …`, `## P-2 …`.
var rfdPrinciple = regexp.MustCompile(`(?m)^(?:#{2,4}\s+|[-*]\s+(?:\*\*)?)(P-\d+[a-z]?)\b`)

// rfdDecision matches the legacy decision rows an RFD collected before
// the JDR class existed: a `| DX-13 |` table row, a `**DX-13**` bullet,
// or a `### Decision 1` heading. These are exactly what a registry takes
// over via `inherits`, so they stay readable until it does.
var rfdDecision = regexp.MustCompile(
	`(?m)^(?:\|\s*[` + "`" + `*]*(DX-\d+[a-z]?)` +
		`|#{2,4}\s+Decision\s+(\d+[a-z]?)\b` +
		`|[-*]\s+\*\*(DX-\d+[a-z]?)\*\*)`)

// LoadRFDs reads every RFD under root. Like LoadRegistries it reports
// whether a tree was READ, so an unbound root leaves citations unchecked
// rather than reporting every one of them as dangling.
func LoadRFDs(root string) ([]*RFD, bool) {
	if strings.TrimSpace(root) == "" {
		return nil, false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	var out []*RFD
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		// An RFD is `rfd/NNNN/README.md` or `rfd/NNNN-slug.md`. The dir
		// form exists SO an RFD can carry companion artifacts beside it
		// (`rfd/0007/advisory-inventory.md`), so only the README is the
		// document: taking every .md in the dir loaded the artifact as a
		// second RFD 0007 with no anchors, which overwrote the real one
		// and reported 42 live citations of an existing section as
		// dangling.
		base := filepath.Base(path)
		num := registryNumber(base)
		if num == "" && strings.EqualFold(base, "README.md") {
			num = rfdDirNumber(filepath.Dir(path))
		}
		if num == "" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		out = append(out, parseRFD(string(body), num, path))
		return nil
	})
	return out, true
}

var rfdDir = regexp.MustCompile(`^(\d{3,4})$`)

func rfdDirNumber(dir string) string {
	if m := rfdDir.FindStringSubmatch(filepath.Base(dir)); m != nil {
		return m[1]
	}
	return ""
}

func parseRFD(body, num, path string) *RFD {
	r := &RFD{Number: num, Path: path, Anchors: map[string]bool{}}
	_, rest := splitFrontmatter(body)
	for _, m := range rfdSection.FindAllStringSubmatch(rest, -1) {
		r.Anchors[strings.ToLower(m[1])] = true
	}
	for _, m := range rfdPrinciple.FindAllStringSubmatch(rest, -1) {
		r.Anchors[strings.ToLower(m[1])] = true
	}
	for _, m := range rfdDecision.FindAllStringSubmatch(rest, -1) {
		switch {
		case m[1] != "":
			r.Anchors[strings.ToLower(m[1])] = true
		case m[2] != "":
			r.Anchors["decision-"+strings.ToLower(m[2])] = true
		case m[3] != "":
			r.Anchors[strings.ToLower(m[3])] = true
		}
	}
	return r
}

// Has reports whether an anchor resolves against this RFD.
func (r *RFD) Has(anchor string) bool {
	return r.Anchors[strings.ToLower(strings.TrimPrefix(anchor, "§"))]
}
