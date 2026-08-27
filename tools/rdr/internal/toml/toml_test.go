package toml

import "testing"

// The subset's whole point is that a line outside it is REFUSED, never
// skipped: a parser that skips what it cannot read turns a typo into a
// missing fact, and a missing fact reads as absent when it was only
// misspelled. These pin that, because the doc comment claiming it is not
// a test.
func TestParseRefusesWhatIsOutsideTheSubset(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"dotted key", "[t]\na.b = \"x\"\n", "dotted or quoted keys"},
		{"quoted key", "[t]\n\"a\" = \"x\"\n", "dotted or quoted keys"},
		{"array of tables", "[[t]]\na = \"x\"\n", "array-of-tables"},
		{"unterminated header", "[t\na = \"x\"\n", "unterminated table header"},
		{"empty header", "[]\n", "empty table header"},
		{"key above any table", "a = \"x\"\n", "sits above every table header"},
		{"duplicate key", "[t]\na = \"x\"\na = \"y\"\n", "set twice"},
		{"empty key", "[t]\n = \"x\"\n", "empty key"},
		{"not a key or header", "[t]\nbare\n", "not a key = value"},
		{"unterminated list", "[t]\na = [\"x\"\n", "unterminated list"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.src)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded; a line outside the subset must be an error, never a skip", c.src)
			}
			if !contains(err.Error(), c.want) {
				t.Errorf("Parse(%q) = %v, want an error mentioning %q", c.src, err, c.want)
			}
		})
	}
}

// Every refusal names the line it refused, so a table author is pointed
// at the typo rather than left to find it.
func TestRefusalNamesTheLine(t *testing.T) {
	_, err := Parse("[t]\na = \"x\"\n\nb.c = \"y\"\n")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !contains(err.Error(), "line 4") {
		t.Errorf("error = %v, want it to name line 4", err)
	}
}

func TestParseReadsTheSubset(t *testing.T) {
	tables, err := Parse(`# a comment
[facts]
version = 1
description = "the table"

[fact.status]
kind = "field"
paths = ["a", "b"]
absent = ""
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}
	// Declaration order is preserved: a consumer switching on Name reads
	// them in the order the file writes them.
	if tables[0].Name != "facts" || tables[1].Name != "fact.status" {
		t.Errorf("names = %q, %q", tables[0].Name, tables[1].Name)
	}
	if got := tables[0].Int("version"); got != 1 {
		t.Errorf("version = %d, want 1", got)
	}
	if got := tables[1].List("paths"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("paths = %v", got)
	}
	// Str collapses absent and empty; Scalar tells them apart, which is
	// what `absent = ""` needs to be a real declaration.
	if v, ok := tables[1].Scalar("absent"); !ok || v != "" {
		t.Errorf("Scalar(absent) = %q, %v; want \"\", true", v, ok)
	}
	if _, ok := tables[1].Scalar("nope"); ok {
		t.Error("Scalar(nope) reported set")
	}
	// A list asked for as a scalar is not silently flattened.
	if got := tables[1].Str("paths"); got != "" {
		t.Errorf("Str(paths) = %q, want \"\" — a list is not a scalar", got)
	}
	if got := tables[1].List("kind"); got != nil {
		t.Errorf("List(kind) = %v, want nil — a scalar is not a list", got)
	}
	if got := tables[1].Keys(); len(got) != 3 || got[0] != "kind" {
		t.Errorf("Keys() = %v, want them as written", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
