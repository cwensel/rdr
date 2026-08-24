package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
}

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
	return truth(false)
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
	found := r.grep(sym)
	r.symbols[sym] = found
	return truth(found)
}

// grep walks the repo for the symbol as a whole word. It is a plain walk
// rather than a shell-out: the binary is stdlib-only, and a walk is both
// portable and free of the quoting hazards of building a grep command
// out of corpus text.
func (r *Resolver) grep(sym string) bool {
	needle := []byte(sym)
	found := false
	filepath.WalkDir(r.repo, func(p string, e os.DirEntry, err error) error {
		if err != nil || found {
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
		if containsWord(body, needle) {
			found = true
		}
		return nil
	})
	return found
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
	for i := 0; i+len(needle) <= len(body); i++ {
		if string(body[i:i+len(needle)]) != string(needle) {
			continue
		}
		if i > 0 && identByte(body[i-1]) {
			continue
		}
		if j := i + len(needle); j < len(body) && identByte(body[j]) {
			continue
		}
		return true
	}
	return false
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

// ResolveAll decides the edges of a whole corpus.
func (r *Resolver) ResolveAll(docs []*Document) {
	for _, d := range docs {
		r.Resolve(d)
	}
}

// Member is one record of a derived cluster, with the relation that put
// it there.
type Member struct {
	Record string `json:"record"`
	Title  string `json:"title"`
	// Relation is why the record is a member: `seed`, `declared`,
	// `mutual-predecessor`, `peer-evidence` or `cross-cutting`.
	Relation string `json:"relation"`
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

	members := []Member{{Record: seed, Title: self.Title, Relation: "seed"}}
	rest := make([]string, 0, len(out))
	for rec := range out {
		rest = append(rest, rec)
	}
	sortStrings(rest)
	for _, rec := range rest {
		members = append(members, Member{Record: rec, Title: byRecord[rec].Title, Relation: out[rec]})
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
