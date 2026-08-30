package scan

import (
	"regexp"
	"strings"
)

// The 07.1 gate's note carries the target stage as prose in records that
// predate the `@<stage>` qualifier slot: `- **Target re-entry stage**: 3
// (refine — …)` or `fix + /rdr-finalize`. Records are never amended, so
// the projector reads that line when the qualifier names no target.
var (
	noteTargetLine   = regexp.MustCompile(`(?i)target re-entry stage\**\s*:\s*\**\s*(.+)$`)
	noteTargetVerb   = regexp.MustCompile(`/rdr-(propose|refine|resolve|finalize)\b`)
	noteTargetNumber = regexp.MustCompile(`^\s*(\d)\b`)
)

// noteReentryTarget reads the re-entry note's TARGET RE-ENTRY STAGE line
// and returns the stage verb it names, or "" when there is no note or the
// line names nothing the grammar knows. The qualifier's `@<stage>`, when
// present, outranks it (see metadataField).
func (d *Document) noteReentryTarget() string {
	for _, n := range d.nodes {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(n.Heading)), "refinement context") {
			continue
		}
		for i := n.LineStart; i <= n.LineEnd && i <= len(d.lines); i++ {
			m := noteTargetLine.FindStringSubmatch(d.lines[i-1])
			if m == nil {
				continue
			}
			return targetFromNoteText(m[1])
		}
	}
	return ""
}

// targetFromNoteText maps the note line's prose to a stage verb: a
// `/rdr-<stage>` command wins, then the leading stage number (2 propose,
// 3 refine, 4 resolve, 7 finalize).
func targetFromNoteText(s string) string {
	if m := noteTargetVerb.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	if m := noteTargetNumber.FindStringSubmatch(s); m != nil {
		switch m[1] {
		case "2":
			return "propose"
		case "3":
			return "refine"
		case "4":
			return "resolve"
		case "7":
			return "finalize"
		}
	}
	return ""
}
