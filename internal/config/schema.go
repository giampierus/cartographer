package config

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// KeyDoc documents one YAML key of the server configuration. Schema is the
// single table of them: the Atlas Settings panel and the commented YAML are
// rendered from it, and schema_test.go fails when a key of the config structs
// has no entry (or an entry names a key that does not exist), so a key added
// to the config cannot ship undocumented.
//
// Path is the YAML path with "[]" for a list element: "git.sync",
// "kbs[].path", "auth.tokens[].scopes". RuleSpec (auth.roles[].rules) and
// provisioning.MCPAllowlistEntry (kbs[].mcp_allowlist) are documented as one
// key each, their fields named in the description.
type KeyDoc struct {
	Path    string
	Doc     string
	Values  []string // accepted values when the key is an enum
	Default string   // display string; empty when the key has no default
	Secret  bool
	Env     string // CARTOGRAPHER_* variable that sets it, when one exists
	Flag    string // `serve` flag that sets it, when one exists
}

func kd(p, def, doc string) KeyDoc       { return KeyDoc{Path: p, Default: def, Doc: doc} }
func (k KeyDoc) vals(v ...string) KeyDoc { k.Values = v; return k }
func (k KeyDoc) env(e string) KeyDoc     { k.Env = e; return k }
func (k KeyDoc) flag(f string) KeyDoc    { k.Flag = f; return k }
func (k KeyDoc) secret() KeyDoc          { k.Secret = true; return k }

// Key returns the last segment of the path ("sync" for "git.sync").
func (k KeyDoc) Key() string {
	if i := strings.LastIndex(k.Path, "."); i >= 0 {
		return k.Path[i+1:]
	}
	return k.Path
}

// Schema documents every key of the server configuration, in the order the
// Atlas and the commented YAML present them.
var Schema = []KeyDoc{
	// Server (top level)
	kd("http", "", "Address the server listens on, for example \":39273\". Empty serves over stdio instead of HTTP.").env("CARTOGRAPHER_HTTP").flag("--http"),
	kd("init", "false", "Create a missing KB (an empty git repository with the standard layout) instead of failing to start.").flag("--init"),
	kd("data", "", "Directory whose subdirectories are mounted as KBs automatically, and where KBs declared by remote are cloned.").env("CARTOGRAPHER_DATA").flag("--data"),
	kd("update_check", "true", "Look up the latest release once a day and report it in /health and kb_status when it is newer. A development build never checks (D254).").env("CARTOGRAPHER_UPDATE_CHECK").flag("--update-check"),

	// Auth
	kd("auth.mode", "auto", "Whether the HTTP endpoint requires a bearer token. auto turns it on when at least one token is configured.").vals("on", "off", "auto").env("CARTOGRAPHER_AUTH"),
	kd("auth.tokens[].token", "", "The bearer token. Write ${NAME} to read it from an environment variable, so the file never carries the secret. A bare string in the list is an admin token.").secret().env("CARTOGRAPHER_TOKENS").flag("--tokens"),
	kd("auth.tokens[].scopes", "", "Per-KB access of this token, as kb:<name>:r (read) or kb:<name>:rw (read and write). Empty with no roles means full access to every KB (admin)."),
	kd("auth.tokens[].roles", "", "Names of auth.roles this token holds. Roles and scopes can be combined: the permissions are the union."),
	kd("auth.tokens[].id", "", "Stable name of the token in logs and the audit trail. When empty, a non-secret ID is derived from the token hash."),
	kd("auth.tokens[].author_name", "", "Git author name of the commits written through this token, instead of the KB's identity. Set together with author_email (D341)."),
	kd("auth.tokens[].author_email", "", "Git author email of the commits written through this token. Set together with author_name."),
	kd("auth.roles[].name", "", "Name of the role, referenced from auth.tokens[].roles."),
	kd("auth.roles[].rules", "", "The role's allow rules. Each rule has kb (the KB name), access (r or rw) and optionally maps, journals and types to narrow it to some folders or concept types; empty lists match everything. There are no deny rules."),

	// KBs
	kd("kbs[].artifact_signing_seed", "", "32-byte hexadecimal Ed25519 seed used to sign the provisioning artifacts this KB serves. Write ${NAME} to read it from the environment; never commit it.").secret(),
	kd("kbs[].mcp_allowlist", "", "Third-party MCP servers this KB may provision to clients, each with name, transport (http or stdio) and target (the exact URL or command). Empty denies all of them."),
	kd("kbs[].path", "", "Folder of a KB already on disk. Set either path or remote."),
	kd("kbs[].remote", "", "Git remote to clone the KB from into the data directory. Set either path or remote."),
	kd("kbs[].name", "", "Name of the KB, used in the HTTP endpoint (/mcp/<name>), in token scopes and for the clone folder. When empty it is derived from the remote or path (D53)."),
	kd("kbs[].ssh_key", "", "SSH private key used to reach this KB's remote. Empty uses git.ssh_key."),
	kd("kbs[].known_hosts", "", "known_hosts file for this KB's SSH remote. Empty uses git.known_hosts."),
	kd("kbs[].author_name", "", "Git author name of this KB's commits. Empty uses git.author_name."),
	kd("kbs[].author_email", "", "Git author email of this KB's commits. Empty uses git.author_email."),
	kd("kbs[].committer_name", "", "Git committer name of this KB's commits. Empty uses git.committer_name, then the author."),
	kd("kbs[].committer_email", "", "Git committer email of this KB's commits. Empty uses git.committer_email, then the author."),
	kd("kbs[].sops_age_key_file", "", "age key file that decrypts this KB's SOPS secrets. Empty uses sops.age_key_dir, then sops.age_key_file."),
	kd("kbs[].git_profile", "", "Git workflow of this KB, overriding git.profile. local pushes straight to the branch; server writes on a working branch and opens a pull request (D117).").vals("local", "server"),
	kd("kbs[].git_base_branch", "", "Server profile: the protected branch that pull requests target. Overrides git.base_branch."),
	kd("kbs[].git_working_branch", "", "Server profile: the branch Cartographer writes to. Default cartographer/<kb-name>."),
	kd("kbs[].git_forge", "", "Server profile: the code host that receives pull requests. Overrides git.forge."),
	kd("kbs[].github_owner", "", "Server profile: GitHub owner of the repository; must match the origin remote. Overrides git.github_owner."),
	kd("kbs[].github_repository", "", "Server profile: GitHub repository name. Overrides git.github_repository."),
	kd("kbs[].github_api_url", "", "Server profile: GitHub API base URL. Overrides git.github_api_url."),
	kd("kbs[].github_token_env", "", "Server profile: name of the environment variable holding the GitHub token. Overrides git.github_token_env."),
	kd("kbs[].git_branch", "", "Local profile: branch to write to instead of the remote's default one. The first push creates it when missing (D335)."),
	kd("kbs[].allow_artifact_write", "false", "Enable artifact_write and artifact_delete on this KB. Writing a skill injects instructions clients will run, so a read-write token alone does not grant it (D71)."),
	kd("kbs[].allow_template_write", "true", "Let artifact_write and artifact_delete write templates/<slug>.md and nothing else when allow_artifact_write is off. Templates are never provisioned to a client (D352)."),
	kd("kbs[].auto_repair", "every check that is safe unattended", "Lint checks whose mechanical fix the server may apply on its own, by name (there is no \"all\"). An explicit empty list turns auto repair off (D299, D323)."),
	kd("kbs[].repair_on_write", "follows auto_repair", "Apply the auto_repair fixes to the concepts a write has just touched, in the same commit. By default on exactly when auto_repair is not empty (D349)."),
	kd("kbs[].write_gate", "off", "Refuse, and roll back, a content write that introduces a lint finding at or above this level: error, or warning (errors and warnings) (D350).").vals("off", "error", "warning"),
	kd("kbs[].doctor_auto_interval", "1d", "How often the server runs the auto_repair checks by itself, as days (\"1d\" or \"1\"). \"0\" turns it off (D323)."),
	kd("kbs[].doctor_interval", "14d", "Days after the last kb-doctor session before the server proposes the next one, as \"14d\" or \"14\". \"0\" disables the proposal; a session never starts by itself (D299, D704)."),
	kd("kbs[].doctor_mode", "", "Accepted and ignored since D704: a doctor session starts only when the operator asks for one.").vals("assisted", "unattended"),
	kd("kbs[].doctor_budget", "10", "How many review items one kb-doctor session decides before it closes (D358)."),
	kd("kbs[].usage_stale_days", "42", "Days a skill or agent may go unused by every client before the artifact_unused check reports it. 0 disables the finding (D326)."),

	// Git
	kd("git.autocommit", "true", "Commit every write to the KB's git repository.").env("CARTOGRAPHER_GIT_AUTOCOMMIT").flag("--git-autocommit"),
	kd("git.sync", "true", "Pull from and push to the KB's remote around reads and writes.").env("CARTOGRAPHER_GIT_SYNC").flag("--git-sync"),
	kd("git.ssh_key", "", "SSH private key used to reach KB remotes."),
	kd("git.known_hosts", "", "known_hosts file used with the SSH key."),
	kd("git.author_name", "", "Default git author name of the server's commits (a KB can override it)."),
	kd("git.author_email", "", "Default git author email of the server's commits (a KB can override it)."),
	kd("git.committer_name", "", "Default git committer name. Empty uses the author."),
	kd("git.committer_email", "", "Default git committer email. Empty uses the author."),
	kd("git.token_dir", "", "Directory with one <name>.token file per KB, used as HTTPS git credentials for that KB's remote (D53).").env("CARTOGRAPHER_GIT_TOKEN_DIR"),
	kd("git.in_window", "30s", "After a successful pull, reads and writes within this window skip the next one. 0 pulls every time (D76, D93).").env("CARTOGRAPHER_SYNC_IN_WINDOW"),
	kd("git.out_debounce", "3s", "Wait this long after the last write before pushing, so a burst becomes one push. 0 pushes synchronously on each write (D76).").env("CARTOGRAPHER_SYNC_OUT_DEBOUNCE"),
	kd("git.profile", "local", "Git workflow. local pushes straight to the branch; server writes on a working branch and opens a pull request, and needs the keys below (D117).").vals("local", "server").env("CARTOGRAPHER_GIT_PROFILE"),
	kd("git.base_branch", "", "Server profile: the protected branch pull requests target. Cartographer never pushes to it directly."),
	kd("git.working_branch", "", "Server profile: the branch Cartographer writes to. Default cartographer/<kb-name>."),
	kd("git.forge", "", "Server profile: the code host that receives pull requests (github)."),
	kd("git.github_owner", "", "Server profile: GitHub owner of the repository; must match the origin remote."),
	kd("git.github_repository", "", "Server profile: GitHub repository name."),
	kd("git.github_api_url", "", "Server profile: GitHub API base URL, for example https://api.github.com."),
	kd("git.github_token_env", "", "Server profile: name of the environment variable holding the GitHub token."),

	// SOPS
	kd("sops.age_key_file", "", "Default age key file that decrypts SOPS secrets referenced by concepts.").env("CARTOGRAPHER_SOPS_AGE_KEY_FILE"),
	kd("sops.age_key_dir", "", "Directory with one <name>.age key per KB, used before age_key_file (D53).").env("CARTOGRAPHER_SOPS_AGE_KEY_DIR"),

	// Audit
	kd("audit.log", "", "File of the append-only, tamper-evident audit trail. Empty turns the audit off.").env("CARTOGRAPHER_AUDIT_LOG"),
	kd("audit.key_seed", "", "Seed that signs the audit trail so it can be verified. Write ${NAME} to read it from the environment.").secret().env("CARTOGRAPHER_AUDIT_KEY"),
	kd("audit.mode", "best_effort", "What a failed audit write does. best_effort logs it and lets the call proceed; required rejects the call, so the trail never misses an operation (D119).").vals("best_effort", "required"),
	kd("audit.max_segment_bytes", "package default", "Size at which the active audit segment is rotated."),
	kd("audit.archive_dir", "", "Folder for rotated audit segments. Empty keeps them beside the log."),
	kd("audit.retention_days", "90", "Rotated segments older than this many days are deleted once their checkpoint is written. 0 keeps everything (D325)."),

	// Tools, MCP, web
	kd("tools.profile", "agent", "Which tools the MCP tools/list advertises. agent lists the core set; full lists every tool. Hidden tools stay callable (D65).").vals("agent", "full").env("CARTOGRAPHER_TOOLS_PROFILE").flag("--tools-profile"),
	kd("mcp.allowed_origins", "", "Browser origins allowed to reach the MCP endpoint, scheme and port included. Empty accepts only an Origin matching the request's own Host; \"*\" accepts any. Requests without an Origin (every non-browser client) are unaffected (D128).").env("CARTOGRAPHER_MCP_ALLOWED_ORIGINS"),
	kd("web.enabled", "true", "Serve the read-only Atlas UI at /ui/ and its API at /api/ui/v1 when the server is in HTTP mode.").env("CARTOGRAPHER_WEB_ENABLED").flag("--web-enabled"),
}

// SectionDoc describes one section of the configuration.
type SectionDoc struct{ Key, Title, Doc string }

// SchemaSections lists the sections in presentation order.
var SchemaSections = []SectionDoc{
	{"server", "Server", "How the server listens and where its data lives."},
	{"auth", "Auth", "Bearer-token authentication of the HTTP endpoint: tokens, and roles that group permissions."},
	{"kbs", "Knowledge bases", "The KBs the server mounts, one entry each, with their per-KB overrides."},
	{"git", "Git", "Commit and sync behaviour, and the identity and credentials used to reach KB remotes."},
	{"sops", "SOPS", "The age keys that decrypt SOPS secrets referenced by concepts."},
	{"audit", "Audit", "The append-only, tamper-evident record of every tool call."},
	{"tools", "Tools", "Which MCP tools are advertised to agents."},
	{"mcp", "MCP", "Behaviour of the MCP endpoint that is not specific to one KB."},
	{"web", "Web", "The embedded read-only Atlas UI."},
}

func sectionOf(p string) string {
	i := strings.IndexAny(p, ".[")
	if i < 0 {
		return "server"
	}
	return p[:i]
}

// docFor returns the schema entry for a path, which may contain concrete
// indexes ("kbs[2].path" resolves to "kbs[].path").
func docFor(p string) (KeyDoc, bool) {
	for _, k := range Schema {
		if k.Path == p {
			return k, true
		}
	}
	return KeyDoc{}, false
}

// ---- resolved view ---------------------------------------------------------

// Entry is one configuration key with its effective value.
type Entry struct {
	Path      string      `json:"path"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Default   string      `json:"default"`
	IsDefault bool        `json:"is_default"`
	Doc       string      `json:"doc"`
	Values    []string    `json:"values,omitempty"`
	Env       string      `json:"env,omitempty"`
	Flag      string      `json:"flag,omitempty"`
	Secret    bool        `json:"secret"`
}

// Item is one element of a list section (a KB, a token, a role).
type Item struct {
	Title   string  `json:"title"`
	Entries []Entry `json:"entries"`
}

// SchemaSection is one section of the resolved view: plain entries, or items
// for a list.
type SchemaSection struct {
	Key     string  `json:"key"`
	Title   string  `json:"title"`
	Doc     string  `json:"doc,omitempty"`
	Entries []Entry `json:"entries,omitempty"`
	Items   []Item  `json:"items,omitempty"`
}

// Redacted replaces a non-empty secret in the resolved view.
const Redacted = "••••••"

type acc func(v interface{}) (value interface{}, isDefault bool)

func strAcc(get func(interface{}) string, def string) acc {
	return func(v interface{}) (interface{}, bool) {
		s := get(v)
		if s == "" {
			return def, true
		}
		return s, s == def
	}
}

func cfgStr(f func(*Config) string, def string) acc {
	return strAcc(func(v interface{}) string { return f(v.(*Config)) }, def)
}
func kbStr(f func(KBSpec) string) acc {
	return strAcc(func(v interface{}) string { return f(v.(KBSpec)) }, "")
}
func tokStr(f func(TokenSpec) string) acc {
	return strAcc(func(v interface{}) string { return f(v.(TokenSpec)) }, "")
}
func cfgBool(f func(*Config) bool, def bool) acc {
	return func(v interface{}) (interface{}, bool) { b := f(v.(*Config)); return b, b == def }
}
func kbBool(f func(KBSpec) bool, def bool) acc {
	return func(v interface{}) (interface{}, bool) { b := f(v.(KBSpec)); return b, b == def }
}
func cfgList(f func(*Config) []string) acc {
	return func(v interface{}) (interface{}, bool) { l := f(v.(*Config)); return nonNil(l), len(l) == 0 }
}
func nonNil(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}
func daysAcc(raw func(KBSpec) string, resolve func(KBSpec) (int, error), def int) acc {
	return func(v interface{}) (interface{}, bool) {
		s := v.(KBSpec)
		n, err := resolve(s)
		if err != nil {
			return raw(s), false
		}
		if n == 0 {
			return "0", def == 0
		}
		return strconv.Itoa(n) + "d", n == def
	}
}

// accessors resolve the effective value of each key and whether it is the
// default. Keyed by schema path; the subject is *Config, KBSpec, TokenSpec or
// RoleSpec according to the prefix.
var accessors = map[string]acc{
	"http":         cfgStr(func(c *Config) string { return c.HTTP }, ""),
	"init":         cfgBool(func(c *Config) bool { return c.Init }, false),
	"data":         cfgStr(func(c *Config) string { return c.Data }, ""),
	"update_check": cfgBool(func(c *Config) bool { return c.UpdateCheck }, true),
	"auth.mode":    cfgStr(func(c *Config) string { return c.Auth.Mode }, "auto"),

	"auth.tokens[].token":        tokStr(func(t TokenSpec) string { return t.Token }),
	"auth.tokens[].id":           tokStr(func(t TokenSpec) string { return t.ID }),
	"auth.tokens[].author_name":  tokStr(func(t TokenSpec) string { return t.AuthorName }),
	"auth.tokens[].author_email": tokStr(func(t TokenSpec) string { return t.AuthorEmail }),
	"auth.tokens[].scopes": func(v interface{}) (interface{}, bool) {
		l := v.(TokenSpec).Scopes
		return nonNil(l), len(l) == 0
	},
	"auth.tokens[].roles": func(v interface{}) (interface{}, bool) {
		l := v.(TokenSpec).Roles
		return nonNil(l), len(l) == 0
	},
	"auth.roles[].name": func(v interface{}) (interface{}, bool) { return v.(RoleSpec).Name, false },
	"auth.roles[].rules": func(v interface{}) (interface{}, bool) {
		rules := v.(RoleSpec).Rules
		out := make([]map[string]interface{}, 0, len(rules))
		for _, r := range rules {
			m := map[string]interface{}{"kb": r.KB, "access": r.Access}
			if len(r.Maps) > 0 {
				m["maps"] = r.Maps
			}
			if len(r.Journals) > 0 {
				m["journals"] = r.Journals
			}
			if len(r.Types) > 0 {
				m["types"] = r.Types
			}
			out = append(out, m)
		}
		return out, len(out) == 0
	},

	"kbs[].artifact_signing_seed": kbStr(func(s KBSpec) string { return s.ArtifactSigningSeed }),
	"kbs[].mcp_allowlist": func(v interface{}) (interface{}, bool) {
		l := v.(KBSpec).MCPAllowlist
		if l == nil {
			return []interface{}{}, true
		}
		return l, len(l) == 0
	},
	"kbs[].path":               kbStr(func(s KBSpec) string { return s.Path }),
	"kbs[].remote":             kbStr(func(s KBSpec) string { return s.Remote }),
	"kbs[].name":               kbStr(func(s KBSpec) string { return s.Name }),
	"kbs[].ssh_key":            kbStr(func(s KBSpec) string { return s.SSHKey }),
	"kbs[].known_hosts":        kbStr(func(s KBSpec) string { return s.KnownHosts }),
	"kbs[].author_name":        kbStr(func(s KBSpec) string { return s.AuthorName }),
	"kbs[].author_email":       kbStr(func(s KBSpec) string { return s.AuthorEmail }),
	"kbs[].committer_name":     kbStr(func(s KBSpec) string { return s.CommitterName }),
	"kbs[].committer_email":    kbStr(func(s KBSpec) string { return s.CommitterEmail }),
	"kbs[].sops_age_key_file":  kbStr(func(s KBSpec) string { return s.SopsAgeKeyFile }),
	"kbs[].git_profile":        kbStr(func(s KBSpec) string { return s.GitProfile }),
	"kbs[].git_base_branch":    kbStr(func(s KBSpec) string { return s.GitBaseBranch }),
	"kbs[].git_working_branch": kbStr(func(s KBSpec) string { return s.GitWorkingBranch }),
	"kbs[].git_forge":          kbStr(func(s KBSpec) string { return s.GitForge }),
	"kbs[].github_owner":       kbStr(func(s KBSpec) string { return s.GitHubOwner }),
	"kbs[].github_repository":  kbStr(func(s KBSpec) string { return s.GitHubRepository }),
	"kbs[].github_api_url":     kbStr(func(s KBSpec) string { return s.GitHubAPIURL }),
	"kbs[].github_token_env":   kbStr(func(s KBSpec) string { return s.GitHubTokenEnv }),
	"kbs[].git_branch":         kbStr(func(s KBSpec) string { return s.GitBranch }),

	"kbs[].allow_artifact_write": kbBool(func(s KBSpec) bool { return s.AllowArtifactWrite }, false),
	"kbs[].allow_template_write": kbBool(KBSpec.TemplateWriteEnabled, true),
	"kbs[].auto_repair": func(v interface{}) (interface{}, bool) {
		checks, isDef := v.(KBSpec).AutoRepairChecks()
		return nonNil(checks), isDef
	},
	"kbs[].repair_on_write": func(v interface{}) (interface{}, bool) {
		s := v.(KBSpec)
		return s.RepairOnWriteEnabled(), s.RepairOnWrite == nil
	},
	"kbs[].write_gate": func(v interface{}) (interface{}, bool) {
		m := v.(KBSpec).WriteGateMode()
		return m, m == "off"
	},
	"kbs[].doctor_auto_interval": daysAcc(func(s KBSpec) string { return s.DoctorAutoInterval }, KBSpec.DoctorAutoIntervalDays, DefaultDoctorAutoIntervalDays),
	"kbs[].doctor_interval":      daysAcc(func(s KBSpec) string { return s.DoctorInterval }, KBSpec.DoctorIntervalDays, DefaultDoctorIntervalDays),
	"kbs[].doctor_mode":          kbStr(func(s KBSpec) string { return s.DoctorMode }),
	"kbs[].doctor_budget": func(v interface{}) (interface{}, bool) {
		n := v.(KBSpec).DoctorBudgetItems()
		return n, n == DefaultDoctorBudget
	},
	"kbs[].usage_stale_days": func(v interface{}) (interface{}, bool) {
		n := v.(KBSpec).UsageStale()
		return n, n == DefaultUsageStaleDays
	},

	"git.autocommit":        cfgBool(func(c *Config) bool { return c.Git.Autocommit }, true),
	"git.sync":              cfgBool(func(c *Config) bool { return c.Git.Sync }, true),
	"git.ssh_key":           cfgStr(func(c *Config) string { return c.Git.SSHKey }, ""),
	"git.known_hosts":       cfgStr(func(c *Config) string { return c.Git.KnownHosts }, ""),
	"git.author_name":       cfgStr(func(c *Config) string { return c.Git.AuthorName }, ""),
	"git.author_email":      cfgStr(func(c *Config) string { return c.Git.AuthorEmail }, ""),
	"git.committer_name":    cfgStr(func(c *Config) string { return c.Git.CommitterName }, ""),
	"git.committer_email":   cfgStr(func(c *Config) string { return c.Git.CommitterEmail }, ""),
	"git.token_dir":         cfgStr(func(c *Config) string { return c.Git.TokenDir }, ""),
	"git.in_window":         cfgStr(func(c *Config) string { return c.Git.SyncInWindow.String() }, "30s"),
	"git.out_debounce":      cfgStr(func(c *Config) string { return c.Git.SyncOutDebounce.String() }, "3s"),
	"git.profile":           cfgStr(func(c *Config) string { return c.Git.Profile }, "local"),
	"git.base_branch":       cfgStr(func(c *Config) string { return c.Git.BaseBranch }, ""),
	"git.working_branch":    cfgStr(func(c *Config) string { return c.Git.WorkingBranch }, ""),
	"git.forge":             cfgStr(func(c *Config) string { return c.Git.Forge }, ""),
	"git.github_owner":      cfgStr(func(c *Config) string { return c.Git.GitHubOwner }, ""),
	"git.github_repository": cfgStr(func(c *Config) string { return c.Git.GitHubRepository }, ""),
	"git.github_api_url":    cfgStr(func(c *Config) string { return c.Git.GitHubAPIURL }, ""),
	"git.github_token_env":  cfgStr(func(c *Config) string { return c.Git.GitHubTokenEnv }, ""),

	"sops.age_key_file": cfgStr(func(c *Config) string { return c.Sops.AgeKeyFile }, ""),
	"sops.age_key_dir":  cfgStr(func(c *Config) string { return c.Sops.AgeKeyDir }, ""),

	"audit.log":      cfgStr(func(c *Config) string { return c.Audit.Log }, ""),
	"audit.key_seed": cfgStr(func(c *Config) string { return c.Audit.KeySeed }, ""),
	"audit.mode":     cfgStr(func(c *Config) string { return c.Audit.Mode }, "best_effort"),
	"audit.max_segment_bytes": func(v interface{}) (interface{}, bool) {
		n := v.(*Config).Audit.MaxSegmentBytes
		return n, n == 0
	},
	"audit.archive_dir": cfgStr(func(c *Config) string { return c.Audit.ArchiveDir }, ""),
	"audit.retention_days": func(v interface{}) (interface{}, bool) {
		n := v.(*Config).Audit.RetentionDays
		return n, n == DefaultAuditRetentionDays
	},

	"tools.profile":       cfgStr(func(c *Config) string { return c.ToolsProfile }, "agent"),
	"mcp.allowed_origins": cfgList(func(c *Config) []string { return c.MCP.AllowedOrigins }),
	"web.enabled":         cfgBool(func(c *Config) bool { return c.Web.Enabled }, true),
}

func entryFor(k KeyDoc, subject interface{}, concretePath string) Entry {
	val, isDef := accessors[k.Path](subject)
	if s, ok := val.(string); ok && k.Secret && s != "" {
		val = Redacted
	}
	return Entry{
		Path: concretePath, Key: k.Key(), Value: val, Default: k.Default, IsDefault: isDef,
		Doc: k.Doc, Values: k.Values, Env: k.Env, Flag: k.Flag, Secret: k.Secret,
	}
}

func entriesWhere(prefix string, subject interface{}, concrete func(string) string) []Entry {
	var out []Entry
	for _, k := range Schema {
		if strings.HasPrefix(k.Path, prefix) {
			out = append(out, entryFor(k, subject, concrete(k.Path)))
		}
	}
	return out
}

// kbTitle names a KB the way the server does: Name, else the basename of the
// remote or path.
func kbTitle(s KBSpec, i int) string {
	if s.Name != "" {
		return s.Name
	}
	for _, p := range []string{s.Remote, s.Path} {
		if p = strings.TrimRight(strings.TrimSpace(p), "/"); p != "" {
			b := path.Base(strings.ReplaceAll(strings.TrimSuffix(p, ".git"), ":", "/"))
			if b != "" && b != "." && b != "/" {
				return b
			}
		}
	}
	return fmt.Sprintf("kb %d", i+1)
}

// DescribeConfig returns the effective configuration as documented sections,
// in the order of SchemaSections. Secrets are replaced by Redacted, and a
// token is titled by its ID (or its position), never by its value.
func DescribeConfig(cfg *Config) []SchemaSection {
	if cfg == nil {
		cfg = Default()
	}
	var out []SchemaSection
	for _, sd := range SchemaSections {
		sec := SchemaSection{Key: sd.Key, Title: sd.Title, Doc: sd.Doc}
		switch sd.Key {
		case "server":
			for _, k := range Schema {
				if sectionOf(k.Path) == "server" {
					sec.Entries = append(sec.Entries, entryFor(k, cfg, k.Path))
				}
			}
		case "auth":
			sec.Entries = append(sec.Entries, entryFor(mustDoc("auth.mode"), cfg, "auth.mode"))
			for i, t := range cfg.Auth.Tokens {
				idx := fmt.Sprintf("auth.tokens[%d].", i)
				title := t.ID
				if title == "" {
					title = fmt.Sprintf("token %d", i+1)
				}
				sec.Items = append(sec.Items, Item{Title: title, Entries: entriesWhere("auth.tokens[].", t,
					func(p string) string { return idx + strings.TrimPrefix(p, "auth.tokens[].") })})
			}
			for i, r := range cfg.Auth.Roles {
				idx := fmt.Sprintf("auth.roles[%d].", i)
				sec.Items = append(sec.Items, Item{Title: r.Name, Entries: entriesWhere("auth.roles[].", r,
					func(p string) string { return idx + strings.TrimPrefix(p, "auth.roles[].") })})
			}
		case "kbs":
			for i, s := range cfg.KBs {
				idx := fmt.Sprintf("kbs[%d].", i)
				sec.Items = append(sec.Items, Item{Title: kbTitle(s, i), Entries: entriesWhere("kbs[].", s,
					func(p string) string { return idx + strings.TrimPrefix(p, "kbs[].") })})
			}
		default:
			sec.Entries = entriesWhere(sd.Key+".", cfg, func(p string) string { return p })
		}
		out = append(out, sec)
	}
	return out
}

func mustDoc(p string) KeyDoc {
	k, ok := docFor(p)
	if !ok {
		panic("config: no schema entry for " + p)
	}
	return k
}

// ---- commented YAML --------------------------------------------------------

// RenderYAML writes the sections as a commented YAML file. Every key is
// preceded by its description (and its accepted values and default); a key at
// its default is written commented out as "# key: default", one that was
// changed is a real line. An unchanged configuration therefore parses to an
// empty document, and uncommenting a line is how a key is set.
func RenderYAML(sections []SchemaSection) string {
	var b strings.Builder
	b.WriteString("# Cartographer server configuration. Keys left commented are at their default.\n")
	for _, sec := range sections {
		b.WriteString("\n# ---- " + sec.Title + " ----\n")
		for _, l := range wrapComment(sec.Doc, "") {
			b.WriteString(l + "\n")
		}
		if sec.Key == "server" {
			writeEntries(&b, sec.Entries, "")
		} else {
			writeBlock(&b, sec)
		}
	}
	return b.String()
}

// hasReal reports whether any entry is a real (non-default) line.
func hasReal(es []Entry) bool {
	for _, e := range es {
		if !e.IsDefault {
			return true
		}
	}
	return false
}

func writeBlock(b *strings.Builder, sec SchemaSection) {
	switch sec.Key {
	case "kbs":
		writeList(b, "kbs", "", sec.Items, "kbs[].", "one entry per KB")
	case "auth":
		real := hasReal(sec.Entries)
		for _, it := range sec.Items {
			real = real || len(it.Entries) > 0
		}
		head := "auth:"
		if !real {
			head = "# auth:"
		}
		b.WriteString(head + "\n")
		writeEntries(b, sec.Entries, "  ")
		var toks, roles []Item
		for _, it := range sec.Items {
			if strings.HasPrefix(it.Entries[0].Path, "auth.tokens") {
				toks = append(toks, it)
			} else {
				roles = append(roles, it)
			}
		}
		writeList(b, "tokens", "  ", toks, "auth.tokens[].", "one entry per token")
		writeList(b, "roles", "  ", roles, "auth.roles[].", "one entry per role")
	default:
		head := sec.Key + ":"
		if !hasReal(sec.Entries) {
			head = "# " + head
		}
		b.WriteString(head + "\n")
		writeEntries(b, sec.Entries, "  ")
	}
}

// writeList writes a list key. With no item, a commented template item
// documents the keys; an item with no real key is written as {}.
func writeList(b *strings.Builder, key, indent string, items []Item, schemaPrefix, note string) {
	if len(items) == 0 {
		b.WriteString(indent + "# " + key + ":\n")
		b.WriteString(indent + "#   - (" + note + "; uncomment the keys to set)\n")
		var tmpl []Entry
		for _, k := range Schema {
			if strings.HasPrefix(k.Path, schemaPrefix) {
				tmpl = append(tmpl, Entry{Path: k.Path, Key: k.Key(), Default: k.Default, IsDefault: true,
					Doc: k.Doc, Values: k.Values, Env: k.Env, Flag: k.Flag, Secret: k.Secret})
			}
		}
		writeEntries(b, tmpl, indent+"    ")
		return
	}
	b.WriteString(indent + key + ":\n")
	for _, it := range items {
		b.WriteString(indent + "  # " + it.Title + "\n")
		writeItem(b, it.Entries, indent+"  ")
	}
}

type yline struct {
	text string
	real bool
}

func entryLines(e Entry) []yline {
	var ls []yline
	for _, l := range wrapComment(e.Doc, "") {
		ls = append(ls, yline{l, false})
	}
	if len(e.Values) > 0 {
		ls = append(ls, yline{"# values: " + strings.Join(e.Values, " | "), false})
	}
	if e.Default != "" {
		ls = append(ls, yline{"# default: " + e.Default, false})
	}
	if e.Env != "" || e.Flag != "" {
		var p []string
		if e.Env != "" {
			p = append(p, "env "+e.Env)
		}
		if e.Flag != "" {
			p = append(p, "flag "+e.Flag)
		}
		ls = append(ls, yline{"# " + strings.Join(p, ", "), false})
	}
	if e.IsDefault {
		d := e.Default
		if e.Secret {
			d = ""
		}
		ls = append(ls, yline{strings.TrimRight("# "+e.Key+": "+d, " "), false})
		return ls
	}
	raw, err := yaml.Marshal(map[string]interface{}{e.Key: e.Value})
	if err != nil {
		raw = []byte(e.Key + ": \"\"\n")
	}
	for _, l := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		ls = append(ls, yline{l, true})
	}
	return ls
}

func writeEntries(b *strings.Builder, es []Entry, indent string) {
	for _, e := range es {
		for _, l := range entryLines(e) {
			b.WriteString(indent + l.text + "\n")
		}
	}
}

// writeItem writes a list element: the first real line carries the "- ".
func writeItem(b *strings.Builder, es []Entry, indent string) {
	var all []yline
	for _, e := range es {
		all = append(all, entryLines(e)...)
	}
	first := true
	var buf bytes.Buffer
	for _, l := range all {
		switch {
		case !l.real:
			buf.WriteString(indent + "  " + l.text + "\n")
		case first:
			buf.WriteString(indent + "- " + l.text + "\n")
			first = false
		default:
			buf.WriteString(indent + "  " + l.text + "\n")
		}
	}
	if first {
		buf.WriteString(indent + "- {}\n")
	}
	b.Write(buf.Bytes())
}

// wrapComment splits text into "# " lines of at most ~78 columns.
func wrapComment(text, indent string) []string {
	if text == "" {
		return nil
	}
	var lines []string
	cur := "#"
	for _, w := range strings.Fields(text) {
		if len(cur)+1+len(w) > 78 && cur != "#" {
			lines = append(lines, indent+cur)
			cur = "#"
		}
		cur += " " + w
	}
	return append(lines, indent+cur)
}
