package config

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/BeppeTemp/cartographer/internal/provisioning"
	"gopkg.in/yaml.v3"
)

// yamlKeys walks the config structs by their yaml tags and returns every key
// path. RuleSpec and MCPAllowlistEntry are leaves: the schema documents each
// as one key (see KeyDoc).
func yamlKeys(t reflect.Type, prefix string, out map[string]bool) {
	leaves := map[reflect.Type]bool{
		reflect.TypeOf(RuleSpec{}):                       true,
		reflect.TypeOf(provisioning.MCPAllowlistEntry{}): true,
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := strings.Split(f.Tag.Get("yaml"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		p := prefix + name
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		list := false
		if ft.Kind() == reflect.Slice {
			ft = ft.Elem()
			list = true
		}
		if ft.Kind() == reflect.Struct && !leaves[ft] {
			if list {
				p += "[]"
			}
			yamlKeys(ft, p+".", out)
			continue
		}
		out[p] = true
	}
}

// TestSchemaCoversEveryYAMLKey is the guard of issue #747: a key added to the
// config structs without a schema entry (or with an empty description) fails
// here, and so does an entry for a key that does not exist.
func TestSchemaCoversEveryYAMLKey(t *testing.T) {
	keys := map[string]bool{}
	yamlKeys(reflect.TypeOf(rawConfig{}), "", keys)
	if len(keys) < 60 {
		t.Fatalf("walked only %d keys: the reflection walk is broken", len(keys))
	}
	doc := map[string]bool{}
	for _, k := range Schema {
		if doc[k.Path] {
			t.Errorf("schema lists %q twice", k.Path)
		}
		doc[k.Path] = true
		if strings.TrimSpace(k.Doc) == "" {
			t.Errorf("schema entry %q has no description", k.Path)
		}
		if !keys[k.Path] {
			t.Errorf("schema entry %q names a key that is not in the config", k.Path)
		}
		if _, ok := accessors[k.Path]; !ok {
			t.Errorf("schema entry %q has no accessor", k.Path)
		}
	}
	var missing []string
	for p := range keys {
		if !doc[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("config keys with no schema entry: %v", missing)
	}
	for p := range accessors {
		if !doc[p] {
			t.Errorf("accessor %q has no schema entry", p)
		}
	}
}

func TestSchemaSectionsAreComplete(t *testing.T) {
	known := map[string]bool{}
	for _, s := range SchemaSections {
		known[s.Key] = true
	}
	for _, k := range Schema {
		if !known[sectionOf(k.Path)] {
			t.Errorf("%q belongs to unknown section %q", k.Path, sectionOf(k.Path))
		}
	}
}

func TestDescribeDefaultConfigIsAllDefaults(t *testing.T) {
	secs := DescribeConfig(Default())
	var keys []string
	for _, s := range secs {
		keys = append(keys, s.Key)
		if len(s.Items) != 0 {
			t.Errorf("section %s has items in the default config", s.Key)
		}
		for _, e := range s.Entries {
			if !e.IsDefault {
				t.Errorf("%s is not at its default in Default(): value %v, default %q", e.Path, e.Value, e.Default)
			}
		}
	}
	want := []string{"server", "auth", "kbs", "git", "sops", "audit", "tools", "mcp", "web"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("sections = %v, want %v", keys, want)
	}
}

func TestRenderDefaultYAMLHasNoRealKey(t *testing.T) {
	out := RenderYAML(DescribeConfig(Default()))
	var raw rawConfig
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(out), &node); err != nil {
		t.Fatalf("default render does not parse: %v\n%s", err, out)
	}
	if node.Kind != 0 && len(node.Content) > 0 && len(node.Content[0].Content) > 0 {
		t.Errorf("default render has real keys:\n%s", out)
	}
	if err := yaml.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# values: on | off | auto", "# default: 90", "# audit:"} {
		if !strings.Contains(out, want) {
			t.Errorf("default render lacks %q", want)
		}
	}
}

func nonDefaultConfig() *Config {
	cfg := Default()
	cfg.HTTP = ":39273"
	cfg.Git.Sync = false
	cfg.Git.SyncInWindow = 0
	cfg.Audit.Log = "/data/audit.log"
	cfg.Audit.KeySeed = "audit-seed-secret"
	cfg.Web.Enabled = false
	cfg.MCP.AllowedOrigins = []string{"https://app.example.com"}
	cfg.Auth.Tokens = []TokenSpec{
		{Token: "tok-secret", ID: "bot", Scopes: []string{"kb:kb-a:r"}},
		{Token: "tok-two-secret"},
	}
	cfg.Auth.Roles = []RoleSpec{{Name: "reader", Rules: []RuleSpec{{KB: "kb-a", Access: "r", Maps: []string{"infra"}}}}}
	days := 3
	cfg.KBs = []KBSpec{
		{Remote: "https://gitlab.example.com/team/kb-a.git", ArtifactSigningSeed: "seed-secret", WriteGate: "error",
			DoctorInterval: "7d", UsageStaleDays: &days,
			MCPAllowlist: []provisioning.MCPAllowlistEntry{{Name: "t", Transport: "http", Target: "https://tools.example.com/mcp"}}},
		{Path: "/data/work-kb", Name: "work", AutoRepair: []string{}},
	}
	return cfg
}

func TestRenderYAMLParsesAndRoundTripsValues(t *testing.T) {
	out := RenderYAML(DescribeConfig(nonDefaultConfig()))
	var raw rawConfig
	if err := yaml.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("render does not parse: %v\n%s", err, out)
	}
	if raw.HTTP != ":39273" || raw.Git.Sync == nil || *raw.Git.Sync || raw.Audit.Log != "/data/audit.log" ||
		raw.Web.Enabled == nil || *raw.Web.Enabled || raw.Git.InWindow != "0s" {
		t.Errorf("scalar values lost: %+v\n%s", raw, out)
	}
	if len(raw.KBs) != 2 || raw.KBs[0].WriteGate != "error" || raw.KBs[0].DoctorInterval != "7d" ||
		raw.KBs[0].UsageStaleDays == nil || *raw.KBs[0].UsageStaleDays != 3 || len(raw.KBs[0].MCPAllowlist) != 1 ||
		raw.KBs[1].Name != "work" || raw.KBs[1].AutoRepair == nil || len(raw.KBs[1].AutoRepair) != 0 {
		t.Errorf("kbs lost: %+v\n%s", raw.KBs, out)
	}
	if len(raw.Auth.Tokens) != 2 || raw.Auth.Tokens[0].ID != "bot" || len(raw.Auth.Roles) != 1 ||
		len(raw.Auth.Roles[0].Rules) != 1 || raw.Auth.Roles[0].Rules[0].Maps[0] != "infra" {
		t.Errorf("auth lost: %+v\n%s", raw.Auth, out)
	}
	// An unchanged key stays commented, with its description.
	if !strings.Contains(out, "# autocommit: true") || !strings.Contains(out, "# Commit every write") {
		t.Errorf("default key not rendered as a commented line with its doc:\n%s", out)
	}
}

func TestDescribeRedactsSecrets(t *testing.T) {
	secs := DescribeConfig(nonDefaultConfig())
	body, _ := json.Marshal(secs)
	yml := RenderYAML(secs)
	for _, secret := range []string{"tok-secret", "tok-two-secret", "audit-seed-secret", "seed-secret"} {
		if strings.Contains(string(body), secret) || strings.Contains(yml, secret) {
			t.Errorf("secret %q leaked", secret)
		}
	}
	if !strings.Contains(string(body), Redacted) {
		t.Error("no redaction marker in the sections")
	}
	var titles []string
	for _, s := range secs {
		for _, it := range s.Items {
			titles = append(titles, it.Title)
		}
	}
	want := []string{"bot", "token 2", "reader", "kb-a", "work"}
	if !reflect.DeepEqual(titles, want) {
		t.Errorf("item titles = %v, want %v", titles, want)
	}
}
