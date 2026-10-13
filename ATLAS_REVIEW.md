# Atlas review — branch `feat/atlas-surface-features`

> Nota di lavoro da rivedere in un'altra sessione. **Da cancellare prima del merge**
> (non è documentazione del progetto). Il branch non è stato mergiato e nessuna PR è
> stata aperta.

## Issue coperti

| Issue | Stato nel branch | Dove |
|---|---|---|
| #742 Atlas: path placeholder mapping | ✅ fatto | Library → *Path placeholders*; `GET /kbs/{kb}/paths` |
| #743 Atlas: server settings | ✅ lettura fatta · ✏️ modifica **non** fatta | pannello **Settings** |
| #744 Atlas: surface every feature (lato KB/server) | ✅ quasi tutto · vedi sotto | Library, Settings, Inspector |
| #744 lato client (bindings, agenti, approvals, servizio, update check) | ⏭️ non fatto | l'issue stesso dice "probabilmente issue separato": il server non vede la macchina del client |
| #745 step 2 (endpoint MCP pubblico) | ⏭️ non fatto | è hosting/deploy con decisioni aperte (Cloud Run vs homelab, dominio), non codice Atlas |

### Dettaglio #744 (lato KB/server)

| Voce dell'inventario | Fatto? | Dove |
|---|---|---|
| Contradictions (non-gap, bloccano `commit_gate`) | ✅ | Library → *Contradictions*; `GET /contradictions` |
| Source ledger | ✅ | Library → *Source ledger*; `GET /sources` |
| Map contracts (ontology_mode, concept_types, field_values, stale_after, lint_ignore) | ✅ | Settings → *Map contracts*; `GET /contracts` (= `map_list`) |
| Per-concept history | ✅ | Inspector → tab *History*; `GET /history?id=` |
| Journal (`log_tail`) | ✅ | Settings → *Journal*; `GET /journal` |
| Supersede chains | ✅ | Inspector → *Links* → *Supersede chain*; campi `supersedes`/`superseded_by` in `/concept` |
| Archived concepts come relazione | ❌ | non fatto: restano solo frontmatter |
| KB assets | ✅ | Inspector → tab *History* (sezione Assets); `GET /assets?id=` |
| Services | ✅ | Library → *Services*; `GET /services` (filtrato per visibilità, a differenza di `service_list`) |
| PR & gate cycle | 🟡 parziale | Settings mostra workflow, branch, PR (#, link), errori forge da `kb_status`. **Non** fatto: `gate_check`/`commit_gate`/`pr_status` live |
| Git conflicts | ✅ | Settings → *Git conflicts*; `GET /conflicts` |
| Audit log | ✅ (solo admin) | Settings → *Audit log*; `GET /audit` |
| Who is looking | ✅ | Settings → *Who is looking*; `GET /whoami` |

## Scelte da validare (D744)

- **Due nuove voci nel rail**: *Library* e *Settings*. Alternativa: tutto dentro Health (scartata nel decision file).
- **Audit log solo per admin** (i record nominano altri principal). Un lettore whole-KB non admin riceve 404.
- **Journal, conflicts, paths**: solo per chi vede tutta la KB (come `/status`).
- **Settings in sola lettura**: nessun hot-reload della config → il pannello dice "cambia in config.yaml, vale al riavvio". La modifica dei toggle "sicuri" resta un follow-up (route di scrittura + reload che oggi non esistono).
- **Demo statica**: le nuove route sono esportate da `demo/export.mjs` tranne `/audit` e `/whoami` (l'export gira come local admin).
- `paths`: conflitto = stessa chiave dichiarata da un'altra KB montata con `default` o `remote` diversi (solo KB che il chiamante vede intere).

## File toccati

- Backend: `internal/mcpserver/uiapi_explore.go` (nuovo), `uiapi.go` (routing + supersede), `internal/kb/pathregistry.go` (`ArtifactPlaceholderCitations`).
- Test Go: `internal/mcpserver/uiapi_explore_test.go`.
- Frontend: `web/src/components/{Library,Settings}.tsx` (nuovi), `Inspector.tsx`, `LeftRail.tsx`, `Icon.tsx`, `App.tsx`, `api/{client,types}.ts`, `lib/{useResource,viewstate}.ts`, CSS.
- Test web: `web/src/test/explore-panels.test.tsx`, `web/e2e/a11y.spec.ts` (axe + overflow anche su Library/Settings).
- Docs: `docs/control-plane.md` (tabella UI API), `docs/deployment.md` (pannelli), `docs/decisions/D744-…md`.
- Demo: `demo/export.mjs`. Bundle: `internal/webui/dist/` ricostruito.

## Verifiche eseguite

- `make gate` ✅
- `vitest` ✅ tranne `layout.test.ts > falls back when storage throws`, che fallisce **anche su main** in questo container (Node 22 invece del 26 richiesto da `.nvmrc`).
- e2e Playwright completo (`web/e2e/run.sh`) ✅ 59/59 + a11y esteso ✅ 25/25.
- `demo/build.sh` ✅ (la build della demo statica esporta le nuove route senza errori).
- Bundle costruito con Node 22: la CI (Node 26) rifà `make web-check`; la provenance è un hash dei sorgenti, quindi dovrebbe coincidere.

## Per chiudere

1. Rivedere UI (Library, Settings, tab History) — `make dev` o la demo statica.
2. Decidere se tenere due voci nel rail o accorpare.
3. Aprire la PR (`feat(web): …`, `Closes #742`, `Closes #743`?, `Refs #744`), **cancellando questo file**.
4. Eventuali follow-up come issue: modifica settings (#743), lato client di #744, archived concepts, gate/PR live.
