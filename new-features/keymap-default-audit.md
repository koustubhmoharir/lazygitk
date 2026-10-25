# Item 8 default keymap audit

Bounded evidence for the main effective key/action matrix; this report does not allocate replacement keys. Platform values below are the checked-in `inventory.json` `platformDefaults` produced from `config.GetDefaultConfigForPlatform`. They include the fork overlay. Runtime config loading additionally merges deprecated alternate fields.

## Defaults and override semantics at the audit baseline

- The three serialized trees are `new-features/inventory/inventory.json` keys `platformDefaults.darwin`, `.linux`, `.windows` (approximately lines 12419–13089). Each platform has a `Universal`, `Status`, `Files`, `Branches`, `Commits`, `AmendAttribute`, `Stash`, `CommitFiles`, `Main`, `Submodules`, and `CommitMessage` group. Values can be checked directly there; the inventory README documents that they serialize `GetDefaultConfigForPlatform` and are defaults, not loaded user/repository config.
- `pkg/config/user_config.go`: `KeybindingConfig` and group schemas begin at `KeybindingConfig`; `GetDefaultConfigForPlatform` builds the upstream table and returns `applyForkDefaults(...)`. `pkg/config/fork_defaults.go:6` documents overlay timing; `applyForkDefaults` only changes the listed fields there: it sets direct diff defaults (`w/W`, `v/V`) and nils `DiffWithWorkingFile`, `NewWorktree`, `DiffingMenu`, `DiffingMenuAlt`, `ToggleRangeSelect`, `CommitChangesWithoutHook`, and `PasteCommits`. It does not clear `j/k/h/l` alternates or Ctrl defaults.
- `pkg/config/app_config.go`: `KeybindingPlatform` selects runtime platform at line 152; `loadUserConfigWithDefaults` and `loadUserConfig` begin at 160/164. The loader starts with defaults, decodes config files into that populated value, then calls `base.Keybinding.MergeLegacyAltKeybindings()` at line 227. Omitted fields retain defaults; explicitly supplied fields replace those slices. Deprecated alternate values are folded into primary fields by `pkg/config/user_config.go:858-873` (e.g. `PrevItemAlt` into `PrevItem`). Final `KeybindingConfig` carries no “user explicitly set this” provenance, so equality with a default cannot reveal whether it was configured. Preserve overrides via presence-aware config merge/change data, not post-merge value comparison.
- `pkg/config/user_config_validation.go`: `validateKeybindingsRecurse` / `validateKeybindings` validate key syntax; they do not establish collision freedom.

## Case-fold overlaps in the effective default fields

The following case-only overlaps are present in serialized defaults on **each** of darwin, linux and windows. They are field-value overlaps, not a claim that every pair is live simultaneously in every modal dispatcher. Deprecated fields are called out because the loader folds them into primary fields before normal use.

- Universal: `Quit=q` / `QuitWithoutChangingDirectory=Q`; `PrevItemAlt=k` (folded into `PrevItem`) / `ScrollUpMainAlt1=K` (folded into `ScrollUpMain`); `NextItemAlt=j` (folded into `NextItem`) / `ScrollDownMainAlt1=J` (folded into `ScrollDownMain`); `ScrollLeft=H` / `PrevBlockAlt=h` (folded into `PrevBlock`); `ScrollRight=L` / `NextBlockAlt=l` (folded into `NextBlock`); `NextMatch=n`, `PrevMatch=N`, `New=n`; `Push=P` / `Pull=p`; `Undo=z` / `Redo=Z`.
- `Status`: `AllBranchesLogGraph=a` / `AllBranchesLogGraphReverse=A`.
- `Files`: `CommitChanges=c` / `CommitChangesWithEditor=C`; `AmendLastCommit=A` / `ToggleStagedAll=a`; `StashAllChanges=s` / `ViewStashOptions=S`.
- `Branches`: `CreatePullRequest=o` / `ViewPullRequestOptions=O`; `ForceCheckoutBranch=F` / `FastForward=f` / `FetchRemote=f` / `AddForkRemote=F`; `RebaseBranch=r` / `RenameBranch=R`.
- `Commits`: `SquashDown=s` / `SquashAboveCommits=S`; `RenameCommit=r` / `RenameCommitWithEditor=R`; `MarkCommitAsFixup=f` / `CreateFixupCommit=F`; `SetFixupMessage=c` / `CherryPickCopy=C`; `AmendToCommit=A` / `ResetCommitAuthor=a`; `RevertCommit=t` / `CreateTag=T`; `MarkCommitAsBaseForRebase=B` / `ViewBisectOptions=b`.
- `Main`: `PrevFile=N` / `NextFile=n`.
- Fork direct-diff bindings each deliberately accept two case variants for one action: `DiffWithWorkingTree=w,W` and `DiffWithPreviousCommit=v,V`; these are aliases within one field, not two actions. Inventory records exact serialized values in each platform subtree. Runtime availability, pane precedence, and local/global fallback require the main dispatch audit.

## Ctrl defaults (counts from serialized platform fields)

Counting every serialized binding value whose spelling begins `<ctrl+`, including multi-key binding arrays and text-editing keys: **darwin 20, linux 25, windows 25**. The five-count platform difference is the Linux/Windows-only `MoveWordLeft`, `MoveWordRight`, `BackspaceWord`, `ForwardDeleteWord`, and primary `ConfirmInEditor` (`<ctrl+enter>`); on darwin `ConfirmInEditor` is `<meta+enter>`. All platforms have `ConfirmInEditorAlt=<ctrl+s>`.

Shared Ctrl fields on all platforms: `QuitAlt1=<ctrl+c>`, `SuspendApp=<ctrl+z>`, `JumpToFile=<ctrl+g>`, `ConfirmInEditorAlt=<ctrl+s>`, `ScrollUpMainAlt2=<ctrl+u>`, `ScrollDownMainAlt2=<ctrl+d>`, `CreatePatchOptionsMenu=<ctrl+p>`, `FilteringMenu=<ctrl+s>`, `CopyToClipboard=<ctrl+o>`, `OpenRecentRepos=<ctrl+r>`, `ToggleWhitespaceInDiffView=<ctrl+w>`, `OpenDiffTool=<ctrl+t>`, `Files.FindBaseCommitForFixup=<ctrl+f>`, `Files.OpenStatusFilter=<ctrl+b>`, `Branches.CopyPullRequestURL=<ctrl+y>`, `Commits.MoveDownCommit=<ctrl+j>`, `Commits.MoveUpCommit=<ctrl+k>`, `Commits.ResetCherryPick=<ctrl+r>`, `Commits.OpenLogMenu=<ctrl+l>`, `CommitMessage.CommitMenu=<ctrl+o>`. Linux/Windows additionally have the four word-edit fields and `ConfirmInEditor=<ctrl+enter>`. This is a field count, not a count of distinct key spellings or active command dispatches. Schema fields are in `pkg/config/user_config.go` under `KeybindingUniversalConfig`; text entry/editor controls should be interpreted in their editor contexts.

## Enter, Tab, and source limits

- `SubmitEditorText` is a real `KeybindingUniversalConfig` field (`pkg/config/user_config.go`, `yaml:"submitEditorText"`) and appears in all three inventory platform trees with `<enter>`. Other serialized Enter fields include `GoInto`, `Confirm`, `ConfirmMenu`, `ConfirmSuggestion`, and `RecentRepos`; values alone do not prove simultaneous runtime registrations.
- `<tab>` appears as `TogglePanel` and `NextBlockAlt2`; `<backtab>` appears as `PrevBlockAlt2`. The latter alternate fields are merged into primary block navigation in `MergeLegacyAltKeybindings`. Actual terminal byte normalization and context-specific registrations are outside this serialized-default count.
- Inventory scope/limitations are in `new-features/inventory/README.md`: factory entries are Go AST literals, do not evaluate helpers/loops/dynamic factories, and raw `SetKeybinding` calls and `hardcodedDispatchCases` are leads rather than effective registrations. Relevant sources for the main runtime pass are `pkg/gui/keybindings.go`, inventory-referenced `pkg/gui`/`pkg/gocui` factories, and `pkg/config/keynames.go` / terminal event handling in `pkg/gocui`.
- Config keys come from `pkg/config/user_config.go`; semantic verb/action metadata comes from GUI factory `Verb`/`VerbFunc` and command identity metadata in `pkg/gui`. Inventory records raw expressions and source locations, not evaluated dynamic verbs, targets, or runtime identity. It cannot establish that same-letter fields share or differ in semantic verb.

Companion evidence: [effective factory snapshots](keymap/README.md),
[dispatch trace](key-dispatch-audit.md), and [mode matrix](keymap-mode-matrix.md)
record expansion, precedence, dynamic verb/target roles and terminal byte aliases.
This field audit makes no allocation recommendation and does not verify the new
runtime keymap; command/path allocation and acceptance remain Tracker tasks.

The subsequent override foundation adds private load metadata to `UserConfig`
and `KeybindingIsExplicit`, without changing effective key values. It records
recognized YAML fields after migration/validation and before alias folding,
including explicit values equal to defaults and disabled bindings. That closes
the baseline provenance gap above. Command-to-source-field projection was
implemented later; the enabled profile is described in
[Fork_Keybindings.md](../docs-master/keybindings/Fork_Keybindings.md). See the [merge guide](MergeGuide.md#command-key-foundation-item-8-in-progress)
for the loader integration and compatibility boundaries.
