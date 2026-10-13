import { useState } from "react";
import { fetchServerConfig, fetchWhoAmI } from "../api/client";
import type { ConfigEntry, ConfigSection, ServerConfig, WhoAmI } from "../api/types";
import { STATIC_MODE } from "../api/static";
import { useResource } from "../lib/useResource";
import { Page, PageHeader, PageSection, Quiet, SkeletonRows } from "./Page";
import { ErrorState } from "./States";

type Mode = "overview" | "yaml";

/**
 * The Settings panel (issue 743, D744): the server's configuration, read-only,
 * to show how Cartographer can be set up. Two views of the same data: an
 * overview (every setting with its default, accepted values, env var and
 * flag) and the commented YAML. Admin only; nothing here writes — a setting
 * changes in the config and takes effect when the server restarts.
 */
export function Settings({
  reloadKey,
  onRetry,
  onFailure,
}: {
  reloadKey: string;
  onRetry(): void;
  onFailure(err: unknown): boolean;
}) {
  const [mode, setMode] = useState<Mode>("overview");
  const config = useResource(STATIC_MODE ? null : reloadKey, (s) => fetchServerConfig(s), onFailure);
  const who = useResource(STATIC_MODE ? null : reloadKey, (s) => fetchWhoAmI(s), onFailure);

  const error = config.error ?? who.error;
  if (error) return <ErrorState error={error} onRetry={onRetry} />;
  const cfg = config.data;

  return (
    <Page label="Settings" className="settings" busy={config.loading}>
      <PageHeader
        eyebrow="Settings"
        title="How the server is configured."
        subtitle={
          <>
            Read-only. Configuration comes from{" "}
            {cfg?.source ? <code>{cfg.source}</code> : "flags, environment and defaults"}; a change takes effect when
            the server restarts.
          </>
        }
        help={
          <p>
            Each setting names the <code>config.yaml</code> key, the environment variable and the flag that set it. The
            same values, commented, are in the YAML view.
          </p>
        }
      />
      {!who.absent && <WhoLine who={who.data} />}
      {config.absent ? (
        <Quiet tone="neutral">Only an admin can see the server's configuration.</Quiet>
      ) : !cfg ? (
        <SkeletonRows label="Loading the configuration" rows={6} />
      ) : (
        <>
          <div className="settings__toggle" role="group" aria-label="View">
            {(["overview", "yaml"] as const).map((m) => (
              <button
                key={m}
                type="button"
                className="settings__toggle-btn"
                aria-pressed={mode === m}
                onClick={() => setMode(m)}
              >
                {m === "overview" ? "Overview" : "YAML"}
              </button>
            ))}
          </div>
          {mode === "overview" ? (
            cfg.sections.map((s) => <Section key={s.key} section={s} />)
          ) : (
            <YamlView config={cfg} />
          )}
        </>
      )}
    </Page>
  );
}

function WhoLine({ who }: { who: WhoAmI | null }) {
  if (!who) return null;
  return (
    <p className="settings__who">
      <span className="settings__who-label">Who is looking</span>
      <code>{who.id || "anonymous"}</code>
      <span>{who.admin ? "admin: every KB, read and write" : `${who.permissions.length} rule(s)`}</span>
    </p>
  );
}

function Section({ section }: { section: ConfigSection }) {
  return (
    <PageSection title={section.title} id={`settings-${section.key}`}>
      {section.doc && <p className="settings__doc">{section.doc}</p>}
      {section.entries && section.entries.length > 0 && <Entries entries={section.entries} />}
      {section.items?.map((item, i) => (
        <div key={`${item.title}:${i}`} className="settings__contract">
          <strong>{item.title}</strong>
          <Entries entries={item.entries} />
        </div>
      ))}
    </PageSection>
  );
}

function Entries({ entries }: { entries: ConfigEntry[] }) {
  return (
    <ul className="settings__entries">
      {entries.map((e) => (
        <Entry key={e.path} entry={e} />
      ))}
    </ul>
  );
}

function isEmpty(v: unknown): boolean {
  return v === null || v === undefined || v === "" || (Array.isArray(v) && v.length === 0);
}

function Value({ entry }: { entry: ConfigEntry }) {
  const v = entry.value;
  if (isEmpty(v)) return <span className="settings__unset">not set</span>;
  if (typeof v === "boolean") return <span className={v ? "pill pill--ok" : "pill"}>{v ? "on" : "off"}</span>;
  if (Array.isArray(v)) {
    return (
      <span className="settings__chips">
        {v.map((x, i) => (
          <span key={i} className="settings__chip">
            {String(x)}
          </span>
        ))}
      </span>
    );
  }
  return <code className="settings__value">{typeof v === "object" ? JSON.stringify(v) : String(v)}</code>;
}

function Entry({ entry }: { entry: ConfigEntry }) {
  const current = typeof entry.value === "string" ? entry.value : null;
  return (
    <li className={entry.is_default ? "settings__entry" : "settings__entry settings__entry--set"}>
      <div className="settings__entry-head">
        <code className="settings__key">{entry.key}</code>
        <Value entry={entry} />
        {entry.is_default && <span className="settings__default">default</span>}
      </div>
      {entry.doc && <p className="settings__entry-doc">{entry.doc}</p>}
      {entry.values && entry.values.length > 0 && (
        <span className="settings__chips" aria-label="Accepted values">
          {entry.values.map((x) => (
            <span key={x} className={x === current ? "settings__chip settings__chip--on" : "settings__chip"}>
              {x}
            </span>
          ))}
        </span>
      )}
      {(entry.env || entry.flag) && (
        <span className="settings__how">
          {entry.env && <code>{entry.env}</code>}
          {entry.flag && <code>{entry.flag}</code>}
        </span>
      )}
    </li>
  );
}

function YamlView({ config }: { config: ServerConfig }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    void navigator.clipboard
      ?.writeText(config.yaml)
      .then(() => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1500);
      })
      .catch(() => undefined);
  };
  return (
    <PageSection
      title="config.yaml"
      id="settings-yaml"
      actions={
        <button type="button" className="chip" onClick={copy}>
          {copied ? "Copied" : "Copy"}
        </button>
      }
    >
      <pre className="settings__yaml" tabIndex={0} aria-label="Server configuration as YAML">
        {config.yaml.split("\n").map((line, i) => (
          <span key={i} className={line.trimStart().startsWith("#") ? "settings__yaml-comment" : undefined}>
            {line}
            {"\n"}
          </span>
        ))}
      </pre>
    </PageSection>
  );
}
