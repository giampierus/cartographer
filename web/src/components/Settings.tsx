import { fetchAudit, fetchConflicts, fetchContracts, fetchJournal, fetchStatus, fetchWhoAmI } from "../api/client";
import type { AuditLog, GitConflict, KBStatus, KBSummary, MapContract, WhoAmI } from "../api/types";
import { STATIC_MODE } from "../api/static";
import { useResource } from "../lib/useResource";
import { Figures, Hero, Page, PageHeader, PageSection, Quiet, SkeletonRows, relativeDay } from "./Page";
import { ErrorState } from "./States";

/**
 * The Settings panel (issue 743, D744): how the server and this KB are set up,
 * read-only. The KB half is kb_status — the replication facts and every
 * capability with the YAML key that controls it — so the page and an agent
 * never disagree. Then who is looking, the Map contracts, open git conflicts,
 * the journal and, for an admin, the audit log. Nothing here writes: a setting
 * changes in the server's config.yaml and takes effect when it restarts.
 */
export function Settings({
  kb,
  kbs,
  reloadKey,
  onOpenConcept,
  onRetry,
  onFailure,
}: {
  kb: string;
  kbs: KBSummary[];
  reloadKey: string;
  onOpenConcept(id: string): void;
  onRetry(): void;
  onFailure(err: unknown): boolean;
}) {
  const key = `${kb}:${reloadKey}`;
  const status = useResource(key, (s) => fetchStatus(kb, s), onFailure);
  const who = useResource(STATIC_MODE ? null : reloadKey, (s) => fetchWhoAmI(s), onFailure);
  const contracts = useResource(key, (s) => fetchContracts(kb, s), onFailure);
  const conflicts = useResource(key, (s) => fetchConflicts(kb, s), onFailure);
  const journal = useResource(key, (s) => fetchJournal(kb, s), onFailure);
  const audit = useResource(STATIC_MODE ? null : key, (s) => fetchAudit(kb, s), onFailure);

  const error = status.error ?? who.error ?? contracts.error ?? conflicts.error ?? journal.error ?? audit.error;
  if (error) return <ErrorState error={error} onRetry={onRetry} />;
  const st = status.data;

  return (
    <Page label="Settings" className="settings" busy={status.loading}>
      <PageHeader
        eyebrow="Settings"
        title={st?.server_version ? `Cartographer ${st.server_version}, serving ${kb}.` : `How ${kb} is set up.`}
        subtitle={
          st?.latest_version && st.latest_version !== st.server_version
            ? `Release ${st.latest_version} is available.`
            : "Read-only: a setting changes in the server's config.yaml and takes effect when the server restarts."
        }
        help={
          <p>
            Every capability names the <code>config.yaml</code> key that controls it. The same facts are in{" "}
            <code>kb_status</code>, which an agent reads, and in <code>cartographer status</code> on a client.
          </p>
        }
      />

      <Hero label="Settings in numbers">
        <Figures
          items={[
            { label: "Knowledge bases", value: kbs.length },
            { label: "Concepts", value: st?.total ?? "—" },
            {
              label: "Capabilities on",
              value: st?.capabilities
                ? Object.values(st.capabilities).filter((c) => c.state !== "disabled").length
                : "—",
            },
            {
              label: "Unpushed commits",
              value: st ? (st.unpushed_commits ?? 0) : "—",
              tone: (st?.unpushed_commits ?? 0) > 0 ? "warning" : undefined,
            },
            { label: "Maps and journals", value: contracts.data?.length ?? "—" },
          ]}
        />
      </Hero>
      {!who.absent && <WhoIsLooking who={who.data} loading={who.loading} />}
      {status.absent ? (
        <Quiet tone="neutral">The server's settings for this KB are for a reader that can see all of it.</Quiet>
      ) : (
        <>
          <Replication status={st} />
          <Capabilities status={st} />
        </>
      )}
      <MountedKBs kbs={kbs} current={kb} />
      {!contracts.absent && <Contracts list={contracts.data} />}
      {!conflicts.absent && <Conflicts list={conflicts.data} onOpenConcept={onOpenConcept} />}
      {!journal.absent && <Journal text={journal.data?.text ?? null} />}
      {!audit.absent && <Audit log={audit.data} />}
    </Page>
  );
}

function Facts({ items }: { items: [string, React.ReactNode][] }) {
  return (
    <dl className="settings__facts">
      {items.map(([label, value]) => (
        <div key={label} className="settings__fact">
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function WhoIsLooking({ who, loading }: { who: WhoAmI | null; loading: boolean }) {
  return (
    <PageSection title="Who is looking" id="settings-who">
      {loading && !who ? (
        <SkeletonRows label="Loading your identity" rows={2} />
      ) : who ? (
        <>
          <Facts
            items={[
              ["Principal", <code key="id">{who.id || "anonymous"}</code>],
              ["Access", who.admin ? "admin: every KB, read and write" : `${who.permissions.length} rule(s)`],
            ]}
          />
          {!who.admin && who.permissions.length > 0 && (
            <ul className="rows">
              {who.permissions.map((p, i) => (
                <li key={i} className="row settings__rule">
                  <span className={p.write ? "pill pill--warning" : "pill"}>{p.write ? "read-write" : "read"}</span>
                  <span className="row__body">
                    <span className="row__title">{p.kb}</span>
                    <span className="row__meta">
                      {p.maps?.length ? <span>maps: {p.maps.join(", ")}</span> : null}
                      {p.journals?.length ? <span>journals: {p.journals.join(", ")}</span> : null}
                      {p.types?.length ? <span>types: {p.types.join(", ")}</span> : null}
                      {!p.maps?.length && !p.journals?.length && !p.types?.length && <span>the whole KB</span>}
                    </span>
                  </span>
                  <span />
                </li>
              ))}
            </ul>
          )}
        </>
      ) : null}
    </PageSection>
  );
}

function when(iso?: string): string {
  return iso ? relativeDay(iso) : "never";
}

function Replication({ status }: { status: KBStatus | null }) {
  return (
    <PageSection title="Git and replication" id="settings-git">
      {!status ? (
        <SkeletonRows label="Loading replication state" rows={3} />
      ) : (
        <>
          <Facts
            items={[
              ["Remote", status.has_remote ? <code key="r">{status.remote_url}</code> : "none: local only"],
              ["Workflow", status.git_workflow === "pr" ? "pull requests" : (status.git_workflow ?? "—")],
              ["Sync", status.git_sync ? "on" : "off"],
              [
                "Push state",
                <span
                  key="p"
                  className={
                    status.push_state === "failed" || status.push_state === "degraded"
                      ? "pill pill--error"
                      : status.push_state === "pending"
                        ? "pill pill--warning"
                        : "pill"
                  }
                >
                  {status.push_state || "—"}
                </span>,
              ],
              ["Unpushed / behind", `${status.unpushed_commits ?? 0} / ${status.behind_commits ?? 0}`],
              ["Last push", when(status.last_push_at)],
              ["Last pull", when(status.last_sync_in_at)],
              ...(status.working_branch
                ? ([["Branches", `${status.working_branch} → ${status.base_branch ?? "?"}`]] as [string, string][])
                : []),
              ...(status.pr_url
                ? ([
                    [
                      "Pull request",
                      <a key="pr" href={status.pr_url} target="_blank" rel="noreferrer noopener">
                        #{status.pr_number}
                      </a>,
                    ],
                  ] as [string, React.ReactNode][])
                : []),
            ]}
          />
          {status.push_last_error && (
            <p className="library__warn" role="note">
              Last push failed: {status.push_last_error}
            </p>
          )}
          {status.last_forge_error && (
            <p className="library__warn" role="note">
              Forge: {status.last_forge_error}
            </p>
          )}
        </>
      )}
    </PageSection>
  );
}

function Capabilities({ status }: { status: KBStatus | null }) {
  const caps = Object.entries(status?.capabilities ?? {}).sort(([a], [b]) => a.localeCompare(b));
  return (
    <PageSection title="Capabilities" id="settings-capabilities" count={status ? caps.length : undefined}>
      {!status ? (
        <SkeletonRows label="Loading capabilities" />
      ) : (
        <table className="settings__table">
          <thead>
            <tr>
              <th scope="col">Capability</th>
              <th scope="col">State</th>
              <th scope="col">Setting</th>
            </tr>
          </thead>
          <tbody>
            {caps.map(([name, cap]) => (
              <tr key={name}>
                <th scope="row">{name.replace(/_/g, " ")}</th>
                <td>
                  <span
                    className={
                      cap.state === "enabled" ? "pill pill--ok" : cap.state === "disabled" ? "pill" : "pill pill--info"
                    }
                  >
                    {cap.state}
                  </span>
                  {cap.default && <span className="settings__default"> default</span>}
                  {cap.checks && cap.checks.length > 0 && (
                    <span className="settings__checks">{cap.checks.join(", ")}</span>
                  )}
                </td>
                <td>
                  <code>{cap.setting}</code>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PageSection>
  );
}

function MountedKBs({ kbs, current }: { kbs: KBSummary[]; current: string }) {
  return (
    <PageSection title="Knowledge bases on this server" id="settings-kbs" count={kbs.length}>
      <ul className="rows">
        {kbs.map((k) => (
          <li key={k.name} className="row">
            <span className={k.ready ? "pill pill--ok" : "pill pill--warning"}>{k.status}</span>
            <span className="row__body">
              <span className="row__title">{k.name}</span>
              <span className="row__meta">
                {k.capabilities?.secrets?.state === "enabled" && <span>secrets</span>}
                {k.capabilities?.artifact_write?.state === "enabled" && <span>artifact writes</span>}
                {k.capabilities?.write_gate && <span>write gate {k.capabilities.write_gate.state}</span>}
              </span>
            </span>
            <span className="row__end">{k.name === current ? "shown" : ""}</span>
          </li>
        ))}
      </ul>
    </PageSection>
  );
}

function Contracts({ list }: { list: MapContract[] | null }) {
  return (
    <PageSection title="Map contracts" id="settings-contracts" count={list?.length}>
      {list === null ? (
        <SkeletonRows label="Loading map contracts" />
      ) : list.length === 0 ? (
        <Quiet tone="neutral">No Map or Journal yet.</Quiet>
      ) : (
        <ul className="settings__contracts">
          {list.map((m) => {
            const fields = Object.entries(m.field_values ?? {});
            const byType = Object.entries(m.field_values_by_type ?? {});
            return (
              <li key={m.name} className="settings__contract">
                <div className="library__path-head">
                  <strong>{m.title || m.name}</strong>
                  <span className="pill">{m.kind || "map"}</span>
                  {m.ontology_mode && <span className="pill pill--info">{m.ontology_mode}</span>}
                </div>
                <Facts
                  items={[
                    ["Name", <code key="n">{m.name}</code>],
                    ["Concept types", m.concept_types?.length ? m.concept_types.join(", ") : "any"],
                    [
                      "Stale after",
                      m.stale_after ? `${m.stale_after} days${m.stale_after_defaulted ? " (default)" : ""}` : "—",
                    ],
                    ["Accepted checks", m.lint_ignore?.length ? m.lint_ignore.join(", ") : "none"],
                    ["Expanded concepts", String(m.expanded_concepts)],
                  ]}
                />
                {fields.length > 0 && (
                  <p className="settings__fields">
                    {fields.map(([field, values]) => (
                      <span key={field}>
                        <code>{field}</code>: {values.join(" · ")}
                      </span>
                    ))}
                  </p>
                )}
                {byType.map(([type, f]) => (
                  <p key={type} className="settings__fields">
                    <span className="pill">{type}</span>
                    {Object.entries(f).map(([field, values]) => (
                      <span key={field}>
                        <code>{field}</code>: {values.join(" · ")}
                      </span>
                    ))}
                  </p>
                ))}
              </li>
            );
          })}
        </ul>
      )}
    </PageSection>
  );
}

function Conflicts({ list, onOpenConcept }: { list: GitConflict[] | null; onOpenConcept(id: string): void }) {
  return (
    <PageSection title="Git conflicts" id="settings-conflicts" count={list?.length}>
      {list === null ? (
        <SkeletonRows label="Loading git conflicts" rows={2} />
      ) : list.length === 0 ? (
        <Quiet>No open rebase conflict.</Quiet>
      ) : (
        <ul className="rows">
          {list.map((c) => (
            <li key={c.concept_id + c.detected_at}>
              <button type="button" className="row" onClick={() => onOpenConcept(c.concept_id)}>
                <span className="pill pill--error">conflict</span>
                <span className="row__body">
                  <span className="row__title">{c.concept_id}</span>
                  <span className="row__meta">
                    <span>{c.branch}</span>
                    <span>{c.files.length} file(s)</span>
                    <span>{c.guidance}</span>
                  </span>
                </span>
                <span className="row__end">{relativeDay(c.detected_at)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </PageSection>
  );
}

function Journal({ text }: { text: string | null }) {
  return (
    <PageSection title="Journal" id="settings-journal">
      {text === null ? (
        <SkeletonRows label="Loading the journal" />
      ) : text.trim() === "" ? (
        <Quiet tone="neutral">The KB's log.md has no entries yet.</Quiet>
      ) : (
        <pre className="settings__journal" tabIndex={0} aria-label="Last entries of log.md">
          {text.trim()}
        </pre>
      )}
    </PageSection>
  );
}

function Audit({ log }: { log: AuditLog | null }) {
  return (
    <PageSection title="Audit log" id="settings-audit" count={log?.entries.length}>
      {log === null ? (
        <SkeletonRows label="Loading the audit log" />
      ) : !log.enabled ? (
        <Quiet tone="neutral">
          No audit sink on this server: set <code>audit.path</code> in config.yaml to record every tool call.
        </Quiet>
      ) : log.entries.length === 0 ? (
        <Quiet tone="neutral">No tool call recorded for this KB yet.</Quiet>
      ) : (
        <table className="settings__table settings__audit">
          <thead>
            <tr>
              <th scope="col">When</th>
              <th scope="col">Who</th>
              <th scope="col">Tool</th>
              <th scope="col">Outcome</th>
            </tr>
          </thead>
          <tbody>
            {log.entries.map((e, i) => (
              <tr key={i}>
                <td title={e.at}>{new Date(e.at).toLocaleString()}</td>
                <td>
                  <code>{e.principal || "—"}</code>
                </td>
                <td>
                  <code>{e.tool}</code>
                  {e.phase && <span className="settings__default"> {e.phase}</span>}
                  {e.resources && (
                    <span className="settings__checks">
                      {Object.entries(e.resources)
                        .map(([k, v]) => `${k}=${v}`)
                        .join(" ")}
                    </span>
                  )}
                </td>
                <td>
                  <span
                    className={e.outcome === "success" ? "pill pill--ok" : e.outcome ? "pill pill--warning" : "pill"}
                  >
                    {e.outcome || "—"}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PageSection>
  );
}
