---
topic: control-plane
---

# D744 — The Atlas surfaces every server feature read-only, through the tools' own answers

**Decision.** The features that existed only as MCP tools get a read-only place
in the Atlas: a **Library** panel (contradictions that block `commit_gate`, the
source ledger, services, the path placeholder mapping), a **Settings** panel
(`kb_status`'s replication facts and capabilities, who is looking, Map
contracts, git conflicts, the journal, the audit log) and a *History* tab plus a
*Supersede chain* in the reading panel. Each section reads a new
`/api/ui/v1` route that runs the matching tool's handler, or the same `kb`
function the tool calls, with the request's principal. Visibility follows the
tool: per-concept and per-collection answers are filtered, whole-KB ones
(journal, conflicts, `paths.yaml`) answer 404 to a narrowed principal, and the
audit log, which names every principal, answers only an admin.

**Why.** A new user learns what Cartographer does by browsing it, and an
operator needs to see how the server is configured without a terminal. Reusing
the tools' handlers keeps the page from disagreeing with what an agent reads,
the same reason the reader routes of D286 relay their tools. The cost is more
routes and two more rail entries, and a page that shows a setting it cannot
change: editing is left out because a setting lives in `config.yaml`, which the
server does not hot-reload, and a write surface in the UI API would break its
read-only-by-construction guarantee (D226).

**Alternatives rejected.**
- One generic "call any read-only tool" route: the UI would depend on every
  tool's text format, and a tool that answers prose ("No maps found.") or skips
  visibility filtering (`service_list`) would leak or break the page.
- Folding everything into Health: Health answers "what needs attention", and a
  settings table or a source ledger is reference, not attention.
- Editing the safe toggles from Settings now: it needs a write route behind the
  same auth and a reload path the server does not have; it stays a follow-up.
- Exposing the audit log to any whole-KB reader: its entries name other
  principals, which a read token should not learn.

**Consequences.** A new tool a user should see needs its route and its section,
and the route a line in `demo/export.mjs` (D745). The audit log and `/whoami`
are not exported to the static demo: the export runs as the local admin. The
path mapping shows only the declared side; the per-machine resolution stays on
the client (`cartographer paths list`). The client-side features of the issue
(workspace bindings, connected agents, approvals, service and update status)
are not here: the server cannot see the client's machine.
