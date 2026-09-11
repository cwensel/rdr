package scan

import (
	"testing"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
)

func reentryDoc(status, note string) *Document {
	src := "# RDR 0999: x\n\n## Metadata\n\n- **Status**: " + status + "\n- **Date**: 2026-08-01\n\n## Problem Statement\n\nx\n\n## References\n\n- none\n" + note
	return Bytes([]byte(src), Options{})
}

// TestReentryTargetFallsBackToTheNote: a record demoted before the
// `@<stage>` slot existed carries its target only in the 07.1 note.
// Records are never amended, so the projector reads the note line when
// the qualifier names nothing — and the qualifier wins when it does.
func TestReentryTargetFallsBackToTheNote(t *testing.T) {
	note := "\n## Refinement Context (cluster re-entry — delete on re-lock)\n\n- **Cluster + date**: x\n- **Target re-entry stage**: %s\n"
	for _, tc := range []struct{ status, line, want string }{
		{"Draft [revised from Final 2026-08-29 cluster-reconcile; re-verify none — wording fixes]", "fix + /rdr-finalize. **Re-entry scope**: RE-LOCK-ONLY", "finalize"},
		{"Draft [revised from Final 2026-08-29; re-verify A2 — a contract edit]", "3 (refine — bounded contract-arm rewrites)", "refine"},
		{"Draft [revised from Final 2026-08-29; re-verify A2 — x]", "2 (approach)", "propose"},
		{"Draft [revised from Final 2026-08-29; re-verify A2 — x]", "4 (re-resolve)", "resolve"},
		{"Draft [revised from Final 2026-08-29; re-verify A2 @resolve — x]", "3 (refine)", "resolve"},
		{"Draft [revised from Final 2026-08-29; re-verify A2 — x]", "unclear", ""},
		{"Draft", "3 (refine)", ""},
	} {
		d := reentryDoc(tc.status, fmtNote(note, tc.line))
		f := d.MetadataField("Status")
		if f == nil || f.Status == nil {
			t.Fatalf("%q: no Status field projected", tc.status)
		}
		if f.Status.ReentryTarget != tc.want {
			t.Errorf("%q + note %q: reentry_target = %q, want %q", tc.status, tc.line, f.Status.ReentryTarget, tc.want)
		}
	}
	d := reentryDoc("Draft [revised from Final 2026-08-29; re-verify A2 — x]", "")
	if got := d.MetadataField("Status").Status.ReentryTarget; got != "" {
		t.Errorf("no note: reentry_target = %q, want absent", got)
	}
}

func fmtNote(tmpl, line string) string { return replaceOnce(tmpl, "%s", line) }

func replaceOnce(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestRoutedBackProjectsTheSameSlot: the route-back qualifier fills the
// SAME `reentry_target` the demotion fills, which is the whole point of
// reusing the `@<stage>` slot — the navigator routes both forms through
// rows it already had. It also reaches the two stages only a live Draft
// can return to, and it does NOT fall back to the 7.1 note: that note is
// a demotion's artifact, and a route-back that named no target named
// none.
func TestRoutedBackProjectsTheSameSlot(t *testing.T) {
	for _, tc := range []struct{ status, form, want string }{
		{"Draft [routed back from resolve 2026-09-11; re-verify A5,A6 @propose — the approach is refuted]", "routed-back", "propose"},
		{"Draft [routed back from finalize 2026-09-11; re-verify none @prelock — determinacy never ran]", "routed-back", "prelock"},
		{"Draft [routed back from cluster-reconcile 2026-09-11; @reconcile — a spike is open]", "routed-back", "reconcile"},
		{"Draft [routed back from resolve 2026-09-11; re-verify A2 — no target named]", "routed-back", ""},
		{"Draft [routed back from propose 2026-09-11; @refine — propose sends nothing back]", "bracketed", ""},
	} {
		f := reentryDoc(tc.status, "").MetadataField("Status")
		if f == nil || f.Status == nil {
			t.Fatalf("%q: no Status field projected", tc.status)
		}
		if f.Status.Value != "Draft" {
			t.Errorf("%q: status value = %q, want Draft — the marker rides on a live Draft", tc.status, f.Status.Value)
		}
		if f.Status.Form != tc.form {
			t.Errorf("%q: form = %q, want %q", tc.status, f.Status.Form, tc.form)
		}
		if f.Status.ReentryTarget != tc.want {
			t.Errorf("%q: reentry_target = %q, want %q", tc.status, f.Status.ReentryTarget, tc.want)
		}
	}

	// The note fallback is the demotion's alone.
	note := "\n## Refinement Context (cluster re-entry — delete on re-lock)\n\n- **Cluster + date**: x\n- **Target re-entry stage**: %s\n"
	d := reentryDoc("Draft [routed back from resolve 2026-09-11; re-verify A2 — no target named]", fmtNote(note, "3 (refine)"))
	if got := d.MetadataField("Status").Status.ReentryTarget; got != "" {
		t.Errorf("route-back with a 7.1 note: reentry_target = %q, want absent — the note is a demotion's artifact", got)
	}
}

// TestRoutedBackMintsReverifyEdges: the route-back's `re-verify A5,A6`
// means what the demotion's means — this record's own assumptions, owed
// another look — so it mints the same self-edges the scoped stage works
// through. `re-verify none` and an omitted clause both mint nothing:
// neither names an assumption to re-open.
func TestRoutedBackMintsReverifyEdges(t *testing.T) {
	reverifyTargets := func(status string) []string {
		var out []string
		for _, e := range reentryDoc(status, "").Edges {
			if e.Kind == edge.Reverify {
				out = append(out, e.To)
			}
		}
		return out
	}
	for status, want := range map[string]int{
		"Draft [routed back from resolve 2026-09-11; re-verify A5,A6 @propose — refuted]":  2,
		"Draft [routed back from resolve 2026-09-11; re-verify A5 @propose — refuted]":     1,
		"Draft [routed back from resolve 2026-09-11; re-verify none @propose — refuted]":   0,
		"Draft [routed back from resolve 2026-09-11; @propose — refuted]":                  0,
		"Draft [revised from Final 2026-09-11; re-verify A5,A6 @propose — still the form]": 2,
	} {
		if got := reverifyTargets(status); len(got) != want {
			t.Errorf("%q: %d reverify edges %v, want %d", status, len(got), got, want)
		}
	}
}
