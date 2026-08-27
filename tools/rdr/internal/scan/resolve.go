package scan

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/cwensel/rdr/tools/rdr/internal/edge"
	"github.com/cwensel/rdr/tools/rdr/internal/ident"
)

// This file decides `resolved`: does an edge's target actually exist?
//
// It is a SEPARATE PASS from extraction, and separate on purpose. One
// file cannot answer the question — a record citing `0055:A9` knows
// nothing about whether 0055 has an A9 — so resolution needs the records
// dir, and source-anchor resolution needs the repo. Folding it into the
// scanner would make scanning depend on a corpus, and `rdr inspect
// path/to/one.md` would stop working on a loose file.
//
// UNCHECKED IS NOT UNRESOLVED. `resolved` is three-valued: true, false,
// or absent. Absent means nothing looked — no records dir was given, or a
// source anchor was read with no repo to grep. An unchecked edge must
// never read as a broken one (a false finding a consumer will chase) nor
// as a sound one (a skipped check reading as a pass, which is the failure
// this whole flow is built to prevent).

// Resolver decides whether an edge's target exists. It is built once over
// a records dir and, optionally, a repo, and then applied to any number
// of documents.
type Resolver struct {
	// records maps a record number to the projection of that record, for
	// every record in the dir. Element resolution is a lookup in the
	// target's own projection, which is why the resolver holds documents
	// rather than filenames.
	records map[string]*Document
	// slugs maps a record number to its filename slug, so a reference
	// whose number and slug name two different records is caught.
	slugs map[string]string
	// repo is the root a source anchor's symbol is grepped under, or ""
	// when no repo was given and symbol resolution is not attempted.
	repo string
	// symbols caches the grep verdict per symbol.
	symbols map[string]bool
	// onRead counts source files the walk reads. Some costs here are only
	// assertable as a COUNT: a facet that must not walk the tree, and a
	// primed pass that must walk it once, both produce the same verdicts
	// as the versions that walk repeatedly.
	onRead func()
}

// SourceReads counts source files read by every resolver walk in this
// process. A facet that must not touch the source tree, and a primed pass
// that must touch it once, are both invisible in the OUTPUT — they differ
// from the versions that walk repeatedly only in what they read. This
// counter is how a test says so at the command seam, where the regression
// would actually land.
var SourceReads atomic.Int64

// NewResolver builds a resolver over already-scanned documents. Passing
// the scanned corpus rather than a path keeps the resolver from
// re-reading files an index walk has already read.
func NewResolver(docs []*Document, repo string) *Resolver {
	r := &Resolver{
		records: map[string]*Document{},
		slugs:   map[string]string{},
		repo:    repo,
		symbols: map[string]bool{},
	}
	for _, d := range docs {
		if d.Record == "" {
			continue
		}
		r.records[d.Record] = d
		if d.Path != "" {
			base := strings.TrimSuffix(filepath.Base(d.Path), ".md")
			r.slugs[d.Record] = base
		}
	}
	return r
}

// Resolve decides every edge of a document. Kinds whose target is outside
// both the records dir and the repo — an issue, an RFD, an artifact path
// — are left unchecked: this binary has no tracker to ask, and an
// artifact directory is a per-run location the engine does not know.
func (r *Resolver) Resolve(d *Document) {
	for i := range d.Edges {
		e := &d.Edges[i]
		switch e.Kind.Class() {
		case edge.TargetElement:
			e.Resolved = r.resolveElement(e)
		case edge.TargetSymbol:
			e.Resolved = r.resolveSymbol(e.To)
		}
	}
}

// resolveElement looks the target up in the records dir: the record must
// exist, and when the citation reaches inside it, the element must exist
// in that record's own projection. That is the check that catches a
// Peer-RDR record citing `0055 A9` when 0055 has A1 through A7 — the
// finding class this issue exists to make mechanical.
func (r *Resolver) resolveElement(e *Edge) *bool {
	if len(r.records) == 0 {
		return nil // nothing to resolve against
	}
	id, err := ident.Parse(e.To)
	if err != nil {
		// A document reference: `0055` or `cli/0055`.
		num := e.To
		if _, after, ok := strings.Cut(num, "/"); ok {
			num = after
		}
		if !recordNumber.MatchString(num) {
			return nil // not a record reference; nothing here can judge it
		}
		return r.recordExists(num, e.Slug)
	}
	if ok := r.recordExists(id.Record, e.Slug); ok == nil || !*ok {
		return ok
	}
	target := r.records[id.Record]
	want := ident.ID{Record: id.Record, Kind: id.Kind, Key: id.Key, Project: target.Project}.String()
	for _, el := range target.Elements {
		if el.ID == want {
			return truth(true)
		}
	}
	for _, n := range target.Outline {
		if n.ID == want {
			return truth(true)
		}
	}
	if id.Kind == ident.Section {
		// A bold paragraph lead is addressable by its exact name and by
		// nothing else. The outline's prefix rule reads a HEADING, which
		// the template names and the record repeats; a lead is a sentence
		// the author wrote once, and a prefix of a sentence is not a
		// citation of it. The two corpus forms are quoted and verbatim,
		// so exact is all they need.
		for _, a := range target.Anchors {
			if a.ID == want {
				return truth(true)
			}
		}
		if uniqueSectionPrefix(target, id.Key) != "" {
			return truth(true)
		}
	}
	return truth(false)
}

// uniqueSectionPrefix resolves a section citation whose slug is not the
// whole of any heading's, and returns the heading slug it lands on, or ""
// when it lands on none — or on more than one.
//
// UNIQUENESS IS THE WHOLE RULE. Prefix matching was tried once, without
// it, and reversed: `§Semantic` against a target with five `Semantic *`
// headings is a citation the author under-specified, and picking one of
// the five is the parser guessing. That reversal stands. What is added
// here is narrower and is not a guess: a prefix relation that holds for
// EXACTLY ONE heading of the target identifies that heading the way a
// full slug does. Two candidates and the citation stays unresolved, with
// no first-match or longest-match tiebreak — a tiebreak is the guess by
// another name.
//
// It holds in both directions, because the corpus clips both ways:
//
//	CITATION SHORTER — `§Normative` for `Normative Contracts`, `§No-op
//	operations` for `No-op operations and OpID collision`. The author
//	named the heading by its opening words, or the heading grew a
//	qualifier (`Safety boundary (normative)`) the citation predates.
//
//	CITATION LONGER — `§Failure-Modes residual chartered it as a`. The
//	citation grammar's word window ran off the end of the heading into
//	the sentence about it; the heading is a prefix of what it captured.
//
// Either way one heading of the target is being named and the other is
// prose the boundary could not see. Matching on the SLUG rather than the
// text is what makes the hyphenated spelling (`Failure-Modes`) and the
// spaced one (`Failure Modes`) the same citation for free.
//
// The boundary is a SLUG SEGMENT, never a bare string prefix: `§norm`
// must not reach `normative-contracts`. A prefix relation counts only
// when the shorter slug ends where the longer one has a `-`, so the
// citation named whole words.
//
// THE OVER-READ CASE IS SHORTENED ONE WORD AT A TIME, longest first. A
// citation the word window ran off the end of carries the heading plus a
// tail of the sentence, and where the tail begins is not knowable from
// the citation alone — `§Normative reciprocally assigns the simple`
// stopped mid-clause, so no single suffix rule finds the seam. Trying
// each whole-word prefix from longest to shortest and taking the first
// that names exactly one heading finds it exactly, because the target's
// own headings are what decide. Every length carries the same uniqueness
// guard, so a shortening that turns out to be ambiguous stops the search
// rather than falling through to a guess: `§Semantic contract engine`
// does not become `§Semantic` and pick one of five.
//
// This subsumes any case rule. Stopping at "the first lowercase word
// after a Title-Case run" is a heuristic about English that the corpus
// breaks in both directions — headings carry lowercase words (`No-op
// operations and OpID collision`) and prose carries capitalised ones —
// while asking the target which of its headings the citation names is
// not a heuristic at all.
func uniqueSectionPrefix(target *Document, key string) string {
	for ; key != ""; key = dropLastSegment(key) {
		if hit, ok := uniqueHeading(target, key); ok {
			return hit
		}
	}
	return ""
}

// uniqueHeading reports the target's one heading whose slug stands in a
// prefix relation to key, in either direction, and whether there was
// exactly one. Two or more is not a match: the citation names no one
// heading, and choosing between them is the guess this rule exists to
// refuse.
func uniqueHeading(target *Document, key string) (string, bool) {
	found, n := "", 0
	for _, node := range target.Outline {
		id, err := ident.Parse(node.ID)
		if err != nil || id.Kind != ident.Section || id.Key == "" || id.Key == found {
			continue // a repeated slug is one heading, seen twice
		}
		if id.Key != key && !slugPrefix(key, id.Key) && !slugPrefix(id.Key, key) {
			continue
		}
		found, n = id.Key, n+1
		if n > 1 {
			return "", false
		}
	}
	return found, n == 1
}

// slugPrefix reports whether short is a whole-segment prefix of long.
func slugPrefix(short, long string) bool {
	return len(long) > len(short) && strings.HasPrefix(long, short) && long[len(short)] == '-'
}

// dropLastSegment removes the final `-`-delimited word of a slug, or
// returns "" when there is only one left.
func dropLastSegment(key string) string {
	i := strings.LastIndexByte(key, '-')
	if i < 0 {
		return ""
	}
	return key[:i]
}

var recordNumber = regexp.MustCompile(`^\d{4}$`)

// recordExists reports whether the records dir holds the target, and
// whether the slug the author wrote alongside the number agrees with the
// file that number names. A number and slug naming two different records
// is a stale reference the author copied and half-updated.
func (r *Resolver) recordExists(num, slug string) *bool {
	d, ok := r.records[num]
	if !ok || d == nil {
		return truth(false)
	}
	if slug != "" && r.slugs[num] != "" && !strings.EqualFold(slug, r.slugs[num]) {
		return truth(false)
	}
	return truth(true)
}

// resolveSymbol greps the repo for a `path::Symbol` anchor's symbol —
// tooling-pass CHECK 5's rule exactly: the SYMBOL is what resolves, never
// the line number, and a symbol that resolves elsewhere in the repo is
// still resolved (the anchor moved; the claim stands).
//
// With no repo given, nothing is checked and the edge says so.
func (r *Resolver) resolveSymbol(anchor string) *bool {
	if r.repo == "" {
		return nil
	}
	_, sym, ok := strings.Cut(anchor, "::")
	if !ok || sym == "" {
		return nil
	}
	if v, seen := r.symbols[sym]; seen {
		return truth(v)
	}
	found := r.lookup(sym)
	// A receiver-qualified anchor (`views.go::TableVertex.LiveConstraints`)
	// names one symbol, but no language writes the qualifier adjacent to the
	// member at the definition — Go declares `func (v *TableVertex)
	// LiveConstraints()` and calls it `tv.LiveConstraints()`. Grepping the
	// dotted string whole therefore reports a live method as missing, which
	// is a FALSE finding: worse than the absent verdict a skipped check
	// gives, because a consumer chases it. CHECK 5 asks whether the SYMBOL
	// resolves anywhere, so fall back to the member — the qualifier is the
	// author saying where it lived, and a move is a note, not a finding.
	if !found {
		if member := sym[strings.LastIndex(sym, ".")+1:]; member != sym && member != "" {
			found = r.lookup(member)
		}
	}
	r.symbols[sym] = found
	return truth(found)
}

// lookup is grep behind the symbol cache. The fallback below queries a
// second key (the member of a qualified anchor), and many records cite the
// same member, so an uncached second walk would turn one pass over the repo
// into one per citation.
func (r *Resolver) lookup(sym string) bool {
	if v, seen := r.symbols[sym]; seen {
		return v
	}
	found := r.grep(sym)
	r.symbols[sym] = found
	return found
}

// grep walks the repo for the symbol as a whole word. It is a plain walk
// rather than a shell-out: the binary is stdlib-only, and a walk is both
// portable and free of the quoting hazards of building a grep command
// out of corpus text.
func (r *Resolver) grep(sym string) bool {
	needle := []byte(sym)
	found := false
	r.walk(func(body []byte) bool {
		if containsWord(body, needle) {
			found = true
			return false
		}
		return true
	})
	return found
}

// walk reads every searchable file under the repo once, handing each body
// to visit; visit returns false to stop the walk. Callers that need many
// symbols test them all per file rather than walking per symbol.
func (r *Resolver) walk(visit func(body []byte) bool) {
	done := false
	filepath.WalkDir(r.repo, func(p string, e os.DirEntry, err error) error {
		if err != nil || done {
			return nil
		}
		if e.IsDir() {
			switch e.Name() {
			case ".git", "node_modules", "vendor", "target", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !searchable(p) {
			return nil
		}
		info, err := e.Info()
		if err != nil || info.Size() > maxSearchBytes {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		SourceReads.Add(1)
		if r.onRead != nil {
			r.onRead()
		}
		if !visit(body) {
			done = true
		}
		return nil
	})
}

// maxSearchBytes skips a file too large to be source. A generated blob is
// not where a cited symbol is defined, and reading it costs more than the
// whole rest of the walk.
const maxSearchBytes = 4 << 20

// searchable skips the extensions a symbol is never defined in, so the
// walk reads source rather than every asset in the tree.
func searchable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".gz", ".tar",
		".ico", ".woff", ".woff2", ".ttf", ".mp4", ".bin", ".so", ".dylib", ".exe":
		return false
	}
	return true
}

// containsWord reports whether body holds needle bounded by non-identifier
// bytes on both sides, so `Encode` does not match inside `EncodeAll`.
func containsWord(body, needle []byte) bool {
	if len(needle) == 0 {
		return false
	}
	// bytes.Index jumps to each candidate rather than comparing at every
	// offset; the old scan allocated two strings per byte position, which
	// over a corpus-sized walk (100MB+ x ~1.2k symbols) dominated the run.
	for off := 0; ; {
		i := bytes.Index(body[off:], needle)
		if i < 0 {
			return false
		}
		i += off
		j := i + len(needle)
		if (i == 0 || !identByte(body[i-1])) && (j >= len(body) || !identByte(body[j])) {
			return true
		}
		off = i + 1
	}
}

func identByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func truth(v bool) *bool { return &v }

// Backlinks is the reverse edge set: for every target, the edges pointing
// at it. The corpus index derives inbound relations from this without
// re-parsing a single record, which is one of this issue's acceptance
// criteria — reverse edges are a transposition of the forward set, never
// a second scan.
//
// The key is the target as written on the edge, so a document target
// (`0055`) and an element target (`0055:A3`) are distinct keys: "who
// cites this record?" and "who cites this contract?" are different
// questions and a consumer asks the one it means.
type Backlinks map[string][]Edge

// Reverse transposes the forward edges of a set of documents.
func Reverse(docs []*Document) Backlinks {
	out := Backlinks{}
	for _, d := range docs {
		for _, e := range d.Edges {
			out[e.To] = append(out[e.To], e)
		}
	}
	return out
}

// ResolveAll decides the edges of a whole corpus. It primes the symbol
// cache first: a corpus cites ~1.2k distinct symbols, and one walk per
// symbol is one walk per citation over the same tree. Priming tests every
// symbol against each file as it is read, so the repo is walked ONCE.
func (r *Resolver) ResolveAll(docs []*Document) {
	r.primeSymbols(docs)
	for _, d := range docs {
		r.Resolve(d)
	}
}

// primeSymbols greps every source-anchor symbol the corpus cites in a
// single walk. Each anchor contributes its symbol and, for a qualified
// one, its member (resolveSymbol's fallback), so both cache keys are
// filled before any edge is decided and no later grep walks the tree.
//
// A MISS MUST BE CACHED TOO, and it must be cached as the ANSWER rather
// than as the walk's raw result. Caching hits alone left every absent
// symbol a cache miss, and a miss sends resolveSymbol back to grep: 82 of
// the reference corpus's 1,305 symbols are absent, and each one walked the
// tree again — 260,967 file reads where one pass is 2,023, and 13.7s where
// the priming walk is 6.6s. Absent is not the rare case here; it is what
// resolution is looking for.
//
// But seeding the raw result would answer the qualified fallback before
// the member was ever tried, turning a live method into a FALSE finding —
// `TableVertex.LiveConstraints` is absent as a dotted string in every
// language that declares it as a member, which is the defect 852aee4
// fixed. So the fallback is decided HERE, in the priming pass, where both
// keys are already known: a qualified symbol the walk did not find takes
// its member's verdict. resolveSymbol then finds a decided answer for
// every cited symbol and greps nothing.
func (r *Resolver) primeSymbols(docs []*Document) {
	if r.repo == "" {
		return
	}
	// want is every needle the walk tests; cited is the subset that edges
	// actually ask about. They differ by the members added for the
	// fallback, which are needles but not themselves citations.
	want := map[string][]byte{}
	cited := map[string]bool{}
	for _, d := range docs {
		for i := range d.Edges {
			e := &d.Edges[i]
			if e.Kind != edge.SourceAnchor {
				continue
			}
			_, sym, ok := strings.Cut(e.To, "::")
			if !ok || sym == "" {
				continue
			}
			want[sym] = []byte(sym)
			cited[sym] = true
			if m := sym[strings.LastIndex(sym, ".")+1:]; m != sym && m != "" {
				want[m] = []byte(m)
			}
		}
	}
	if len(want) == 0 {
		return
	}
	hit := map[string]bool{}
	r.walk(func(body []byte) bool {
		for sym, needle := range want {
			if containsWord(body, needle) {
				hit[sym] = true
				delete(want, sym)
			}
		}
		return len(want) > 0
	})
	for sym := range hit {
		r.symbols[sym] = true
	}
	// Now the misses, as answers. A qualified symbol the walk did not find
	// resolves to its member's verdict — the same fall-through
	// resolveSymbol would take, decided once instead of once per citation.
	for sym := range cited {
		if _, decided := r.symbols[sym]; decided {
			continue
		}
		verdict := false
		if m := sym[strings.LastIndex(sym, ".")+1:]; m != sym && m != "" {
			verdict = hit[m]
		}
		r.symbols[sym] = verdict
	}
}

// Member is one record of a derived cluster, with the relation that put
// it there.
type Member struct {
	Record string `json:"record"`
	Title  string `json:"title"`
	// Status is the member's lifecycle status, so a caller applying 7.1's
	// Final-and-unimplemented scope filters without opening the record.
	Status string `json:"status,omitempty"`
	// Relation is why the record is a member: `seed`, `declared`,
	// `mutual-predecessor`, `peer-evidence` or `cross-cutting` — or
	// `mutual-mentions`, the candidate tier: two in-flight records that
	// each name the other in prose with no typed relation between them.
	Relation string `json:"relation"`
	// Candidate marks the mutual-mentions tier: a member to confirm or
	// dismiss, not one the records assert.
	Candidate bool `json:"candidate,omitempty"`
}

// ClusterOf derives a record's cluster from the edge graph.
//
// This is Stage 7.1's own membership rule, made a query. The prompt
// defines the cluster as the related records under the records dir, where
// "related = mutual `**Predecessors**:`, Peer-RDR citations, or a shared
// Cross-Cutting Concern", and today that set is built by an LLM reading
// every candidate record in the dir. Over the edge graph it is a
// traversal: the criteria are three edge kinds and one direction test.
//
// MUTUAL is meant strictly. A one-way predecessor is the ordinary
// build-order dependency every record has several of; what makes a
// cluster is two records that each name the other, which is the shape
// that cannot be resolved by implementing one first. Peer-evidence and
// cross-cutting relate in either direction — a record whose claim rests
// on a peer's element is entangled with it whichever way the citation
// runs — and a declared `Cluster` field is authoritative on its own,
// because the author asserted it.
//
// Membership is reported one hop from the seed, with the relation that
// earned it, so the caller sees why each member is in the set rather than
// a bare list to take on trust.
//
// THE CANDIDATE TIER. Checked against the clusters Stage 7.1 actually
// reconciled (seven snapshots of one consumer corpus), the typed rule
// reproduced three exactly and missed members in the rest — members
// joined to the seed only by dense prose cross-reference, twenty or
// thirty bare mentions each way and no typed edge at all. So a pair of
// in-flight records that each mention the other is reported too, marked
// `mutual-mentions` and Candidate, one hop only. It is a lead the caller
// confirms, not an assertion the records make; restricting it to
// in-flight pairs and to mutual mention keeps it from pulling in the
// implemented history every record cites. Two historical members had no
// citation in either direction and were declared by the user; no rule
// over the records can recover those, and a declared `Cluster` field is
// the fix.
func ClusterOf(docs []*Document, seed string) []Member {
	byRecord := map[string]*Document{}
	for _, d := range docs {
		byRecord[d.Record] = d
	}
	self, ok := byRecord[seed]
	if !ok {
		return nil
	}
	// names[record][kind] is true when the seed points at that record
	// with that kind; inbound is the same in reverse.
	out := map[string]string{}
	note := func(rec, relation string) {
		if rec == seed || byRecord[rec] == nil {
			return
		}
		if _, seen := out[rec]; !seen {
			out[rec] = relation
		}
	}
	status := func(d *Document) string {
		if f := d.MetadataField("Status"); f != nil && f.Status != nil {
			return f.Status.Value
		}
		return ""
	}
	outbound := map[string]map[edge.Kind]bool{}
	for _, e := range self.Edges {
		rec := recordOf(e.To)
		if outbound[rec] == nil {
			outbound[rec] = map[edge.Kind]bool{}
		}
		outbound[rec][e.Kind] = true
	}
	// A declared Cluster field is the author's own assertion and needs no
	// derivation.
	for rec, kinds := range outbound {
		if kinds[edge.Cluster] {
			note(rec, "declared")
		}
	}
	for _, d := range docs {
		if d.Record == seed {
			continue
		}
		for _, e := range d.Edges {
			if recordOf(e.To) != seed {
				continue
			}
			switch e.Kind {
			case edge.Cluster:
				note(d.Record, "declared")
			case edge.Predecessor:
				if outbound[d.Record][edge.Predecessor] {
					note(d.Record, "mutual-predecessor")
				}
			case edge.PeerEvidence:
				note(d.Record, "peer-evidence")
			case edge.CrossCuttingOwner:
				note(d.Record, "cross-cutting")
			}
		}
	}
	for rec, kinds := range outbound {
		switch {
		case kinds[edge.PeerEvidence]:
			note(rec, "peer-evidence")
		case kinds[edge.CrossCuttingOwner]:
			note(rec, "cross-cutting")
		}
	}
	if IsInFlight(status(self)) {
		for _, d := range docs {
			if d.Record == seed || !outbound[d.Record][edge.Mentions] || !IsInFlight(status(d)) {
				continue
			}
			for _, e := range d.Edges {
				if e.Kind == edge.Mentions && recordOf(e.To) == seed {
					note(d.Record, "mutual-mentions")
					break
				}
			}
		}
	}

	members := []Member{{Record: seed, Title: self.Title, Status: status(self), Relation: "seed"}}
	rest := make([]string, 0, len(out))
	for rec := range out {
		rest = append(rest, rec)
	}
	sortStrings(rest)
	for _, rec := range rest {
		members = append(members, Member{Record: rec, Title: byRecord[rec].Title, Status: status(byRecord[rec]),
			Relation: out[rec], Candidate: out[rec] == "mutual-mentions"})
	}
	return members
}

// recordOf reads the record number out of an element or document ID, or
// "" when the target is not a record reference at all.
func recordOf(target string) string {
	if _, after, ok := strings.Cut(target, "/"); ok {
		target = after
	}
	before, _, _ := strings.Cut(target, ":")
	if recordNumber.MatchString(before) {
		return before
	}
	return ""
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
