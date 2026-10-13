import { fetchContradictions, fetchPaths, fetchServices, fetchSources } from "../api/client";
import type { ContradictionEntry, PathKey, PathMapping, ServiceEntry, SourceEntry } from "../api/types";
import { useResource } from "../lib/useResource";
import { Count, Figures, Hero, Page, PageHeader, PageSection, Quiet, SkeletonRows } from "./Page";
import { ErrorState } from "./States";

/**
 * The Library panel (D744): the records a KB keeps besides its graph — the
 * contradictions that block commit_gate, the source ledger, the services, and
 * the path placeholders it declares (issue 742). Each section reads the route that
 * mirrors its MCP tool; a section the reader may not see (a 404) is left out.
 */
export function Library({
  kb,
  reloadKey,
  titleOf = () => undefined,
  onOpenConcept,
  onRetry,
  onFailure,
}: {
  kb: string;
  reloadKey: string;
  titleOf?(id: string): string | undefined;
  onOpenConcept(id: string): void;
  onRetry(): void;
  onFailure(err: unknown): boolean;
}) {
  const key = `${kb}:${reloadKey}`;
  const contradictions = useResource(key, (s) => fetchContradictions(kb, s), onFailure);
  const sources = useResource(key, (s) => fetchSources(kb, s), onFailure);
  const services = useResource(key, (s) => fetchServices(kb, s), onFailure);
  const paths = useResource(key, (s) => fetchPaths(kb, s), onFailure);

  const error = contradictions.error ?? sources.error ?? services.error ?? paths.error;
  if (error) return <ErrorState error={error} onRetry={onRetry} />;

  const open = (contradictions.data?.contradictions ?? []).filter((c) => c.resolution_status === "open");
  const pending = (sources.data ?? []).filter((s) => s.ingest_status === "pending");
  const loading = contradictions.loading || sources.loading || services.loading || paths.loading;

  return (
    <Page label="Library" className="library" busy={loading}>
      <PageHeader
        eyebrow="Library"
        title={
          open.length > 0 ? (
            <>
              <Count>{open.length}</Count> open contradiction{open.length === 1 ? " blocks" : "s block"} the commit
              gate.
            </>
          ) : (
            "What the KB keeps besides its graph."
          )
        }
        subtitle="Contradictions, the source ledger, the services and the path placeholders, read with the same tools an agent uses."
        help={
          <p>
            Each section mirrors an MCP tool: <code>contradiction_report</code>, <code>source_list</code>,{" "}
            <code>service_list</code> and the KB's <code>paths.yaml</code>. Nothing here writes; an agent registers a
            source with <code>source_register</code> and resolves a contradiction with <code>conflict_resolve</code>.
          </p>
        }
      />
      <Hero label="Library in numbers">
        <Figures
          items={[
            { label: "Open contradictions", value: open.length, tone: open.length > 0 ? "warning" : undefined },
            { label: "Sources", value: sources.data?.length ?? "—" },
            { label: "Pending ingest", value: pending.length },
            { label: "Services", value: services.data?.services.length ?? "—" },
            { label: "Placeholders", value: paths.data?.keys.length ?? "—" },
          ]}
        />
      </Hero>

      {!contradictions.absent && (
        <Contradictions
          list={contradictions.data?.contradictions ?? null}
          titleOf={titleOf}
          onOpenConcept={onOpenConcept}
        />
      )}
      {!sources.absent && <Sources list={sources.data} onOpenConcept={onOpenConcept} />}
      {!services.absent && <Services list={services.data?.services ?? null} onOpenConcept={onOpenConcept} />}
      {!paths.absent && <Paths mapping={paths.data} onOpenConcept={onOpenConcept} />}
    </Page>
  );
}

function Contradictions({
  list,
  titleOf,
  onOpenConcept,
}: {
  list: ContradictionEntry[] | null;
  titleOf(id: string): string | undefined;
  onOpenConcept(id: string): void;
}) {
  return (
    <PageSection title="Contradictions" id="library-contradictions" count={list?.length}>
      {list === null ? (
        <SkeletonRows label="Loading contradictions" />
      ) : list.length === 0 ? (
        <Quiet>No two concepts disagree. Knowledge gaps are in Health.</Quiet>
      ) : (
        <ul className="rows">
          {list.map((c) => (
            <li key={c.id}>
              <button type="button" className="row" onClick={() => onOpenConcept(c.id)}>
                <span className="library__glyph" aria-hidden="true">
                  ≠
                </span>
                <span className="row__body">
                  <span className="row__title">{c.title || c.id}</span>
                  <span className="row__meta">
                    <span className={c.resolution_status === "open" ? "pill pill--warning" : "pill pill--ok"}>
                      {c.resolution_status}
                    </span>
                    {c.contradiction_kind && <span className="pill">{c.contradiction_kind.replace(/_/g, " ")}</span>}
                    {(c.involves ?? []).map((id) => (
                      <span key={id} className="library__involves">
                        {titleOf(id) ?? id}
                      </span>
                    ))}
                  </span>
                </span>
                <span className="row__end">Open concept</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </PageSection>
  );
}

function Sources({ list, onOpenConcept }: { list: SourceEntry[] | null; onOpenConcept(id: string): void }) {
  return (
    <PageSection title="Source ledger" id="library-sources" count={list?.length}>
      {list === null ? (
        <SkeletonRows label="Loading sources" />
      ) : list.length === 0 ? (
        <Quiet tone="neutral">No source registered yet. An agent records one with source_register.</Quiet>
      ) : (
        <ul className="rows">
          {list.map((s) => (
            <li key={s.id}>
              <button type="button" className="row" onClick={() => onOpenConcept(s.id)}>
                <span className="library__glyph" aria-hidden="true">
                  ❝
                </span>
                <span className="row__body">
                  <span className="row__title">{s.title || s.id}</span>
                  <span className="row__meta">
                    <span
                      className={
                        s.ingest_status === "ingested"
                          ? "pill pill--ok"
                          : s.ingest_status === "pending"
                            ? "pill pill--warning"
                            : "pill"
                      }
                    >
                      {s.ingest_status}
                    </span>
                    {s.source_kind && <span className="pill">{s.source_kind}</span>}
                    {s.timestamp && <time dateTime={s.timestamp}>{day(s.timestamp)}</time>}
                  </span>
                </span>
                <span className="row__end">
                  cited by {s.cited_by} concept{s.cited_by === 1 ? "" : "s"}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </PageSection>
  );
}

function Services({ list, onOpenConcept }: { list: ServiceEntry[] | null; onOpenConcept(id: string): void }) {
  return (
    <PageSection title="Services" id="library-services" count={list?.length}>
      {list === null ? (
        <SkeletonRows label="Loading services" />
      ) : list.length === 0 ? (
        <Quiet tone="neutral">No concept of type Service.</Quiet>
      ) : (
        <ul className="rows">
          {list.map((s) => (
            <li key={s.id}>
              <button type="button" className="row" onClick={() => onOpenConcept(s.id)}>
                <span className="library__glyph" aria-hidden="true">
                  ⚙
                </span>
                <span className="row__body">
                  <span className="row__title">{s.title || s.id}</span>
                  <span className="row__meta">
                    {s.status && <span className="pill">{s.status}</span>}
                    <code>{s.id}</code>
                  </span>
                </span>
                <span className="row__end">Open concept</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </PageSection>
  );
}

/** A timestamp as the other pages say dates; the raw text when it does not parse. */
function day(iso: string): string {
  const t = Date.parse(iso);
  return Number.isNaN(t)
    ? iso
    : new Date(t).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
}

/** The `{{kind:key}}` form a concept writes. */
const placeholder = (k: PathKey) => `{{${k.kind}:${k.key}}}`;

function Paths({ mapping, onOpenConcept }: { mapping: PathMapping | null; onOpenConcept(id: string): void }) {
  const keys = mapping?.keys ?? [];
  const undeclared = keys.filter((k) => !k.declared);
  return (
    <PageSection title="Path placeholders" id="library-paths" count={mapping ? keys.length : undefined}>
      {mapping === null ? (
        <SkeletonRows label="Loading path placeholders" />
      ) : (
        <>
          <p className="page-note">
            What <code>paths.yaml</code> declares. Each machine resolves a key to its own local path: run{" "}
            <code>cartographer paths list</code> on yours to see where they point.
          </p>
          {!mapping.present && keys.length > 0 && (
            <p className="library__warn" role="note">
              The KB cites {keys.length} placeholder{keys.length === 1 ? "" : "s"} but has no paths.yaml.
            </p>
          )}
          {mapping.malformed.map((m) => (
            <p key={m} className="library__warn" role="note">
              {m}
            </p>
          ))}
          {undeclared.length > 0 && mapping.present && (
            <p className="library__warn" role="note">
              {undeclared.length} cited but not declared: {undeclared.map(placeholder).join(", ")}
            </p>
          )}
          {keys.length === 0 ? (
            <Quiet tone="neutral">No placeholder declared or cited.</Quiet>
          ) : (
            <ul className="library__paths">
              {keys.map((k) => (
                <li key={k.id} className="library__path">
                  <div className="library__path-head">
                    <code className="library__placeholder">{placeholder(k)}</code>
                    <span className={k.declared ? "pill pill--ok" : "pill pill--warning"}>
                      {k.declared ? "declared" : "undeclared"}
                    </span>
                    {k.declared && k.cited_by.length === 0 && k.artifacts.length === 0 && (
                      <span className="pill">unused</span>
                    )}
                  </div>
                  {k.description && <p className="library__path-desc">{k.description}</p>}
                  <dl className="library__path-facts">
                    {k.default && (
                      <>
                        <dt>Default</dt>
                        <dd>
                          <code>{k.default}</code>
                        </dd>
                      </>
                    )}
                    {k.remote && (
                      <>
                        <dt>Remote</dt>
                        <dd>
                          <code>{k.remote}</code>
                        </dd>
                      </>
                    )}
                    {k.artifacts.length > 0 && (
                      <>
                        <dt>Artifacts</dt>
                        <dd>
                          {k.artifacts.map((a) => (
                            <code key={a} className="library__file">
                              {a}
                            </code>
                          ))}
                        </dd>
                      </>
                    )}
                  </dl>
                  {k.cited_by.length > 0 && (
                    <div className="library__cited">
                      <span className="library__cited-label">Cited by</span>
                      {k.cited_by.map((id) => (
                        <button key={id} type="button" className="chip" onClick={() => onOpenConcept(id)}>
                          {id}
                        </button>
                      ))}
                    </div>
                  )}
                  {(k.conflicts ?? []).map((c) => (
                    <p key={c.kb} className="library__warn" role="note">
                      {c.kb} declares it too, with {c.default ? <code>{c.default}</code> : "no default"}
                      {c.remote ? (
                        <>
                          {" "}
                          and <code>{c.remote}</code>
                        </>
                      ) : null}
                      : a machine mounting both resolves it once.
                    </p>
                  ))}
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </PageSection>
  );
}
