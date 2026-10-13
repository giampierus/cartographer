package mcpserver

import (
	"net/http"
	"strings"
	"testing"

	"github.com/BeppeTemp/cartographer/internal/auth"
	"github.com/BeppeTemp/cartographer/internal/config"
)

func configHandler(t *testing.T, ts *auth.TokenStore) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.HTTP = ":39273"
	cfg.Auth.Tokens = []config.TokenSpec{{Token: "tok-secret-value", ID: "bot"}}
	cfg.KBs = []config.KBSpec{{Path: "/data/kb-a", ArtifactSigningSeed: "seed-secret-value"}}
	cfg.Audit.KeySeed = "audit-secret-value"
	multi := NewMultiKBServer("test")
	multi.SetServerConfig(cfg, "/etc/example/config.yaml")
	multi.EnableWeb(nil)
	return ts.Middleware(multi.Handler())
}

func TestUIAPI_ConfigAdminGetsSectionsAndYAMLWithoutSecrets(t *testing.T) {
	handler := configHandler(t, auth.NewTokenStore(nil))
	rr := getUI(t, handler, UIAPIPrefix+"/config", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	for _, secret := range []string{"tok-secret-value", "seed-secret-value", "audit-secret-value"} {
		if strings.Contains(rr.Body.String(), secret) {
			t.Errorf("secret %q in the response", secret)
		}
	}
	body := decodeUI(t, rr)
	if body["source"] != "/etc/example/config.yaml" {
		t.Errorf("source = %v", body["source"])
	}
	if y, _ := body["yaml"].(string); !strings.Contains(y, "http: :39273") || !strings.Contains(y, "# ---- Git ----") {
		t.Errorf("yaml = %q", y)
	}
	secs, _ := body["sections"].([]interface{})
	if len(secs) != 9 || secs[0].(map[string]interface{})["key"] != "server" {
		t.Errorf("sections = %v", secs)
	}
}

func TestUIAPI_ConfigIsAdminOnly(t *testing.T) {
	ts := auth.NewScopedTokenStore([]auth.ScopedToken{
		{Principal: "reader", Token: "narrow", Policy: auth.Policy{Permissions: []auth.Permission{{KB: "docs"}}}},
	})
	handler := configHandler(t, ts)
	rr := getUI(t, handler, UIAPIPrefix+"/config", "narrow")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rr.Code)
	}
	audit := getUI(t, handler, UIAPIPrefix+"/kbs/docs/audit", "narrow")
	if audit.Body.String() != rr.Body.String() {
		t.Errorf("refusal differs from the audit route's: %q vs %q", rr.Body.String(), audit.Body.String())
	}
}
