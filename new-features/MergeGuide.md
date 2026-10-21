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
- Fork planning/research lives under `new-features/`; root `AGENTS.md` is
  fork-specific. Application divergences are recorded below as they land.
- Treat root `AGENTS.md` as fork-owned. Review upstream guidance changes for
  relevant technical constraints and incorporate them deliberately while
  retaining this fork's concise workflow and usability principles.

## Live integration record

### Explicit-action defaults: audited integration seams

- `pkg/config/user_config.go:GetDefaultConfigForPlatform` is the shared
  constructor for app loading, platform variants, dummy configs and generated
  docs/schema. Apply fork defaults here before `loadUserConfig` overlays YAML,
  not after loading. Loading/reloading starts with fresh defaults; no migration
  or rewrite of existing config is needed. Later config files win.
- `pkg/gui/background.go:startBackgroundRoutines` independently gates fetch,
  file polling and external-ref detection. Startup fetch only runs inside its
  enabled routine. `Gui.onSwitchToNewRepo` checks auto-fetch before retriggering.
  Preserve the independent read-only ref detector.
- `Gui.checkForChangedConfigsThatDontAutoReload` warns that fetch/file/ref
  polling settings and intervals need restart. Forwarding and conflict-staging
  settings are read live; do not introduce a separate fork reload policy.
- `Gui.runSubprocessWithSuspenseAndRefresh` explicitly refreshes on return.
  Disabling periodic file polling must not suppress this or action refreshes.
- `BranchesHelper.PostFetchRefresh` invokes `AutoForwardBranches` after both
  manual and background fetch. That method returns for `none`; no handler
  replacement is needed to prevent forwarding after a manual fetch.
- `RefreshHelper.refreshStateFiles` gates resolved-file staging on
  `AutoStageResolvedConflicts`. Its continue prompt depends on the previous
  conflict count collected within that gate, so disabling it also disables
  the automatic continue prompt, as upstream's option documents.
- Existing integration scenarios specifically testing automatic staging or
  forwarding must opt in explicitly. Default-policy scenarios should not set
  the option to its default: they must detect a regression in the constructor.

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
