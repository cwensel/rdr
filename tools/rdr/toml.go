package main

// A TOML subset, exactly as wide as the fact table needs.
//
// The binary is Go stdlib only, by design — `rdr-doctor` gets one static
// thing to check for and the build has no dependency to age. TOML is not
// in the stdlib, so reading `models/rdr-facts.toml` means either taking
// a dependency, embedding the table into the binary, or parsing the
// subset the table actually uses.
//
// The subset is the third option and it is small: `[table.name]` headers,
// `key = "string"`, `key = 123`, `key = ["a", "b"]`, `#` comments, blank
// lines. That is the whole grammar. It is not TOML and does not pretend
// to be — no dotted keys, no inline tables, no multi-line strings, no
// datetimes, no escapes beyond the two a path might need.
//
// What matters more than the grammar's width is that every line outside
// it is REFUSED. A parser that skips what it does not understand turns a
// typo into a missing fact, and a missing fact into a signal that reads
// as absent when it was merely misspelled — which is the failure this
// package spends all its care avoiding. So an unparseable line is an
// error naming the line number, never a silent skip.

import (
	"fmt"
	"strconv"
	"strings"
)

// tomlValue is one parsed right-hand side. A value is either a scalar or
// a list; the parser records which, so a fact declaring `paths` cannot
// quietly be handed a single string.
type tomlValue struct {
	scalar string
	list   []string
	isList bool
	line   int
}

// tomlTable is one `[header]` and the keys under it, with the header's
// declaration order preserved by the slice that holds these.
type tomlTable struct {
	name   string
	line   int
	values map[string]tomlValue
	// order is the keys as written, so an error can name them in the
	// order a reader would find them.
	order []string
}

func (t tomlTable) str(key string) string {
	v, ok := t.values[key]
	if !ok || v.isList {
		return ""
	}
	return v.scalar
}

func (t tomlTable) list(key string) []string {
	v, ok := t.values[key]
	if !ok || !v.isList {
		return nil
	}
	return v.list
}

func (t tomlTable) int_(key string) int {
	n, err := strconv.Atoi(t.str(key))
	if err != nil {
		return 0
	}
	return n
}

// parseTOMLSubset reads the subset and returns the tables in file order.
func parseTOMLSubset(src string) ([]tomlTable, error) {
	var tables []tomlTable
	var cur *tomlTable

	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		lineNo := i + 1
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("line %d: unterminated table header", lineNo)
			}
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return nil, fmt.Errorf("line %d: empty table header", lineNo)
			}
			// `[[array]]` is valid TOML this subset does not read. Refusing
			// it names the limitation; skipping it would drop a table.
			if strings.HasPrefix(name, "[") {
				return nil, fmt.Errorf("line %d: array-of-tables is outside this subset", lineNo)
			}
			tables = append(tables, tomlTable{name: name, line: lineNo, values: map[string]tomlValue{}})
			cur = &tables[len(tables)-1]
			continue
		}

		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: not a key = value or a table header: %q", lineNo, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo)
		}
		if strings.ContainsAny(key, ".\"'") {
			return nil, fmt.Errorf("line %d: dotted or quoted keys are outside this subset: %q", lineNo, key)
		}
		if cur == nil {
			return nil, fmt.Errorf("line %d: key %q sits above every table header", lineNo, key)
		}
		if _, dup := cur.values[key]; dup {
			return nil, fmt.Errorf("line %d: key %q is set twice in [%s]", lineNo, key, cur.name)
		}
		value := strings.TrimSpace(rest)
		// A list may wrap across lines — the readable way to write six
		// paths. Join until the bracket closes, so the value the parser
		// sees is the whole list rather than its first line.
		if strings.HasPrefix(value, "[") && !strings.HasSuffix(value, "]") {
			for i+1 < len(lines) {
				i++
				value += " " + strings.TrimSpace(stripComment(lines[i]))
				if strings.HasSuffix(value, "]") {
					break
				}
			}
		}
		v, err := parseValue(value, lineNo)
		if err != nil {
			return nil, err
		}
		cur.values[key] = v
		cur.order = append(cur.order, key)
	}
	return tables, nil
}

// parseValue reads a scalar or a list.
func parseValue(s string, lineNo int) (tomlValue, error) {
	if s == "" {
		return tomlValue{}, fmt.Errorf("line %d: no value", lineNo)
	}
	if strings.HasPrefix(s, "[") {
		if !strings.HasSuffix(s, "]") {
			// The caller joins a wrapped list before calling in, so an
			// unclosed bracket here means it never closed at all.
			return tomlValue{}, fmt.Errorf("line %d: unterminated list", lineNo)
		}
		body := strings.TrimSpace(s[1 : len(s)-1])
		out := []string{}
		if body != "" {
			for _, part := range splitList(body) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				lit, err := parseScalar(part, lineNo)
				if err != nil {
					return tomlValue{}, err
				}
				out = append(out, lit)
			}
		}
		return tomlValue{list: out, isList: true, line: lineNo}, nil
	}
	lit, err := parseScalar(s, lineNo)
	if err != nil {
		return tomlValue{}, err
	}
	return tomlValue{scalar: lit, line: lineNo}, nil
}

// parseScalar reads a quoted string, a bare integer, or a bare boolean.
func parseScalar(s string, lineNo int) (string, error) {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		body := s[1 : len(s)-1]
		if strings.Contains(body, `"`) {
			return "", fmt.Errorf("line %d: an embedded quote is outside this subset: %s", lineNo, s)
		}
		// The two escapes a declared path or a description might carry.
		return strings.NewReplacer(`\\`, `\`, `\n`, "\n").Replace(body), nil
	}
	if s == "true" || s == "false" {
		return s, nil
	}
	if _, err := strconv.Atoi(s); err == nil {
		return s, nil
	}
	return "", fmt.Errorf("line %d: a value is a quoted string, an integer or a boolean; got %s", lineNo, s)
}

// splitList splits on commas that sit outside quotes, so a description
// or a path containing a comma survives.
func splitList(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			inQuote = !inQuote
			b.WriteByte(c)
		case c == ',' && !inQuote:
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	out = append(out, b.String())
	return out
}

// stripComment removes a `#` comment, respecting quotes so a `#` inside
// a string is content.
func stripComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return s[:i]
			}
		}
	}
	return s
}
