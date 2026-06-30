# Memphis agent skills

These three skills implement the spec-driven development lifecycle that Memphis is built for, with the `memphis project` / `gate` / `hooks` / MCP grounding steps wired directly into each phase. They use the `SKILL.md` format shared by **Claude Code** ([docs](https://code.claude.com/docs/en/skills)) and **Kiro** ([docs](https://kiro.dev/docs/cli/skills/)), so the same folders install into both:

| Skill | Phase | Memphis integration |
|---|---|---|
| `spec` | Requirements -> Design -> Tasks | Projects each approved `requirements.md` / `design.md` into typed Canon and gates it. |
| `dev` | Implementation | Grounds the work in Canon over MCP (`find_decisions` / `get_artifact` / `get_context`) before writing code; rebuilds indexes after status changes. |
| `code-review` | Review | Runs `memphis gate --sarif` as a required authority check and cites touched artifacts via `memphis relationships --summary`. |

Every Memphis step is guarded by "if this is a Memphis store (a `.okf/config.yaml` exists)", so the skills also work unchanged in repositories that don't use Memphis.

## Install

The quickest way is the bundled installer at the repo root. It auto-detects each toolchain you have (Claude Code, Kiro, git) by its folder and, for the ones it finds, copies the skills into its skills dir (`~/.claude/skills` and/or `~/.kiro/skills`) and runs `memphis hooks install` for that target:

```bash
./install_skills.sh .        # pass the store dir; defaults to .
```

Or copy the skill folders into your personal skills directory by hand — `~/.claude/skills` for Claude Code, `~/.kiro/skills` for Kiro:

```bash
cp -R .claude/skills/spec .claude/skills/dev .claude/skills/code-review ~/.claude/skills/   # Claude Code
cp -R .claude/skills/spec .claude/skills/dev .claude/skills/code-review ~/.kiro/skills/     # Kiro
```

They become available as `/spec`, `/dev`, and `/code-review`. To use them only within a single repository instead, place them under that repo's `.claude/skills/` (Claude Code) or `.kiro/skills/` (Kiro); both load project-scoped skills automatically.

## The loop

```
/spec         requirements.md / design.md  ->  memphis project  ->  Canon (gated)
/dev          read Canon over MCP          ->  implement (TDD)   ->  memphis rebuild
/code-review  memphis gate --sarif         ->  findings cite the Canon they touch
```

Run `memphis hooks install` once in the repo so the gate also fires automatically on write, commit, and merge across git, Claude Code, and Kiro. See the project [README](../../README.md) for the full command set.
