# Review 02: palette-usage

## Summary

Read-only review of the requested area against BASE `4e1eebe278790f257a875da6633234391a5a85b3`, at HEAD `a9d765542`.
Two clear interprocess correctness bugs can lose persisted usage counts; popup attribution also has a plausible cross-worker ownership bug.
Other-pane palette entries are disabled before invocation, their dynamic callbacks are stripped, and ordering is deterministic for a fixed binding declaration order.
The request's usage-order feature description is historical: commit `47d30f89f` intentionally replaced it with pane sections and key-group ordering and removed `gui.commandPaletteOrder`. Current Guide.md, Design.md and MergeGuide.md agree with that change.
Most implementation lives in additive files. The principal avoidable upstream replacement surface is the body of `OptionsMenuAction.Call`; popup hooks themselves are small and commented.
No repository files were changed. No implementation, commits, builds, or test runs were performed, as required by the worker request's read-only and no-tests-by-default constraints.

## Findings

### F1. Startup corruption recovery can delete another instance's repaired counts

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/commandusage/store.go:47–54; pkg/commandusage/store.go:173–202
- Problem: `Open` reads and removes a corrupt file without acquiring the writer lock or rechecking its contents under that lock. Atomic rename does not protect this separate deletion.
- Scenario / evidence: A reads corrupt JSON in `Open`, then is descheduled. B calls `Flush`, obtains the lock, treats the corrupt file as empty, merges its pending counts, atomically renames a valid replacement and releases the lock. A resumes at `os.Remove(path)` and deletes B's valid replacement. B has cleared its pending deltas and cannot restore them on an otherwise idle exit. Two concurrent startup recoveries can produce the same race when one instance starts recording before the other finishes removal. This follows directly from the unlocked read/remove versus locked update paths; no timing probe was run. `TestCorruptFileIsRemovedOnOpen` tests only one opener; the two-store test flushes sequentially.
- Suggested fix: On suspected corruption, acquire the same interprocess lock, reread the current file, and remove/reset only if it is still corrupt. If another instance already repaired it, load the repaired counts. Keep missing-file startup non-destructive.

### F2. Stale-lock removal and unconditional unlock can break mutual exclusion

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/commandusage/lock.go:13; pkg/commandusage/lock.go:25–36; pkg/commandusage/lock.go:48–53
- Problem: Ten seconds of lock-file age is treated as proof of an abandoned writer. Neither stale removal nor unlock identifies the owner of the file it removes.
- Scenario / evidence: A acquires the lock and is suspended for over ten seconds, or blocks in filesystem I/O/Sync. B removes A's live lock, acquires a replacement and starts updating the same counts. A resumes with its previously read snapshot and can overwrite B's changes; A's deferred unlock can also remove B's replacement lock and admit C. There is a second race even with a genuinely abandoned lock: B stats the old lock, pauses, C removes it and acquires a new lock, then B's `os.Remove(lockPath)` removes C's fresh lock based on stale stat information. `TestStaleLockIsBroken` covers an aged orphan only; `TestConcurrentAddIsRaceFree` exercises in-memory additions on one store, not concurrent file writers.
- Suggested fix: Use process-owned locking that survives scheduling delays and releases on process exit, or a recovery protocol that safely proves abandonment and cannot unlink a replacement lock. A timeout increase or another unlocked age check does not establish mutual exclusion.

### F3. A worker popup can claim a different command's pending use

- Category: bug
- Severity: medium
- Confidence: SUSPECTED
- Location: pkg/gui/fork_command_usage.go:94–104; pkg/gui/popup/fork_usage_hook.go:15–37; pkg/gui/popup/popup_handler.go:60–61; pkg/gui/gui.go:861–880
- Problem: The hook claims the innermost entry in a process-wide running-handler stack without identifying the popup's producing command. Hook evaluation occurs on the producer's goroutine, before the injected popup/menu creation function queues UI work.
- Scenario / evidence: UI command A is running with a pending use. A worker left by earlier command B calls `PopupHandler.Menu`, `Confirm` or `Prompt` during A's handler. B's popup claims A. When A returns, `end` sees the use as attached and does not record it. Cancelling B's popup then discards A's use even if A completed; accepting B's popup records A regardless of A's own eventual popup cancellation. The mutex prevents a data race, but establishes no causal ownership. The existing nested-use test covers synchronous nesting, not an unrelated worker. The call ordering is established by inspection; a live UI scheduling reproduction was not run.
- Suggested fix: Ensure worker-originated popups cannot consume an unrelated UI invocation's pending use. Preserve the already documented approximation that popups opened after their own handler returned count at invocation; any more precise asynchronous tracking should be separately agreed. Add a focused interleaving regression test when fixes are authorized.

### F4. The upstream palette entry point carries the whole replacement implementation

- Category: merge-ease
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/options_menu_action.go:17–88
- Problem: The upstream `Call` body is comprehensively replaced with fork-specific partitioning, tooltip generation, section construction, disabling and menu configuration. The helpers are additive, but most orchestration remains at an upstream integration point.
- Scenario / evidence: BASE-to-HEAD numstat is 33 added / 64 removed lines in this file. MergeGuide.md correctly calls it a replacement, and inline fork comments explain portions of it. A future upstream menu-action edit still conflicts with the replacement body.
- Suggested fix: Move the fork implementation to an additive fork-owned controller file and keep `Call` as a short, explicitly commented delegation explaining the upstream behavior it replaces. No UX redesign is needed.

## Merge-ease notes

Numbers below are whole-file BASE-to-HEAD numstat, not attribution of unrelated edits to this review area.

| Upstream file | Footprint and assessment |
| --- | --- |
| pkg/gui/controllers/options_menu_action.go | +33/-64. Broad replacement; see F4. |
| pkg/gui/popup/popup_handler.go | +7/-1. Usage adds one hook field and four small wrapper calls in Menu/Confirm/ConfirmIf/Prompt, all commented; policy lives in additive fork_usage_hook.go. The HandleTab forwarding change belongs to another feature. |
| pkg/gui/context/menu_context.go | +57/-13 across several features. In-scope rendering uses a cloned column slice and a short dim/strike branch; ExplainDisabled adds a small conditional. These are reasonable local seams. Prefix input reservation, BeforeAccept and other dispatch changes account for much of the remaining footprint and were not reviewed here. |
| pkg/gui/gui_common.go | +16/-0. Two small usage methods and the command-label bridge. Suitable adapter surface. |
| pkg/gui/types/common.go | +45/-6 across several features. Usage dispatch/reset methods and ExplainDisabled are small additions; identity/Binding metadata supports the palette. BeforeAccept, prompt completion and screen-mode fields are separate concerns. |
| pkg/gui/gui.go | +67/-3 across several features. Usage changes are fields, startup initialization, hook installation and final flush. These are small commented lifecycle seams. Worker popup marshalling precedes the actual UI creation, but follows hook evaluation; relevant to F3. |
| pkg/gui/services/custom_commands/client.go | +10/-3. Identity and verb metadata only, plus a placeholder comment. Local and small; MergeGuide.md records it. |
| pkg/gui/services/custom_commands/handler_creator.go | +2/-0. Comments distinguish parameter choices from independent actions. |
| pkg/gui/services/custom_commands/keybinding_creator.go | +3/-0. Shared stable identity and verb per command. Small seam. |

`pkg/commandusage/`, `pkg/config/custom_command_identity.go`, `pkg/config/fork_command_usage.go`, `pkg/gui/fork_command_usage.go`, palette helper/order/reset files and popup usage helpers are additive. Hash identities avoid literal command/prompt text in stored IDs; explicit IDs are namespaced and validated, and menu identities sort child identities. Corresponding tests cover these identity properties.

MergeGuide.md's Palette section matches the current pane ordering, callback stripping, disabled explanation, popup hooks and custom identity integration. It does not document either lock hazard. The full replaced options-menu implementation could have a clearer single replacement comment after extraction.

## Scope reconciliation and unresolved questions

- Should the review's acceptance baseline be the maintained pane/key-group design or the older usage-order description in the worker request? History explicitly records the former as intentional, so this review does not classify removed usage ordering, dim scope tags or `commandPaletteOrder` as regressions. HANDOFF.md:30–32 is dated session history and is stale relative to that commit. No design work or implementation was started.
- Should a handler returning an error without a popup still count? `end` receives no error, and `TestForkUsageCommitsAfterErrorToo` explicitly expects a count. Popup wrappers likewise commit before calling the accepted callback, so input validation errors and later failures count. Given HANDOFF.md's explicit preference for merge ease over precision, this is an unresolved acceptance question rather than an asserted bug.
- First-popup acceptance intentionally counts before later nested confirmation cancellation; it does not measure final operation success. A caller-supplied Cancel menu item with OnPress also counts; the generated Cancel row is added after wrapping and calls OnCancel, so it drops correctly. HANDOFF.md already records the former limitation. Preserve these distinctions when deciding which counting discrepancies need fixes.
- Custom menu leaf items carry identities, but are invoked directly rather than as new tracked palette/shortcut uses: choosing one settles the pending root menu-opener use. Should future counting be per menu opener or per leaf? Current behavior is consistent with the first-popup approximation; no change is proposed without a decision.

## Checks and coverage assessment

- Other-pane static copies clear DescriptionFunc, ShortDescriptionFunc, CommandIDFunc, VerbFunc, TooltipFunc, CommandActive and GetDisabledReason. Their Handler remains present, but options_menu_action assigns an inapplicable DisabledReason and MenuContext.OnMenuPress refuses it before popping/invoking. No normal UI path bypassing that guard was found. The existing sections integration scenario checks that choosing Reword from Branches produces `Disabled: Available in Commits`.
- Focused/global aliases merge by live ID with local precedence; copies own key/path slices. Other panes use static IDs and deduplicate within each pane; duplicate commands across different foreign panes are intentional and tested.
- Pane-map iteration is normalized by fixed pane rank then name. Binding ordering uses stable key-path sorting, preserving declaration order on ties and for unbound commands. Navigation is appended last.
- Disabled rendering clones LabelColumns before styling, avoiding accumulated ANSI state. Inapplicable labels dim; failed preconditions remain struck through. Existing rendering tests check both and repeated renders.
- Popup settlement uses sync.Once, so confirm/close races cannot double-record a single pending use. Ordinary first-popup cancellation and synchronous nested use are covered by source tests.
- Normal writes reread under the lock, merge deltas, create a temp file in the same directory, Sync and close it, then rename it over the target. In-memory pending/in-flight accounting preserves additions during Flush and restores failed deltas. The flaws are lock ownership/recovery and unlocked startup deletion, not a missing temp/rename path.
- Read all four current ui/fork_palette* scenarios. They exercise counts/reset, previous-session persistence, startup corruption removal and section visibility/refusal. They do not exercise interprocess writers, simultaneous corruption recovery, live stale-lock takeover or unrelated-worker popup ownership.
- No tests were run: the request permits targeted tests only to confirm a specific suspicion, and the available tests do not reproduce F1/F2/F3. No integration tests were run. No claim of runtime reproduction is made.

## Out of scope observations

No additional cross-area defects are asserted. Read a few dependency paths (keybinding invocation, binding cache, menu installation and popup UI injection) only to establish the in-scope control flow. Their broader features were not reviewed.

## Commands run and outcomes

- `cat /tmp/lazygitk-review-run-1793910387/02-worker-request.md` and `cat /tmp/lazygitk-review-run-1793910387/common-review-rules.md`: read request and all restrictions.
- `cat AGENTS.md`, `cat docs-master/fork/Tracker.md`, `cat docs-master/fork/Design.md`; `rg -n -i 'palette|usage|decisions' HANDOFF.md docs-master/fork/Guide.md docs-master/fork/MergeGuide.md`; focused `sed`/`cat` reads of relevant sections: established design baseline and historical mismatch.
- `git status --short` at start and again during review: same pre-existing untracked HANDOFF.md, reviews/, test/.viminfo; no tracked modifications reported.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- <palette/store/usage paths>`; `git show --stat 47d30f89f`; `git show 2944e7932 --format=fuller --stat`; `git show --format=fuller --no-patch dc4959836`; `git log --oneline -8`: traced intended ordering change and usage acceptance behavior.
- `git diff --stat`, `git diff --numstat`, and focused `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped files>`: reviewed implementation changes and measured footprint.
- `cat`/`sed`/`rg`/`nl -ba` reads of all requested implementation files, commandusage tests, palette partition/order tests, usage tracker/hook tests, identity tests, disabled-display test and all current ui/fork_palette* scenarios: evidence above. Read `justfile`; did not invoke mutating recipes or broad tests.
- A few exploratory reads used nonexistent guessed paths (pkg/gui/menu.go, popup.go, menus.go, popup_panel.go, popup_handler.go and pkg/config/custom_command.go); those commands returned missing-file errors. Located actual definitions with rg and read menu_panel.go, popup/popup_handler.go, controllers/helpers/confirmation_helper.go and user_config.go instead. No unresolved investigation blocker remains from those errors.
- `git diff --check 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <all scoped implementation paths>`: exit 0; no whitespace errors.
- Wrote this result with a heredoc to the authorized artifact path. No other files were written.

## Work performed, files changed, and next step

Performed code/history/docs/test-source review only. Repository files changed: none. Sole created artifact: `/tmp/lazygitk-review-run-1793910387/02-worker-result.md`.

All findings remain unfixed by design of this read-only request. Prioritize narrow fixes with deterministic regression tests for startup repair versus Flush and for stale-lock ownership. Then reproduce the worker-popup interleaving before deciding whether to fix F3 or explicitly accept it as a documented approximation. Handle F4 as a separate mechanical extraction. Resolve the historical acceptance-baseline mismatch and counting questions with the review owner; do not initiate a new design phase or restore the old palette behavior based on this request alone.
