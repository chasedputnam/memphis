# Memphis

<p align="center">
<img width="704" alt="memphis_github" src="https://github.com/user-attachments/assets/851ab1f8-9412-497c-8194-0b4cb4c4ea61" />
</p>

> "In Memphis was founded one of the most important monuments of the world, and the only surviving wonder of the ancient world, namely, the Great Pyramid of Giza."

**Memphis (MEM-phis) is an enforceable authority layer for AI coding agents.** It is a single Go binary that turns the decisions, requirements, and designs your team agrees to into **Canon**: typed Markdown artifacts, validated against real standards, wired into a blocking **gate**, and served to agents over **MCP**. No decision goes unrecorded, and no agent silently violates one.

The problem memphis solves is simple to state and expensive to ignore: an AI agent's only real constraint is its context window, and most "memory" tools conflate two very different properties. **Authority** asks whether something is the canonical truth the team agreed to. **Discoverability** asks whether the right piece can be found at the right moment. Vector stores optimize discoverability and have no concept of authority. memphis makes authority a first-class, *enforced* property: Canon artifacts are typed, their relationships are integrity-checked, and a deterministic gate rejects malformed or conflicting authority before it lands, with no LLM and no network in that path.

memphis is built for **spec-driven development**. The specs your workflow already produces (`requirements.md`, `design.md`) become typed Canon with one command, and the agents that read them are held to that Canon automatically, whether you drive Claude Code's `/spec → /dev → /code-review` skills, Kiro's specs and agent hooks, or any MCP client.

> **Memory is Canon. Context is the budgeted projection of Canon (and optional Reference). AI lives only in the projection. The substrate is Git.**

The Canon authority model is a faithful Go port of the **Requirements-as-Code** (rac-core) engine, and the two interoperate through the Open Knowledge Format. For the full design, see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## Why memphis: one scenario

You make an architectural decision and record it as Canon:

```bash
memphis new decision canon/adr-001-use-bleve.md --title "Use Bleve for search"
$EDITOR canon/adr-001-use-bleve.md      # fill ## Status (Accepted), ## Decision, ## Consequences
memphis gate .                          # validates structure, standards, relationship integrity
```

Two weeks later, in a brand-new session with no memory of that conversation, an agent proposes ripping out Bleve for a vector database. Because memphis is wired into the workflow:

- The agent **grounds on Canon over MCP** first. `find_decisions` surfaces the "Use Bleve for search" artifact with status **Accepted** and its consequences, so the agent argues *from* the decision instead of around it.
- If a change still lands that contradicts Accepted Canon, the **gate blocks it** at commit time (git hook) and in CI (`gate --sarif`), citing the exact artifact and rule.
- When the decision genuinely *should* change, you mint a successor that `## Supersedes` the old one. memphis follows the supersede chain so agents always see the current truth, never a stale one.

That is the whole point: **the decision is recorded once and respected thereafter**, by people and agents alike, without anyone having to remember it.

---

## The core model

memphis gives an agent two kinds of knowledge over one substrate, plain Markdown plus YAML frontmatter, versioned in Git:

| | **Canon** (authority) | **Reference** (recall, optional) |
|---|---|---|
| Answers | **What is true**: what the team decided and what must hold | **How things work**: supporting documentation |
| Content | Requirements, decisions, designs, roadmaps, prompts | Ingested docs (crawled sites, imported repos) |
| Created by | `memphis new` / `memphis project` / `memphis promote` | `memphis crawl` / `memphis import` |
| Validation | Typed, standards-checked, relationship-integrity-checked, **gated in CI** | Permissive, abundant and searchable |
| Determinism | Pure function of repo state, **no LLM, no network** | AI may summarize and rank in the discovery layer |

A **store** is one directory in Git holding both tiers. The only thing separating them is `canon_roots` in `.okf/config.yaml`: files under those roots are Canon, and everything else is Reference. Canon is the hero of memphis. Reference is an optional convenience for teams that also want a large docs corpus searchable as agent memory (see the [Appendix](#appendix-reference-tier-and-okf-format)).

### The five Canon artifact types

Each is typed Markdown with required sections, a minted opaque ID (`<repository-key>-<12-char Crockford base32>`, for example `OKF-KTQ63DPSMF19`), a lifecycle status, and typed relationships (`## Related <Type>`, `## Supersedes`).

| Type | Captures | Key required sections |
|---|---|---|
| `requirement` | What must hold | `## Problem`, `## Requirements` (`[REQ-NNN] … SHALL …`) |
| `decision` | A choice and its rationale (ADR) | `## Context`, `## Decision`, `## Consequences` |
| `design` | How something is built | `## Context`, `## User Need`, `## Design`, `## Constraints` |
| `roadmap` | Intended outcomes over time | `## Outcomes`, `## Initiatives` |
| `prompt` | A reusable, versioned prompt | `## Objective`, `## Input`, `## Instructions`, `## Output` |

### The gate

`memphis gate` is the enforcement mechanism and the heart of the authority model. It loads the corpus, validates every artifact, checks relationship integrity, applies your enforcement policy, and exits non-zero on any blocking finding, emitting SARIF for required-checks. It is **deterministic and offline** (a build-failing test forbids `net/http` or any LLM dependency in the authority path), so it is safe in pre-commit hooks and CI. Validation includes:

- **BCP-14 / RFC 8174**: only ALL-CAPS `MUST`/`SHALL`/`SHOULD` carry normative weight.
- **ISO/IEC/IEEE 29148**: requirements should be singular and testable.
- **EARS**: Easy Approach to Requirements Syntax conformance.
- **Relationship integrity**: no dangling, ambiguous, miscast, or cyclic references, and live artifacts don't depend on retired ones (except via `## Supersedes`).

---

## Installation

### Download a binary

Download the latest binary for your platform from the [releases page](https://github.com/chasedputnam/memphis/releases).

### Build from source

```bash
go install github.com/chasedputnam/memphis/cmd/memphis@latest
```

Or clone and build (Go 1.25+):

```bash
git clone https://github.com/chasedputnam/memphis.git
cd memphis
make build
```

> **Apple Intelligence (optional, Reference summaries only):** on macOS 26 Tahoe with Apple Silicon, memphis can summarize Reference docs through Apple's on-device Foundation Models via the opt-in `applefm` build tag. See [docs/APPLE_INTELLIGENCE.md](docs/APPLE_INTELLIGENCE.md). This never touches the Canon authority path.

---

## Quick start

```bash
# 1. Scaffold a store (writes .okf/config.yaml + canon roots)
memphis init my-store && cd my-store
git init

# 2. Get Canon in. Author it directly:
memphis new decision canon/adr-001-use-bleve.md --title "Use Bleve for search"
$EDITOR canon/adr-001-use-bleve.md

#    Or project an approved spec into Canon (one command, no re-authoring):
memphis project specs/search/requirements.md

# 3. Enforce it
memphis gate .                      # blocks on any structural, standards, or integrity failure

# 4. Wire the gate into your tools so it runs automatically
memphis hooks install               # git + detected agent toolchains (Claude Code / Kiro)

# 5. Serve Canon (and any Reference) to your agent over MCP
memphis serve . --mcp
```

Then point your MCP client at the **store root**:

```json
{
  "mcpServers": {
    "my-store": {
      "command": "memphis",
      "args": ["serve", "/abs/path/to/my-store", "--mcp"]
    }
  }
}
```

The generated `.okf/config.yaml` is self-documenting:

```yaml
# Repository key: prefix for minted Canon artifact IDs (e.g. OKF-3F8A...).
repository_key: OKF

# Canon roots: directories that hold the authoritative tier. Everything else
# under the store is treated as Reference. Files here are validated by `memphis gate`.
canon_roots:
  - canon

# Spec roots: directories scanned for spec documents (requirements.md,
# design.md) that `memphis project` turns into typed Canon. Covers the local
# specs/ layout and Kiro's .kiro/specs/ layout by default.
spec_roots:
  - specs
  - .kiro/specs

# Ticketing provider: format-lints external "## Related Tickets" links.
# One of: github, jira, linear, azure-devops, servicenow, none.
ticketing:
  provider: github

# Enforcement: reclassify gate findings by rule code. Empty = each rule keeps
# its default severity. Uncomment and list rule codes to override.
enforcement: {}
```

---

## Spec-driven development with memphis

memphis is the authoritative memory beneath your spec-driven workflow. The specs your agent already writes become Canon, the gate keeps that Canon honest, and MCP feeds it back to the agent on every task. The same flow works whether you drive **Claude Code** (`/spec → /dev → /code-review` skills) or **Kiro** (specs + agent hooks). Both emit the same `requirements.md` / `design.md` contract, so one projector serves both.

### The lifecycle, end to end

```bash
# 0. Once per repo
memphis init . && memphis hooks install        # auto-gate on write, commit, and merge

# 1. Requirements: your agent's /spec (or Kiro) writes specs/<feature>/requirements.md
#    Project it into typed Canon (mints a stable ID, fills sections, infers relationships):
memphis project specs/<feature>/requirements.md
#    Or project the whole spec directory at once (skips tasks.md):
memphis project specs/<feature>/

# 2. Design: the same workflow produces design.md; project it to a design artifact:
memphis project specs/<feature>/design.md

# 3. Development: the agent grounds on Canon over MCP before writing code.
#    find_decisions / get_artifact / get_context return the authoritative requirements
#    and decisions the task must honor (Canon-first, with citations and live status).

# 4. Code review: the gate is the required check, and relationships cite what changed:
memphis gate . --sarif > memphis.sarif
memphis relationships . --summary
```

`memphis project` is **ratify-or-correct**: it never rewords your prose or silently overwrites. A new artifact is created, an existing one is only changed with `--write` (or interactive confirmation), and `--dry-run` previews the diff. Re-projecting reuses the artifact's ID, so identity is stable across iterations. Relationships are inferred only from **literal** `OKF-…` and alias references in the prose, for high precision and never fuzzy matching.

### Bootstrap from existing docs

If a decision already lives in ingested Reference, graduate it into Canon instead of retyping it:

```bash
memphis promote <concept-id-or-path> --type decision
```

### Integrating into your agent's skills

memphis is designed to disappear into your workflow. Each phase of spec-driven development emits authoritative memory as a natural byproduct of the work the agent is already doing: requirements become Canon the moment they're approved, decisions are captured as they're made, and the gate enforces all of it continuously. Drop these commands into the skill definitions you already use, and the loop runs itself. Every spec strengthens the memory, every task is grounded in it, and every review is checked against it.

**Claude Code** (`~/.claude/skills/`):

```bash
# /spec: at each approval gate, project the just-approved doc into Canon and enforce it
memphis project "specs/${FEATURE}/requirements.md"
memphis gate .            # block approval on a failing gate

# /dev: ground the implementation in authority before writing code (MCP, already running):
#   find_decisions("<area>"), get_artifact("OKF-..."), get_context("<task>")
memphis rebuild .         # refresh derived indexes after status changes

# /code-review: make the gate a required check and cite touched authority
memphis gate . --sarif > memphis.sarif
memphis relationships . --summary
```

Install the on-write hook so the gate runs inside the agent loop:

```bash
memphis hooks install --claude     # PostToolUse hook → memphis gate after Write/Edit
```

**Kiro** (specs in `.kiro/specs/`, hooks in `.kiro/hooks/` for the IDE and `.kiro/agents/*.json` for the CLI):

```bash
memphis project .kiro/specs/${FEATURE}/    # same projector, Kiro layout
memphis hooks install --kiro               # writes the Kiro IDE + CLI gate hooks
```

**Any MCP client** (no skills required): run `memphis serve . --mcp`, point the client at the store root, and the agent gets the authority tools (`find_decisions`, `get_artifact`, `get_context`, and the rest) directly.

The result is a compounding system. The more your team specs, decides, and ships, the richer and more authoritative the agent's memory becomes, while the gate guarantees it never drifts from what the team actually agreed to.

---

## Commands

In rough order of use. Store-scoped commands default to the current directory (`.`).

### Store setup

| Command | Purpose |
|---|---|
| `memphis init [path]` | Scaffold a store: write `.okf/config.yaml` and create canon roots. Flags: `--repository-key`, `--canon-root` (repeatable), `--ticketing`, `--force`, `--quiet`. |

### Authoring Canon

| Command | Purpose |
|---|---|
| `memphis new <type> <path>` | Scaffold a typed artifact with a minted ID and the type's sections. Flags: `--store`, `--title`. |
| `memphis project <spec-doc-or-dir>` | Project an approved `requirements.md`/`design.md` (local `specs/` or Kiro `.kiro/specs/`) into typed Canon: reuse or mint a stable ID, fill sections from the prose, infer literal relationships, validate. Flags: `--store`, `--type`, `--dry-run`, `--write`/`--force`, `--kiro-agent`, `--json`, `--quiet`. |
| `memphis promote <concept-id-or-path>` | Graduate an ingested Reference concept into a typed Canon draft. Flags: `--store`, `--type`, `--out`. |

### Authority

| Command | Purpose |
|---|---|
| `memphis gate [store]` | Run the unified authority gate (validate + relationships + policy). Exits non-zero on any blocking finding. Flags: `--json`, `--sarif`. |
| `memphis relationships [store]` | Report and validate the typed relationship graph. Flags: `--validate`, `--summary`, `--json`. |

### Automation (event hooks)

| Command | Purpose |
|---|---|
| `memphis hooks install` | Install hooks that run the gate automatically. git is always installed (`pre-commit` runs the blocking gate, `post-merge` runs the integrity guard), and agent targets are auto-detected. Target flags: `--git`, `--claude`, `--kiro-ide`, `--kiro-cli`, `--kiro`, `--all`, plus `--kiro-agent`, `--store`. |
| `memphis hooks uninstall` | Remove only memphis-managed hook content, leaving other hooks intact. |
| `memphis hooks status` | Show which memphis hooks are installed per target. |

Surfaces written: git (`.git/hooks/`), Claude Code (`.claude/settings.json` PostToolUse), Kiro IDE (`.kiro/hooks/memphis-gate.json`), and Kiro CLI (`.kiro/agents/<agent>.json` under `hooks.postToolUse`). Every install is marker-delimited and idempotent.

### Operating the store

| Command | Purpose |
|---|---|
| `memphis rebuild [store]` | Regenerate derived indexes (full-text search + relationship graph) from the Markdown source of truth. |
| `memphis serve <store>` | Serve the store over MCP. Flags: `--mcp` (default), `--name`, `--max-result-chars`. |
| `memphis export [store]` | Export Reference knowledge for scale-out (documents/graph). |
| `memphis demo` | Run an offline demo with a bundled example. |

### Optional: Reference ingestion (secondary)

For teams that also want a large docs corpus searchable as agent memory. These populate the Reference tier and never touch Canon.

| Command | Purpose |
|---|---|
| `memphis crawl <url>` | Crawl a documentation website into an OKF bundle. |
| `memphis import <path>` | Import local Markdown into an OKF bundle. |
| `memphis update <bundle>` | Update an existing bundle from its source. |
| `memphis validate <bundle>` | Validate an OKF bundle. |
| `memphis inspect <bundle>` | Inspect a bundle and show statistics. |

---

## MCP tools

`memphis serve <store> --mcp` exposes the store to any MCP client. Tools are grouped by job.

### Authority (Canon)

| Tool | Returns |
|---|---|
| `find_decisions` | Canon artifacts matching a query, authority-first, with citations and lifecycle status. |
| `get_artifact` | A specific artifact by ID (resolving `## Supersedes` to the current successor). |
| `get_context` | A budgeted, Canon-first context pack for a task, with normative requirement text preserved verbatim. |
| `get_related` | Typed relationships of an artifact (related requirements, decisions, and so on). |
| `get_neighbors` | The relationship neighborhood of an artifact within N hops. |

### Recall (Reference)

| Tool | Returns |
|---|---|
| `search_concepts` | Full-text search across the Reference tier. |
| `read_concept` | A single concept's full content. |
| `get_summary` | A concept's summary callout. |
| `list_types` / `list_tags` | The vocabulary present in the store. |

### Live updates and utility

`check_updates`, `apply_updates` (use `dry_run: true` to preview), `bundle_health`, `bundle_summary`, `compression_stats`.

---

## Configuration: `.okf/config.yaml`

| Key | Meaning |
|---|---|
| `repository_key` | Prefix for minted Canon IDs (for example `OKF`). |
| `canon_roots` | Directories that hold the authority tier; everything else is Reference. |
| `spec_roots` | Directories `memphis project` scans for spec docs. Default: `["specs", ".kiro/specs"]`. |
| `ticketing.provider` | Format-lints `## Related Tickets` links. One of `github`, `jira`, `linear`, `azure-devops`, `servicenow`, `none`. |
| `enforcement` | Reclassify gate findings by rule code into `blocking` / `advisory` / `disabled`. Empty means each rule keeps its default severity. |

`config.yaml` is the only thing that separates the tiers, and the rendered output round-trips through load, so you can edit it by hand or regenerate it with `memphis init --force`.

---

## Appendix: Reference tier and OKF format

The Reference tier is optional supporting material (abundant, summarized, searchable) rendered as an **Open Knowledge Format** (OKF) bundle: human- and agent-readable Markdown with YAML frontmatter, exchangeable without a central registry ([what is OKF](https://openknowledgeformat.com/what-is-okf)). It is the right tool when you want a large docs corpus usable as agent memory without standing up a vector store.

### Retrofit an existing repository (Reference-only)

```bash
# Import a repo's Markdown into a self-contained bundle and serve it directly
memphis import ~/repo/my-project --out ~/repo/my-project/.okf --source-name "My Project"
memphis serve ~/repo/my-project/.okf --mcp
```

With no `canon_roots` populated, the bundle stays pure Reference and behaves like a standalone searchable knowledge base. `memphis promote` is the bridge when a Reference concept matures into a decision worth enforcing as Canon.

### Concept format

Each Reference concept is a Markdown file with frontmatter:

```yaml
---
type: "Guide"
title: "Getting Started"
description: "How to get started"
resource: "https://example.com/docs/getting-started"
tags: ["setup", "onboarding"]
timestamp: "2024-01-01T00:00:00.000Z"
---
```

`type` and `title` are required; `description`, `resource`, `tags`, and `timestamp` are optional. An `index.md` provides summary-first navigation and backlinks across the bundle.

### Summarization

Reference summaries can be generated by fast **extractive** algorithms (offline, deterministic) or, optionally, an **LLM** mode via an external OpenAI-compatible endpoint or a local Ollama fallback. Summarization applies only to Reference and never participates in the Canon authority path.

### Scale ceiling

Summary-first navigation works well up to roughly **100 concepts / ~400K tokens**. Past that, graduate the fuzzy half to an external RAG system via `memphis export`, while Canon always stays canonical in the repo.
