# Upstream merge guide

Maintain this guide with every change to a fork integration point, replacement
codepath, or merge requirement. Link to [Roadmap.md](Roadmap.md) for behavior and
[Tracker.md](Tracker.md) for the current item's remaining work. Keep this guide
about the code that exists, not speculative implementation tasks.

## Baseline and current divergence

- Upstream: `https://github.com/jesseduffield/lazygit.git`, currently remote
  `origin`. No separate hosted fork remote has been configured by this work.
- Last inspected upstream base:
  `4e1eebe278790f257a875da6633234391a5a85b3`.
- Fork changes currently consist of planning/research under `new-features/`
  and fork-specific root `AGENTS.md` instructions. No application codepath or
  default has changed yet.
- Treat root `AGENTS.md` as fork-owned. Review upstream guidance changes for
  relevant technical constraints and incorporate them deliberately while
  retaining this fork's concise workflow and usability principles.

## Live integration record

There are no application integration points or replacement paths yet. As they
are implemented, record each one's source files/symbols, the upstream behavior
it extends or replaces, the reason for divergence, the integration comment,
and the verification that protects it. Include generated outputs and config
migration considerations where applicable; do not rely solely on line numbers.

## Reviewing an upstream merge

1. Record the upstream commit being integrated and review changes around all
   live integration points, including changes that merge without conflicts.
2. Preserve fork-owned additions and re-evaluate the narrow hooks into upstream.
   If upstream now solves the same problem, assess consolidation against the
   roadmap's agreed behavior instead of retaining redundant code.
3. Check replacement comments and this guide against the resulting code. Update
   obsolete paths, behavior descriptions, generated-file requirements and
   regression scenarios in the merge/resolution commit.
4. Regenerate affected outputs and run relevant regression checks plus the
   repository's build, unit-test, formatting and lint recipes. Validate user
   control and modal flows as well as Git results. Record the integrated base,
   verification results and any unresolved issue here before calling the merge
   complete.
