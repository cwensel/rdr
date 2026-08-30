package scan

import "testing"

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
