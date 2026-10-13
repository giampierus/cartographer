package mcpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BeppeTemp/cartographer/internal/auth"
	"github.com/BeppeTemp/cartographer/internal/kb"
)

// exploreFixture adds to uiFixtureKB what the explore routes (D744) read: a
// Service and a Contradiction on each side of the visibility line, a
// paths.yaml, placeholder citations in a concept and in a skill.
func exploreFixture(t *testing.T, k *kb.KB, defaultPath string) {
	t.Helper()
	write := func(full, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data := k.DataRoot()
	write(filepath.Join(data, "visible/api.md"), "---\ntype: Service\ntitle: API\nstatus: active\n---\nLives in {{repo:api}} and {{path:undeclared}}.\n")
	write(filepath.Join(data, "hidden/vault.md"), "---\ntype: Service\ntitle: Vault\n---\nIn {{repo:api}}.\n")
	write(filepath.Join(data, "visible/clash.md"), "---\ntype: Contradiction\ntitle: Clash\ncontradiction_kind: factual\nresolution_status: open\ninvolves: [visible/alpha, visible/beta]\n---\nThey disagree.\n")
	write(filepath.Join(data, "visible/gap.md"), "---\ntype: Contradiction\ntitle: Gap\ncontradiction_kind: missing_context\n---\nUnknown.\n")
	write(filepath.Join(data, "hidden/quarrel.md"), "---\ntype: Contradiction\ntitle: Quarrel\ncontradiction_kind: factual\n---\nHidden.\n")
	write(filepath.Join(k.Root, "paths.yaml"), "repos:\n  api:\n    description: The API repo\n    default: "+defaultPath+"\n    remote: gitlab.example.com/team/api\npaths:\n  unused:\n    description: Nothing cites it\n")
	write(filepath.Join(k.Root, "skills/deploy/SKILL.md"), "---\nname: deploy\ndescription: Deploys.\n---\nRun in {{repo:api}}.\n")
}

func exploreHandler(t *testing.T, ts *auth.TokenStore) http.Handler {
	t.Helper()
	multi := NewMultiKBServer("test")
	docs := uiFixtureKB(t, "docs")
	exploreFixture(t, docs, "$HOME/src/api")
	other := uiFixtureKB(t, "other")
	exploreFixture(t, other, "$HOME/work/api")
	multi.MountKB("docs", func(s *Server) { RegisterKBTools(s, docs, Deps{}) })
	multi.MountKB("other", func(s *Server) { RegisterKBTools(s, other, Deps{}) })
	multi.EnableWeb(nil)
	return ts.Middleware(multi.Handler())
}

func TestUIAPI_ExploreRoutesServeInOpenMode(t *testing.T) {
	handler := exploreHandler(t, auth.NewTokenStore(nil))
	for _, path := range []string{
		"/kbs/docs/contracts",
		"/kbs/docs/sources",
		"/kbs/docs/contradictions",
		"/kbs/docs/services",
		"/kbs/docs/conflicts",
		"/kbs/docs/history?id=visible/alpha",
		"/kbs/docs/assets?id=visible/owner",
		"/kbs/docs/journal",
		"/kbs/docs/paths",
		"/kbs/docs/audit",
		"/whoami",
	} {
		rr := getUI(t, handler, UIAPIPrefix+path, "")
		if rr.Code != http.StatusOK {
			t.Errorf("%s: status %d, want 200: %s", path, rr.Code, rr.Body.String())
		}
	}
	for _, path := range []string{"/kbs/docs/history", "/kbs/docs/assets"} {
		if rr := getUI(t, handler, UIAPIPrefix+path, ""); rr.Code != http.StatusBadRequest {
			t.Errorf("%s without id: status %d, want 400", path, rr.Code)
		}
	}
}

func TestUIAPI_ContradictionsAreTheNonGapKinds(t *testing.T) {
	handler := exploreHandler(t, auth.NewTokenStore(nil))
	body := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/contradictions", ""))
	var ids []string
	for _, raw := range body["contradictions"].([]interface{}) {
		ids = append(ids, raw.(map[string]interface{})["id"].(string))
	}
	// The gap is in /status; both disagreements are here.
	if want := []string{"hidden/quarrel", "visible/clash"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("contradictions = %v, want %v", ids, want)
	}
}

func TestUIAPI_PathsShowDeclaredCitedAndConflicting(t *testing.T) {
	handler := exploreHandler(t, auth.NewTokenStore(nil))
	body := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/paths", ""))
	if body["present"] != true {
		t.Fatalf("present = %v, want true", body["present"])
	}
	keys := map[string]map[string]interface{}{}
	for _, raw := range body["keys"].([]interface{}) {
		row := raw.(map[string]interface{})
		keys[row["id"].(string)] = row
	}
	api := keys["repo:api"]
	if api == nil || api["declared"] != true || api["default"] != "$HOME/src/api" {
		t.Fatalf("repo:api = %v (malformed %v)", api, body["malformed"])
	}
	if got := api["cited_by"].([]interface{}); len(got) != 2 {
		t.Errorf("repo:api cited_by = %v, want both services", got)
	}
	if got := api["artifacts"].([]interface{}); len(got) != 1 || got[0] != "skills/deploy/SKILL.md" {
		t.Errorf("repo:api artifacts = %v", got)
	}
	conflicts, _ := api["conflicts"].([]interface{})
	if len(conflicts) != 1 || conflicts[0].(map[string]interface{})["kb"] != "other" {
		t.Errorf("repo:api conflicts = %v, want the other KB", conflicts)
	}
	if u := keys["path:undeclared"]; u == nil || u["declared"] != false {
		t.Errorf("path:undeclared = %v, want cited but undeclared", u)
	}
	if u := keys["path:unused"]; u == nil || u["declared"] != true || len(u["cited_by"].([]interface{})) != 0 {
		t.Errorf("path:unused = %v, want declared and uncited", u)
	}
}

func TestUIAPI_ExploreRoutesHonourANarrowedToken(t *testing.T) {
	ts := auth.NewScopedTokenStore([]auth.ScopedToken{
		{Principal: "reader", Token: "narrow", Policy: auth.Policy{Permissions: []auth.Permission{
			{KB: "docs", Maps: []string{"visible"}},
		}}},
	})
	handler := exploreHandler(t, ts)

	services := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/services", "narrow"))
	for _, raw := range services["services"].([]interface{}) {
		if raw.(map[string]interface{})["id"] == "hidden/vault" {
			t.Error("hidden service listed")
		}
	}
	contradictions := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/contradictions", "narrow"))
	for _, raw := range contradictions["contradictions"].([]interface{}) {
		if raw.(map[string]interface{})["id"] == "hidden/quarrel" {
			t.Error("hidden contradiction listed")
		}
	}
	for _, path := range []string{
		"/kbs/docs/history?id=hidden/secret",
		"/kbs/docs/assets?id=hidden/secret",
		// Whole-KB answers.
		"/kbs/docs/journal",
		"/kbs/docs/conflicts",
		"/kbs/docs/paths",
		// Admin only.
		"/kbs/docs/audit",
	} {
		if rr := getUI(t, handler, UIAPIPrefix+path, "narrow"); rr.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, rr.Code)
		}
	}

	who := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/whoami", "narrow"))
	if who["id"] != "reader" || who["admin"] != false {
		t.Errorf("whoami = %v", who)
	}
	rules := who["permissions"].([]interface{})
	if len(rules) != 1 || rules[0].(map[string]interface{})["kb"] != "docs" {
		t.Errorf("whoami permissions = %v", rules)
	}
}

func TestUIAPI_ConceptCarriesTheSupersedeChain(t *testing.T) {
	ts := auth.NewScopedTokenStore([]auth.ScopedToken{
		{Principal: "reader", Token: "narrow", Policy: auth.Policy{Permissions: []auth.Permission{
			{KB: "docs", Maps: []string{"visible"}},
		}}},
	})
	multi := NewMultiKBServer("test")
	k := uiFixtureKB(t, "docs")
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(k.DataRoot(), rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("visible/old.md", "---\ntype: Note\ntitle: Old\nsuperseded_by: visible/beta\n---\nSee [beta](beta.md).\n")
	write("hidden/older.md", "---\ntype: Note\ntitle: Older\nsuperseded_by: visible/beta\n---\nSee [beta](../visible/beta.md).\n")
	write("visible/gone.md", "---\ntype: Note\ntitle: Gone\nsuperseded_by: hidden/secret\n---\nSee [secret](../hidden/secret.md).\n")
	multi.MountKB("docs", func(s *Server) { RegisterKBTools(s, k, Deps{}) })
	multi.EnableWeb(nil)
	handler := ts.Middleware(multi.Handler())

	beta := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/concept?id=visible/beta", "narrow"))
	// The hidden predecessor is not disclosed.
	if got := beta["supersedes"].([]interface{}); len(got) != 1 || got[0] != "visible/old" {
		t.Errorf("supersedes = %v, want [visible/old]", got)
	}
	old := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/concept?id=visible/old", "narrow"))
	if old["superseded_by"] != "visible/beta" {
		t.Errorf("superseded_by = %v, want visible/beta", old["superseded_by"])
	}
	// A hidden successor is not named as a relation.
	gone := decodeUI(t, getUI(t, handler, UIAPIPrefix+"/kbs/docs/concept?id=visible/gone", "narrow"))
	if gone["superseded_by"] != "" {
		t.Errorf("superseded_by = %v, want empty for a hidden successor", gone["superseded_by"])
	}
}
