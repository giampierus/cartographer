import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import { json, stubApi } from "./fixtures";

const notFound = () => json({ error: { code: "not_found", message: "not found" } }, 404);

/** The Library and Settings panels and the Inspector's History tab (D744). */
describe("the explore panels", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("cartographer.panel.rail", "0");
    sessionStorage.clear();
  });
  afterEach(() => vi.unstubAllGlobals());

  it("Library lists contradictions, sources, services and the path placeholders", async () => {
    window.history.replaceState(null, "", "/ui/?kb=kb-a&panel=library");
    stubApi({
      "/contradictions": () =>
        json({
          contradictions: [
            {
              id: "infra/clash",
              title: "Clash",
              contradiction_kind: "factual",
              resolution_status: "open",
              involves: ["infra/a"],
            },
          ],
        }),
      "/sources": () =>
        json([
          {
            id: "notes/paper",
            title: "A paper",
            ingest_status: "pending",
            source_kind: "url",
            cited_by: 2,
            cited_by_ids: ["infra/a", "infra/b"],
          },
        ]),
      "/services": () => json({ services: [{ id: "infra/a", title: "Alpha", status: "active" }] }),
      "/paths": () =>
        json({
          present: true,
          unparseable: false,
          malformed: [],
          keys: [
            {
              id: "repo:api",
              kind: "repo",
              key: "api",
              declared: true,
              default: "$HOME/src/api",
              cited_by: ["infra/a"],
              artifacts: ["skills/deploy/SKILL.md"],
              conflicts: [{ kb: "work-kb", default: "$HOME/work/api" }],
            },
            { id: "path:docs", kind: "path", key: "docs", declared: false, cited_by: ["infra/b"], artifacts: [] },
          ],
        }),
    });
    render(<App />);
    expect(await screen.findByText(/open contradiction/)).toBeInTheDocument();
    expect(await screen.findByText("Clash")).toBeInTheDocument();
    expect(screen.getByText("A paper")).toBeInTheDocument();
    expect(screen.getByText("cited by 2 concepts")).toBeInTheDocument();
    expect(screen.getByText("{{repo:api}}")).toBeInTheDocument();
    expect(screen.getByText("$HOME/src/api")).toBeInTheDocument();
    expect(screen.getByText(/cited but not declared: \{\{path:docs\}\}/)).toBeInTheDocument();
    expect(screen.getByText(/work-kb declares it too/)).toBeInTheDocument();
    expect(screen.getByText("cartographer paths list")).toBeInTheDocument();
  });

  it("Library leaves out a section the reader may not see", async () => {
    window.history.replaceState(null, "", "/ui/?kb=kb-a&panel=library");
    stubApi({
      "/contradictions": () => json({ contradictions: [] }),
      "/sources": () => json([]),
      "/services": () => json({ services: [] }),
      "/paths": notFound,
    });
    render(<App />);
    expect(await screen.findByRole("heading", { name: /Source ledger/ })).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("heading", { name: /Path placeholders/ })).not.toBeInTheDocument());
    expect(screen.getByText(/No two concepts disagree/)).toBeInTheDocument();
  });

  it("Settings shows the server, the capabilities, who is looking and the audit log", async () => {
    window.history.replaceState(null, "", "/ui/?kb=kb-a&panel=settings");
    stubApi({
      "/status": () =>
        json({
          server_version: "1.2.3",
          has_remote: true,
          remote_url: "gitlab.example.com/team/kb-a",
          git_workflow: "local",
          git_sync: true,
          push_state: "clean",
          unpushed_commits: 0,
          behind_commits: 0,
          capabilities: {
            secrets: { state: "enabled", setting: "kbs[].sops_age_key_file or sops.age_key_file" },
            write_gate: { state: "warning", setting: "kbs[].write_gate", default: true },
          },
        }),
      "/whoami": () => json({ id: "user", admin: false, permissions: [{ kb: "kb-a", write: false, maps: ["infra"] }] }),
      "/contracts": () =>
        json([{ name: "infra", title: "Infrastructure", kind: "map", ontology_mode: "strict", expanded_concepts: 1 }]),
      "/conflicts": () => json([]),
      "/journal": () => json({ text: "- 2026-10-01 created infra/a\n" }),
      "/audit": () =>
        json({
          enabled: true,
          entries: [{ at: "2026-10-01T10:00:00Z", principal: "user", tool: "concept_read", outcome: "success" }],
        }),
    });
    render(<App />);
    expect(await screen.findByText("Cartographer 1.2.3, serving kb-a.")).toBeInTheDocument();
    expect(screen.getByText("gitlab.example.com/team/kb-a")).toBeInTheDocument();
    expect(screen.getByText("kbs[].write_gate")).toBeInTheDocument();
    expect(await screen.findByText("maps: infra")).toBeInTheDocument();
    const contracts = screen.getByRole("heading", { name: /Map contracts/ }).closest("section")!;
    expect(within(contracts).getByText("Infrastructure")).toBeInTheDocument();
    expect(within(contracts).getByText("strict")).toBeInTheDocument();
    expect(screen.getByText("No open rebase conflict.")).toBeInTheDocument();
    expect(screen.getByLabelText("Last entries of log.md")).toHaveTextContent("created infra/a");
    const audit = screen.getByRole("heading", { name: /Audit log/ }).closest("section")!;
    expect(within(audit).getByText("concept_read")).toBeInTheDocument();
  });

  it("the rail offers both panels", async () => {
    window.history.replaceState(null, "", "/ui/?kb=kb-a");
    stubApi();
    render(<App />);
    const rail = await screen.findByRole("navigation", { name: "Atlas navigation" });
    expect(within(rail).getByRole("button", { name: "Library" })).toBeInTheDocument();
    expect(within(rail).getByRole("button", { name: "Settings" })).toBeInTheDocument();
  });

  it("the Inspector's History tab reads the timeline and the assets of the concept", async () => {
    window.history.replaceState(null, "", "/ui/?kb=kb-a&concept=infra/a");
    const fetchMock = stubApi({
      "/history": () =>
        json({
          id: "infra/a",
          truncated: false,
          revisions: [
            {
              sha: "abcdef1234",
              at: "2026-10-01T10:00:00Z",
              author: "user",
              subject: "docs: describe alpha",
              path: "infra/a.md",
              reason: "first draft",
            },
          ],
        }),
      "/assets": () => json({ assets: [{ path: "diagram.svg", size: 2048, sha256: "x", executable: false }] }),
    });
    const user = userEvent.setup();
    render(<App />);
    const tab = await screen.findByRole("tab", { name: "History" });
    expect(fetchMock.mock.calls.some(([url]) => String(url).includes("/history"))).toBe(false);
    await user.click(tab);
    expect(await screen.findByText("docs: describe alpha")).toBeInTheDocument();
    expect(screen.getByText("first draft")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("diagram.svg")).toBeInTheDocument());
    expect(screen.getByText("2.0 KB")).toBeInTheDocument();
  });
});
