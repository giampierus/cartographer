// Mirrors the server's /api/ui/v1 shapes (D226). Kept hand-written rather than
// generated: the surface is five routes, and a generator would be a build step
// nobody can run from `go install`.

export interface KBCapability {
  state: string;
  setting: string;
  /** The checks an auto_repair capability covers (D299). */
  checks?: string[];
  /** The setting is the product default, not a value the operator wrote (D323). */
  default?: boolean;
  /** auto_repair checks skipped unattended because their fix is not safe (D355). */
  ignored?: string[];
}

export interface KBSummary {
  name: string;
  status: string;
  ready: boolean;
  capabilities?: Record<string, KBCapability>;
  /** Whether this principal may open the Artifacts panel: artifacts are
   *  whole-KB resources (D238). */
  artifacts: boolean;
  /** Whether this principal may open the Secrets panel (whole-KB, like artifacts). */
  secrets?: boolean;
}

/** An agent client an artifact is materialized for. */
export interface ArtifactClient {
  id: string;
  name: string;
}

export interface ArtifactFile {
  path: string;
  sha256: string;
  size: number;
  executable: boolean;
  /** Detail route only: the UTF-8 text, absent when binary or truncated. */
  content?: string;
  binary?: boolean;
  truncated?: boolean;
}

export interface Artifact {
  kind: string;
  name: string;
  description?: string;
  content_hash?: string;
  /** Absent for instructions and templates, which sync does not sign. */
  signed?: boolean;
  clients: ArtifactClient[];
  files: ArtifactFile[];
  /** Concepts the artifact references explicitly ([[id]] or a written id). */
  concepts?: string[];
  /** Lint findings on this artifact's files (D316). */
  findings?: ArtifactFinding[];
  /**
   * When a client last activated the artifact (D326): ISO 8601, or null when
   * nothing reported. Absent from a server that predates the field.
   */
  last_used?: string | null;
  last_used_provider?: string | null;
  last_used_days_ago?: number | null;
  /** The only signal is a client loading a catalogue that lists it: available, not seen used. */
  last_used_catalog_only?: boolean;
}

/** One artifact lint finding (D316). */
export interface ArtifactFinding {
  path: string;
  check: string;
  severity: string;
  message: string;
}

export interface ArtifactList {
  artifacts: Artifact[];
  counts: Record<string, number>;
  /** Why a skill was left out of what the KB ships. */
  issues: string[];
  /** Every artifact finding by check, including those no listed artifact owns (D316). */
  finding_counts?: Record<string, number>;
  /** The same findings by severity. */
  finding_severities?: Record<string, number>;
}

export interface CollectionSummary {
  name: string;
  title?: string;
  kind: string;
  concepts: number;
  expanded_concepts: number;
}

export interface Overview {
  collections: CollectionSummary[];
  concepts: {
    total: number;
    by_type: Record<string, number>;
    by_status: Record<string, number>;
  };
  lint: {
    total: number;
    by_severity: Record<string, number>;
    by_check: Record<string, number>;
  };
  git?: { has_remote: boolean; remote_url?: string };
}

export interface GraphNode {
  id: string;
  collection?: string;
  /** The frontmatter title, when the concept has one: what the UI names a
   *  node by. The id is the fallback. */
  title?: string;
  type?: string;
  status?: string;
  expanded?: boolean;
  self_link?: boolean;
  in_degree: number;
  out_degree: number;
  /** Global PageRank on the caller's visible graph (D244). */
  pagerank?: number;
  /** Rank of the node's community in GraphSnapshot.communities (D244). */
  community?: number;
}

/** One community of the visible graph, computed by the server (D244). */
export interface SnapshotCommunity {
  /** 0-based: 0 is the largest. */
  rank: number;
  size: number;
  /** The member a legend names the community after. */
  anchor: string;
  /** The anchor's title: a Map's snapshot rarely holds the anchor's node. */
  anchor_title?: string;
  /** 1..12, or 0 for singletons and the long tail. */
  slot: number;
}

export interface GraphEdge {
  source: string;
  target: string;
}

export interface BrokenTarget {
  source: string;
  target: string;
}

export interface GraphSnapshot {
  nodes: GraphNode[];
  edges: GraphEdge[];
  broken?: BrokenTarget[];
  truncated?: boolean;
  total_nodes: number;
  total_edges: number;
  total_broken?: number;
  limit: number;
  /** Covers the whole visible graph, not only the returned nodes. */
  communities?: SnapshotCommunity[];
}

export interface OutlineEntry {
  level: number;
  title: string;
  bytes: number;
}

export interface Concept {
  id: string;
  title: string;
  collection: string;
  frontmatter: Record<string, unknown>;
  body: string;
  body_bytes: number;
  outline: OutlineEntry[];
  content_hash: string;
  outbound: string[];
  inbound: string[];
  broken: string[];
  /** Skills, agents and hooks that reference this concept explicitly; absent
   *  for a principal that cannot see the whole KB. */
  used_by?: { kind: string; name: string }[];
  /** Visible concepts whose superseded_by names this one (D744). */
  supersedes?: string[];
  /** This concept's successor, when the reader may see it. */
  superseded_by?: string;
}

export type Severity = "info" | "warning" | "error";

export interface LintFinding {
  path: string;
  concept?: string;
  check: string;
  severity: string;
  message: string;
  /** Who acts on it (D365): the background repair, or the doctor session. */
  handler?: "auto" | "doctor";
}

export interface LintReport {
  findings: LintFinding[];
  count: number;
  total: number;
  by_severity: Record<string, number>;
  by_check: Record<string, number>;
  /** Where each reported check can be accepted with lint_ignore (D313). */
  acceptability?: Record<string, "concept" | "map" | "artifact" | "none">;
  severity_min: string;
}

/** GET /kbs/{kb}/search: the search tool's own response. */
export interface SearchHit {
  id: string;
  title?: string;
  snippet?: string;
  score: number;
}
export interface SearchResponse {
  query: string;
  count: number;
  /** null when nothing matched. */
  results: SearchHit[] | null;
}

/** GET /kbs/{kb}/changes: changes_since's own response. */
export interface ConceptChange {
  id: string;
  change: string;
  last_at: string;
  /** The newest change in the window that was not the server's upkeep:
   *  when the content last changed. Absent when only upkeep touched it. */
  last_edit_at?: string;
  /** Created inside the window, whatever its newest change was. */
  added?: boolean;
  authors?: string[];
  ops?: string[];
  reasons?: string[];
}
export interface ChangesResponse {
  since: string;
  commit_count: number;
  truncated?: boolean;
  concepts: ConceptChange[];
}

/** GET /kbs/{kb}/work: work_list for the caller (D302). */
export interface WorkItem {
  text: string;
  section?: string;
  line: number;
}
export interface WorkEntry {
  id: string;
  title?: string;
  type?: string;
  map?: string;
  status?: string;
  open_phase: boolean;
  timestamp?: string;
  age_days?: number;
  stale: boolean;
  items: WorkItem[];
}
export interface WorkResponse {
  total: number;
  open_concepts: number;
  by_status: Record<string, number>;
  by_map: Record<string, number>;
  open_items: number;
  entries: WorkEntry[];
  next_offset?: number;
}

/** GET /kbs/{kb}/status: the parts of kb_status the home reads. Both are
 *  omitted by the server when empty. */
export interface KBStatus {
  open_gaps?: {
    total: number;
    by_kind?: Record<string, number>;
    recent?: { id: string; title?: string; kind?: string; subjects?: string[] }[];
  };
  /** resolved: the same search now finds something (D319); sorted last. */
  search_misses?: { query: string; count: number; last_seen?: string; resolved?: boolean }[];
  stale_count?: number;
  // The server and replication facts the Settings panel shows (D744): the
  // same kb_status an agent reads.
  kb?: string;
  total?: number;
  open_contradictions?: number;
  git_workflow?: string;
  has_remote?: boolean;
  remote_url?: string;
  git_sync?: boolean;
  push_state?: string;
  push_last_error?: string;
  unpushed_commits?: number;
  behind_commits?: number;
  last_push_at?: string;
  last_sync_in_at?: string;
  base_branch?: string;
  working_branch?: string;
  pr_number?: number;
  pr_url?: string;
  last_forge_error?: string;
  capabilities?: Record<string, KBCapability>;
  server_version?: string;
  latest_version?: string;
  sources?: Record<string, number>;
}

/** GET /kbs/{kb}/maintenance/summary (D323). */
export interface MaintenanceRepair {
  sha: string;
  at: string;
  subject: string;
  reason?: string;
  files: number;
  /** Made by the server's own heartbeat, not by a kb_repair someone ran. */
  background: boolean;
  /** The command that undoes it: shown, never run, by the Atlas. */
  revert: string;
}
export interface MaintenanceRun {
  at: string;
  skipped?: string;
  checks?: { check: string; applied: number; skipped?: number; commit?: string; error?: string }[];
}
/** One daily sample of the whole-KB lint totals (D370). */
export interface LintSample {
  at: string;
  total: number;
  by_severity: Record<string, number>;
  by_handler: Record<string, number>;
}
export interface MaintenanceSummary {
  auto_repair: { enabled: boolean; default: boolean; checks: string[]; interval_days: number };
  last_auto_repair: MaintenanceRun | null;
  last_doctor?: string;
  next_doctor?: string;
  doctor_interval_days: number;
  /** The background repair's runs of the last 30 days, newest first (D365). */
  runs?: MaintenanceRun[];
  /** One sample a day of the lint totals, oldest first, 90 days at most (D370). */
  lint_history?: LintSample[];
  repairs: MaintenanceRepair[];
}

/** GET /kbs/{kb}/maintenance/review (D704): the doctor's agenda beyond lint. */
export interface ReviewItem {
  kind: string;
  concepts: string[];
  evidence: string;
  suggested_action: string;
  term?: string;
  template_slug?: string;
  weight: number;
}
export interface MaintenanceReview {
  items: ReviewItem[];
}

/** GET /kbs/{kb}/checks: every lint check the server runs (D365). */
export interface CheckCatalog {
  categories: string[];
  checks: {
    name: string;
    category: string;
    severity: string;
    fixable: boolean;
    auto: boolean;
    /** False when the check cannot run on this KB (D371): `reason` says why, and a zero would not mean clean. */
    active?: boolean;
    reason?: string;
  }[];
}

/** One sops-encrypted value: the ciphertext and what the browser needs to open it. */
export interface SecretEntry {
  /** RFC 6901 pointer, list indices included. */
  pointer: string;
  /** The additional data sops authenticated the value with. */
  aad_path: string;
  /** The raw ENC[AES256_GCM,...] string. */
  enc: string;
  referenced_by: string[];
}

export interface SecretFile {
  path: string;
  entries: SecretEntry[];
  recipients: string[];
  age_stanzas: string[];
  lastmodified?: string;
  source_of: string[];
  error?: string;
}

export interface SecretList {
  files: SecretFile[];
}

/** GET /kbs/{kb}/contracts (D744): map_list, every visible Map and Journal
 *  with its contract (D107, D229, D343, D346). */
export interface MapContract {
  name: string;
  title?: string;
  kind?: string;
  ontology_mode?: string;
  concept_types?: string[];
  lint_ignore?: string[];
  expanded_concepts: number;
  stale_after?: number;
  stale_after_defaulted?: boolean;
  field_values?: Record<string, string[]>;
  field_values_by_type?: Record<string, Record<string, string[]>>;
}

/** GET /kbs/{kb}/sources: the source ledger. */
export interface SourceEntry {
  id: string;
  title?: string;
  source_kind?: string;
  ingest_status: string;
  timestamp?: string;
  cited_by: number;
  cited_by_ids: string[];
}

/** GET /kbs/{kb}/contradictions: the non-gap kinds, which block commit_gate. */
export interface ContradictionEntry {
  id: string;
  title?: string;
  involves?: string[];
  contradiction_kind?: string;
  resolution_status: string;
}

export interface ServiceEntry {
  id: string;
  title?: string;
  status?: string;
}

/** GET /kbs/{kb}/conflicts: open git rebase conflicts. */
export interface GitConflict {
  concept_id: string;
  path: string;
  branch: string;
  files: string[];
  detected_at: string;
  kind?: string;
  guidance: string;
  pr_url?: string;
}

/** GET /kbs/{kb}/history?id=: concept_history. */
export interface ConceptRevision {
  sha: string;
  at: string;
  author: string;
  subject: string;
  path: string;
  agent?: string;
  reason?: string;
}
export interface ConceptHistory {
  id: string;
  revisions: ConceptRevision[];
  truncated: boolean;
  note?: string;
}

export interface AssetEntry {
  path: string;
  size: number;
  sha256: string;
  executable: boolean;
  oversized?: boolean;
}

/** GET /kbs/{kb}/paths (issue 742): the declared side of the path placeholders. */
export interface PathKey {
  id: string;
  kind: string;
  key: string;
  declared: boolean;
  description?: string;
  default?: string;
  remote?: string;
  cited_by: string[];
  artifacts: string[];
  conflicts?: { kb: string; default?: string; remote?: string }[];
}
export interface PathMapping {
  present: boolean;
  unparseable: boolean;
  malformed: string[];
  keys: PathKey[];
}

/** GET /kbs/{kb}/audit: admin only. */
export interface AuditEntry {
  at: string;
  phase?: string;
  principal?: string;
  transport?: string;
  tool: string;
  read_only?: boolean;
  resources?: Record<string, string>;
  outcome: string;
  duration_ms?: number;
  commit_sha?: string;
}
export interface AuditLog {
  enabled: boolean;
  entries: AuditEntry[];
}

/** GET /whoami: who is looking. */
export interface WhoAmI {
  id: string;
  admin: boolean;
  permissions: { kb: string; write: boolean; maps?: string[]; journals?: string[]; types?: string[] }[];
}

/** GET /config: one setting of the server, with its default and how to set it. */
export interface ConfigEntry {
  path: string;
  key: string;
  value: unknown;
  default: string;
  is_default: boolean;
  doc?: string;
  values?: string[];
  env?: string;
  flag?: string;
  secret: boolean;
}

export interface ConfigItem {
  title: string;
  entries: ConfigEntry[];
}

export interface ConfigSection {
  key: string;
  title: string;
  doc?: string;
  entries?: ConfigEntry[];
  items?: ConfigItem[];
}

export interface ServerConfig {
  source: string;
  yaml: string;
  sections: ConfigSection[];
}
