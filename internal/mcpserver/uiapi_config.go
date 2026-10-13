package mcpserver

import (
	"net/http"

	"github.com/BeppeTemp/cartographer/internal/auth"
	"github.com/BeppeTemp/cartographer/internal/config"
)

// SetServerConfig gives the UI API the resolved server configuration, and the
// path of the file it came from ("" when none was given), for GET /config.
// Called by `serve` before the web UI is mounted.
func (m *MultiKBServer) SetServerConfig(cfg *config.Config, path string) {
	m.serverConfig = cfg
	m.serverConfigPath = path
}

// GET /config — the effective server configuration, as documented sections and
// as commented YAML. Admin only, like the audit route: it names every token
// holder and every KB's remote. Secrets are redacted by config.DescribeConfig.
func (m *MultiKBServer) uiConfig(w http.ResponseWriter, r *http.Request) {
	if !auth.PrincipalFromContext(r.Context()).Policy.Admin {
		writeUINotFound(w)
		return
	}
	sections := config.DescribeConfig(m.serverConfig)
	writeUIJSON(w, http.StatusOK, map[string]interface{}{
		"source":   m.serverConfigPath,
		"yaml":     config.RenderYAML(sections),
		"sections": sections,
	})
}
