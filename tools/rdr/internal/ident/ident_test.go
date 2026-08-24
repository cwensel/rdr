package ident

import (
	"strings"
	"testing"
)

func TestParseRoundTrip(t *testing.T) {
	cases := []struct {
		in   string
		want ID
	}{
		{"0055:A3", ID{Record: "0055", Kind: Assumption, Key: "3"}},
		{"0055:C4", ID{Record: "0055", Kind: Contract, Key: "4"}},
		{"0055:D-identity", ID{Record: "0055", Kind: Decision, Key: "identity"}},
		{"0055:RT1", ID{Record: "0055", Kind: RoundTrip, Key: "1"}},
		{"0055:ALT2", ID{Record: "0055", Kind: Alternative, Key: "2"}},
		{"0055:BR3", ID{Record: "0055", Kind: Rejected, Key: "3"}},
		{"0055:S5", ID{Record: "0055", Kind: Scenario, Key: "5"}},
		{"0055:MVV", ID{Record: "0055", Kind: MVV}},
		{"0055:F2", ID{Record: "0055", Kind: Failure, Key: "2"}},
		{"0055:G-contradiction", ID{Record: "0055", Kind: Gate, Key: "contradiction"}},
		{"0055:§normative-contracts", ID{Record: "0055", Kind: Section, Key: "normative-contracts"}},
		{"cli/0055:C4", ID{Project: "cli", Record: "0055", Kind: Contract, Key: "4"}},
		{"my.proj-2/0001:§approach", ID{Project: "my.proj-2", Record: "0001", Kind: Section, Key: "approach"}},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
		if got.String() != c.in {
			t.Errorf("Parse(%q).String() = %q: does not round-trip", c.in, got.String())
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{
		"", "0055", "55:A3", "0055:A", "0055:X1", "0055:D-", "0055:D-Identity",
		"0055 A3", "cli/0055 A5", "0055:§Normative Contracts", "0055:MVV1",
		"/0055:C4", "0055:ALT", "0055:§",
	} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) accepted; want ErrSyntax", in)
		}
	}
	// Surrounding whitespace is tolerated; interior whitespace is not.
	if _, err := Parse("  0055:C4\n"); err != nil {
		t.Errorf("surrounding whitespace rejected: %v", err)
	}
}

func TestQualification(t *testing.T) {
	id, _ := Parse("cli/0055:C4")
	if id.Local() != "0055:C4" {
		t.Errorf("Local() = %q", id.Local())
	}
	if id.Qualified("other") != "other/0055:C4" {
		t.Errorf("Qualified() = %q", id.Qualified("other"))
	}
	if New("", "0055", Decision, "Wire / byte format").String() != "0055:D-wire-byte-format" {
		t.Errorf("New slugs a keyed kind: %q", New("", "0055", Decision, "Wire / byte format"))
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Round-Trip / Inverse Invariants":       "round-trip-inverse-invariants",
		"Alternative 2: A cryptographic digest": "alternative-2-a-cryptographic-digest",
		"**Identity** — what":                   "identity-what",
		"`parse ∘ render`":                      "parse-render",
		"  Trailing punctuation!  ":             "trailing-punctuation",
		"Épreuve":                               "épreuve",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHashIgnoresLayoutOnly(t *testing.T) {
	a := Hash([]string{"- **Identity** — the key", "  is the pair"})
	b := Hash([]string{"", "    - **Identity** — the key   ", "", "\tis the pair", ""})
	c := Hash([]string{"- **Identity** — the key", "  is the triple"})
	if a != b {
		t.Errorf("reindent/blank lines changed the hash: %s vs %s", a, b)
	}
	if a == c {
		t.Errorf("a content change left the hash alone: %s", a)
	}
	if len(a) != 8 || strings.Trim(a, "0123456789abcdef") != "" {
		t.Errorf("hash is not 8 hex chars: %q", a)
	}
}

func TestRecordOf(t *testing.T) {
	for in, want := range map[string]string{
		"0055-frame-checksum.md": "0055", "0055.md": "0055", "0055": "0055",
		"00555-x.md": "", "README.md": "", "a0055-x.md": "",
	} {
		if got := RecordOf(in); got != want {
			t.Errorf("RecordOf(%q) = %q, want %q", in, got, want)
		}
	}
}
