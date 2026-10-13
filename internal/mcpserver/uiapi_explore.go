package mcpserver

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/BeppeTemp/cartographer/internal/auth"
	"github.com/BeppeTemp/cartographer/internal/kb"
	"github.com/BeppeTemp/cartographer/internal/okf"
)

// The Atlas's explore routes (D744): the features that existed only as MCP
// tools — map contracts, the source ledger, contradictions, services, git
// conflicts, a concept's history and assets, the journal, the path
// placeholders, the audit log and the caller's own identity — answered from
// the same tool handlers or kb functions, read-only, so the UI can show every
// feature without disagreeing with what an agent sees.
//
// Visibility follows the tool each route mirrors: a per-concept or
// per-collection answer is filtered with the caller's principal, a whole-KB
// one (the journal, conflicts, paths.yaml) is only for a caller that sees the
// whole KB, and the audit log, which names other principals, only for an
// admin.

// uiAuditMaxLimit bounds the audit entries one request returns.
const uiAuditMaxLimit = 500

// uiRelayOrEmpty is uiRelay for a tool that answers an empty result with a
// sentence ("No maps found.") instead of an empty JSON list: that one text is
// relayed as empty, anything else exactly as uiRelay would.
func uiRelayOrEmpty(w http.ResponseWriter, context string, res ToolResult, err error, emptyText string, empty interface{}) {
	if err == nil && !res.IsError && len(res.Content) > 0 && strings.TrimSpace(res.Content[0].Text) == emptyText {
		writeUIJSON(w, http.StatusOK, empty)
		return
	}
	uiRelay(w, context, res, err)
}

// GET /kbs/{kb}/contracts — map_list: every visible Map and Journal with its
// contract (ontology_mode, concept_types, field_values, effective stale_after,
// lint_ignore).
func uiContracts(w http.ResponseWriter, r *http.Request, srv *Server) {
	tool, ok := uiTool(srv, "map_list")
	if !ok {
		writeUINotFound(w)
		return
	}
	res, err := tool.Handler(r.Context(), json.RawMessage(`{}`))
	uiRelayOrEmpty(w, "contracts", res, err, "No maps found.", []interface{}{})
}

// GET /kbs/{kb}/sources?status= — source_list: the source ledger, every
// status unless one is asked for.
func uiSources(w http.ResponseWriter, r *http.Request, srv *Server) {
	tool, ok := uiTool(srv, "source_list")
	if !ok {
		writeUINotFound(w)
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "*"
	}
	args, _ := json.Marshal(map[string]interface{}{"status": status, "limit": maxSourceLimit})
	res, err := tool.Handler(r.Context(), args)
	uiRelay(w, "sources", res, err)
}

// GET /kbs/{kb}/contradictions?status= — contradiction_report's non-gap
// kinds, the ones that block commit_gate (gaps are in /status already). Every
// resolution status unless one is asked for, so a resolved disagreement is
// still on the record.
func uiContradictions(w http.ResponseWriter, r *http.Request, k *kb.KB) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "*"
	}
	matches, err := listContradictions(r.Context(), k, "", status, "contradiction")
	if err != nil {
		writeUIInternal(w, "contradictions: walk", err)
		return
	}
	if matches == nil {
		matches = []contradictionEntry{}
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{"contradictions": matches})
}

// GET /kbs/{kb}/services — the visible concepts of type Service. Not a relay
// of service_list: that tool lists every Service of the KB, and here each one
// is filtered with the caller's principal like every other concept.
func uiServices(w http.ResponseWriter, r *http.Request, k *kb.KB) {
	ctx := r.Context()
	type serviceRow struct {
		ID     string `json:"id"`
		Title  string `json:"title,omitempty"`
		Status string `json:"status,omitempty"`
	}
	rows := []serviceRow{}
	err := k.WalkConcepts(func(id okf.ConceptID, content string) error {
		fmRaw, _, _ := okf.SplitFrontmatter(content)
		fm, perr := okf.ParseFrontmatter(fmRaw)
		if perr != nil || !strings.EqualFold(fm.Type(), "Service") || !Visible(ctx, k, string(id)) {
			return nil
		}
		rows = append(rows, serviceRow{ID: string(id), Title: fmString(fm, "title"), Status: fmString(fm, "status")})
		return nil
	})
	if err != nil {
		writeUIInternal(w, "services: walk", err)
		return
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	writeUIJSON(w, http.StatusOK, map[string]interface{}{"services": rows})
}

// GET /kbs/{kb}/conflicts — conflicts_list: open git rebase conflicts. The
// registry is KB-wide and names files, so it is for a whole-KB caller.
func uiConflicts(w http.ResponseWriter, r *http.Request, srv *Server) {
	tool, ok := uiTool(srv, "conflicts_list")
	if !ok || !WholeVisible(r.Context(), srv.kbRef, false) {
		writeUINotFound(w)
		return
	}
	res, err := tool.Handler(r.Context(), json.RawMessage(`{}`))
	uiRelayOrEmpty(w, "conflicts", res, err, "No open conflicts.", []interface{}{})
}

// GET /kbs/{kb}/history?id=&limit= — concept_history for one visible concept.
// The tool's dispatcher checks visibility before the handler runs; this route
// calls the handler directly, so it checks it here.
func uiHistory(w http.ResponseWriter, r *http.Request, srv *Server) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeUIError(w, http.StatusBadRequest, uiCodeInvalidRequest, "id is required", "id")
		return
	}
	tool, ok := uiTool(srv, "concept_history")
	if !ok || !Visible(r.Context(), srv.kbRef, id) {
		writeUINotFound(w)
		return
	}
	args, _ := json.Marshal(map[string]interface{}{"id": id, "limit": uiLimit(r, conceptHistoryDefaultLimit, conceptHistoryMaxLimit)})
	res, err := tool.Handler(r.Context(), args)
	if err == nil && res.IsError {
		// A concept that cannot be resolved is a 404, like /concept.
		writeUINotFound(w)
		return
	}
	uiRelay(w, "history", res, err)
}

// GET /kbs/{kb}/assets?id= — asset_list for one visible concept. A concept
// that is not expanded has no assets: an empty list, not an error.
func uiAssets(w http.ResponseWriter, r *http.Request, k *kb.KB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeUIError(w, http.StatusBadRequest, uiCodeInvalidRequest, "id is required", "id")
		return
	}
	if !Visible(r.Context(), k, id) {
		writeUINotFound(w)
		return
	}
	entries, err := k.ListAssets(okf.ConceptID(id))
	if err != nil || entries == nil {
		entries = []kb.AssetEntry{}
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{"id": id, "assets": entries})
}

// GET /kbs/{kb}/journal?n= — log_tail of the root log: the KB's own journal,
// whole-KB like the log itself.
func uiJournal(w http.ResponseWriter, r *http.Request, k *kb.KB) {
	if !WholeVisible(r.Context(), k, false) {
		writeUINotFound(w)
		return
	}
	text, err := k.LogTail("", uiLimit(r, 50, 500))
	if err != nil {
		text = ""
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{"text": text})
}

// uiPathKey is one declared or cited placeholder key on /paths.
type uiPathKey struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Key         string `json:"key"`
	Declared    bool   `json:"declared"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Remote      string `json:"remote,omitempty"`
	// CitedBy are the visible concepts citing it; Artifacts the KB-root
	// artifact files (skills/…, agents/…, instructions.md).
	CitedBy   []string `json:"cited_by"`
	Artifacts []string `json:"artifacts"`
	// Conflicts are the other mounted KBs declaring the same key with a
	// different default or remote: the client resolves a key once per
	// machine, so two KBs disagreeing on it is worth seeing.
	Conflicts []uiPathConflict `json:"conflicts,omitempty"`
}

type uiPathConflict struct {
	KB      string `json:"kb"`
	Default string `json:"default,omitempty"`
	Remote  string `json:"remote,omitempty"`
}

// GET /kbs/{kb}/paths — the path placeholder mapping (#742, D744): every key
// paths.yaml declares, with its default and remote, who cites it, and the keys
// cited but undeclared. paths.yaml is a KB-level file, so the route is for a
// whole-KB caller, and the cross-KB comparison only reads KBs that caller
// also sees whole. The per-machine resolution lives on the client
// (`cartographer paths list`); this is only the declared side.
func (m *MultiKBServer) uiPaths(w http.ResponseWriter, r *http.Request, name string, k *kb.KB) {
	ctx := r.Context()
	if !WholeVisible(ctx, k, false) {
		writeUINotFound(w)
		return
	}
	state, err := k.ReadPathRegistry()
	if err != nil {
		writeUIInternal(w, "paths: registry", err)
		return
	}
	keys := map[string]*uiPathKey{}
	get := func(id string) *uiPathKey {
		if row, ok := keys[id]; ok {
			return row
		}
		kind, key, _ := okf.SplitPlaceholderID(id)
		row := &uiPathKey{ID: id, Kind: kind, Key: key, CitedBy: []string{}, Artifacts: []string{}}
		keys[id] = row
		return row
	}
	declare := func(kind string, decls map[string]kb.PathDecl) {
		for key, d := range decls {
			row := get(kind + ":" + key)
			row.Declared, row.Description, row.Default, row.Remote = true, d.Description, d.Default, d.Remote
		}
	}
	declare("path", state.Registry.Paths)
	declare("repo", state.Registry.Repos)

	err = k.WalkConcepts(func(id okf.ConceptID, content string) error {
		if !Visible(ctx, k, string(id)) {
			return nil
		}
		for _, pid := range okf.Placeholders(content) {
			row := get(pid)
			row.CitedBy = append(row.CitedBy, string(id))
		}
		return nil
	})
	if err != nil {
		writeUIInternal(w, "paths: walk", err)
		return
	}
	artifacts, err := k.ArtifactPlaceholderCitations()
	if err != nil {
		writeUIInternal(w, "paths: artifacts", err)
		return
	}
	for pid, files := range artifacts {
		get(pid).Artifacts = files
	}

	// The same key declared by another KB this caller sees whole, with a
	// different value.
	for otherName, other := range m.servers {
		if otherName == name || other.kbRef == nil || !WholeVisible(ctx, other.kbRef, false) {
			continue
		}
		otherState, oerr := other.kbRef.ReadPathRegistry()
		if oerr != nil {
			continue
		}
		for _, row := range keys {
			if !row.Declared {
				continue
			}
			d, ok := otherState.Registry.Declared(row.Kind, row.Key)
			if ok && (d.Default != row.Default || d.Remote != row.Remote) {
				row.Conflicts = append(row.Conflicts, uiPathConflict{KB: otherName, Default: d.Default, Remote: d.Remote})
			}
		}
	}

	rows := make([]uiPathKey, 0, len(keys))
	for _, row := range keys {
		sort.Strings(row.CitedBy)
		sort.Slice(row.Conflicts, func(i, j int) bool { return row.Conflicts[i].KB < row.Conflicts[j].KB })
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	malformed := []string{}
	for _, bad := range state.Malformed {
		malformed = append(malformed, bad.String())
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{
		"present":     state.Present,
		"unparseable": state.Unparseable,
		"malformed":   malformed,
		"keys":        rows,
	})
}

// GET /kbs/{kb}/audit?limit= — the newest audit entries for this KB, newest
// first. The log names every principal that called a tool, so it is for an
// admin only; a KB without an audit sink answers enabled: false.
func uiAudit(w http.ResponseWriter, r *http.Request, name string, srv *Server) {
	if !auth.PrincipalFromContext(r.Context()).Policy.Admin {
		writeUINotFound(w)
		return
	}
	srv.mu.Lock()
	l := srv.auditLog
	srv.mu.Unlock()
	if l == nil {
		writeUIJSON(w, http.StatusOK, map[string]interface{}{"enabled": false, "entries": []interface{}{}})
		return
	}
	limit := uiLimit(r, 100, uiAuditMaxLimit)
	// The sink can be shared by every KB: read deeper than the limit so the
	// filter on this KB still fills the page.
	all, err := l.Tail(limit * 10)
	if err != nil {
		writeUIInternal(w, "audit: tail", err)
		return
	}
	type auditRow struct {
		At         string            `json:"at"`
		Phase      string            `json:"phase,omitempty"`
		Principal  string            `json:"principal,omitempty"`
		Transport  string            `json:"transport,omitempty"`
		Tool       string            `json:"tool"`
		ReadOnly   bool              `json:"read_only,omitempty"`
		Resources  map[string]string `json:"resources,omitempty"`
		Outcome    string            `json:"outcome"`
		DurationMs int64             `json:"duration_ms,omitempty"`
		CommitSHA  string            `json:"commit_sha,omitempty"`
	}
	rows := []auditRow{}
	for _, e := range all {
		if e.KB != "" && e.KB != name {
			continue
		}
		rows = append(rows, auditRow{
			At: e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"), Phase: e.Phase, Principal: e.PrincipalID,
			Transport: e.Transport, Tool: e.Tool, ReadOnly: e.ReadOnly, Resources: e.Resources,
			Outcome: e.Outcome, DurationMs: e.DurationMs, CommitSHA: e.CommitSHA,
		})
		if len(rows) == limit {
			break
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{"enabled": true, "state": l.State(), "entries": rows})
}

// GET /whoami — who is looking: the principal behind the request and its
// scopes. It is the caller's own policy, so nothing in it is anyone else's;
// the rules are narrowed to the KBs this server mounts.
func (m *MultiKBServer) uiWhoami(w http.ResponseWriter, r *http.Request) {
	p := auth.PrincipalFromContext(r.Context())
	type ruleRow struct {
		KB       string   `json:"kb"`
		Write    bool     `json:"write"`
		Maps     []string `json:"maps,omitempty"`
		Journals []string `json:"journals,omitempty"`
		Types    []string `json:"types,omitempty"`
	}
	rules := []ruleRow{}
	for _, rule := range p.Policy.Permissions {
		if _, mounted := m.servers[rule.KB]; !mounted {
			continue
		}
		rules = append(rules, ruleRow{KB: rule.KB, Write: rule.Write, Maps: rule.Maps, Journals: rule.Journals, Types: rule.Types})
	}
	writeUIJSON(w, http.StatusOK, map[string]interface{}{
		"id":          p.ID,
		"admin":       p.Policy.Admin,
		"permissions": rules,
	})
}
