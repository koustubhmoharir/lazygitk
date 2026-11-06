package i18n

// Fork-owned English strings.
//
// The fork's additions to TranslationSet are kept here rather than in
// english.go, so that english.go and the translation files stay as close to
// upstream as possible. ForkTranslationSet is embedded in TranslationSet, and
// forkEnglishTranslationSet sets its values together with the fork's changes
// to upstream English values. EnglishTranslationSet calls it after building
// the upstream table.

// ForkTranslationSet holds the strings the fork adds to TranslationSet. Its
// fields are promoted, so callers use them as if they were on TranslationSet.
type ForkTranslationSet struct {
	FocusPanel                           string
	ForkToggleFileStage                  string
	ForkUnstageSelection                 string
	ForkRemovePatchSelection             string
	CheckoutDetached                     string
	CherryPickChangedNote                string
	CombineSelected                      string
	CombineSelectCommits                 string
	CombineChecking                      string
	CombineIgnoredTitle                  string
	CombineIgnoredPrompt                 string
	CombineApproveIgnored                string
	CombineReviewTitle                   string
	CombineReviewPrompt                  string
	CombineChanged                       string
	CombineFreshReview                   string
	CombineApplying                      string
	CombineRepositoryChanged             string
	CombineKeeperTitle                   string
	CombineDescriptionTitle              string
	CombineErrorTitle                    string
	CherryPickMainlineTitle              string
	CherryPickMainlinePrompt             string
	CherryPickMainlineChoice             string
	CherryPickMergeParentDetail          string
	CherryPickIgnoredPrompt              string
	CherryPickReviewPrompt               string
	CollapseTreeNode                     string
	ExpandTreeNode                       string
	BranchActionResetReviewPrompt        string
	BranchActionResetHardWarning         string
	BranchActionDetachedHead             string
	BranchActionUnbornHead               string
	BranchActionRebaseDefaultRange       string
	BranchActionRebaseReviewPrompt       string
	BranchActionRebasePausedError        string
	BranchActionMergeReviewPrompt        string
	BranchActionMergeConfigured          string
	BranchActionMergeAllowFF             string
	BranchActionMergeNoFF                string
	BranchActionMergeDefaultMessage      string
	BranchActionMergeOptions             string
	BranchActionMergeMessageTitle        string
	BranchActionMergeChosenMessage       string
	BranchActionMergeStagedTitle         string
	BranchActionMergeStagedPrompt        string
	BranchActionMergeApproveStaged       string
	BranchActionMergeIgnoredTitle        string
	BranchActionMergeIgnoredPrompt       string
	BranchActionMergeAllIgnoredPrompt    string
	BranchActionMergeApproveIgnored      string
	BranchActionMergeSquashConflicts     string
	BranchActionCreateTitle              string
	BranchActionCreateReview             string
	BranchActionCreateNoTracking         string
	BranchActionCreateConfiguredTracking string
	BranchActionCreateDirectTracking     string
	BranchActionCreateDirectUnavailable  string
	BranchActionTagCreateReview          string
	BranchActionDeleteSources            string
	BranchActionDeleteHead               string
	BranchActionDeleteReserved           string
	BranchActionDeleteNoRemote           string
	BranchActionDeleteChooseRemoteRef    string
	BranchActionDeleteNotBranch          string
	BranchActionDeleteManualTitle        string
	BranchActionDeleteManualAction       string
	BranchActionDeleteRemoteNameTitle    string
	BranchActionDeleteRemoteRefTitle     string
	BranchActionDeleteRemotePrompt       string
	BranchActionDeleteLocalUpstream      string
	BranchActionDeleteForcePrompt        string
	// Fork-owned integration strings are separate to avoid reformatting the
	// upstream table when longer field names are added.
	FilterCommitAncestors                 string
	FilterCommitDescendants               string
	CommitScopeSelectCommits              string
	ShowingCommitAncestorsOne             string
	ShowingCommitAncestorsMany            string
	ShowingCommitDescendantsOne           string
	ShowingCommitDescendantsMany          string
	FilterRef                             string
	ShowInCommits                         string
	CommitNotInCommitsLog                 string
	FilterByPath                          string
	FilterByAuthor                        string
	ClearLogFilter                        string
	ToggleCommitSelection                 string
	ClearCommitSelection                  string
	NoCommitsSelected                     string
	ClearCommitSelectionFirst             string
	CommitScopeNoRef                      string
	CommitScopeSelectBranch               string
	ShowingBranchWithUpstream             string
	ShowingBranchNoUpstream               string
	ShowingRemoteBranchHistory            string
	ShowingTagHistory                     string
	RefScopeLocalWithUpstream             string
	RefScopeTagLabel                      string
	BranchSyncCurrentPullSummary          string
	BranchSyncIntegrationMenu             string
	BranchSyncIntegrationSummary          string
	BranchSyncIntegrationDefault          string
	BranchSyncIntegrationFFOnly           string
	BranchSyncIntegrationMerge            string
	BranchSyncIntegrationNoFF             string
	BranchSyncIntegrationRebase           string
	BranchSyncIntegrationMerges           string
	BranchSyncIntegrationInteractive      string
	BranchSyncIntegrationConfiguredMerge  string
	BranchSyncIntegrationConfiguredRebase string
	BranchSyncIntegrationReviewTitle      string
	BranchSyncIntegrationReviewPrompt     string
	BranchSyncIntegrationMergeOptions     string
	BranchSyncIntegrationRebaseOptions    string
	BranchSyncIntegrationReplayBase       string
	BranchSyncIntegrationDefaultError     string
	BranchSyncIntegrationIgnoredTitle     string
	BranchSyncIntegrationIgnoredPrompt    string
	BranchSyncIntegrationApproveIgnored   string
	BranchSyncIntegrationStatus           string
	BranchSyncMergeSelectedIntoCurrent    string
	BranchSyncRebaseCurrentOntoSelected   string
	BranchSyncResetCurrentToSelected      string
	BranchRenameReviewPrompt              string
	BranchUpstreamRefPrompt               string
	BranchUpstreamMenuPrompt              string
	BranchUpstreamReviewTitle             string
	BranchUpstreamSetReviewPrompt         string
	BranchUpstreamUnsetReviewPrompt       string
	BranchUpstreamStatus                  string
	BranchSyncStackPushChoice             string
	BranchSyncStackPushTitle              string
	BranchSyncStackPushPrompt             string
	BranchSyncStackPushExecute            string
	BranchSyncStackReviewStatus           string
	BranchSyncStackPushStatus             string
	BranchSyncStackNoMembers              string
	BranchSyncStackPushFailed             string
	BranchSyncStackPreparePush            string
	BranchSyncStackMemberReview           string
	BranchSyncStackContinue               string
	BranchSyncStackUpdateTitle            string
	BranchSyncStackUpdateContext          string
	BranchSyncStackUpdatePrompt           string
	BranchSyncStackUpdateExecute          string
	BranchSyncStackUpdateStatus           string
	BranchSyncStackUpdateNoMembers        string
	BranchSyncStackUpdateStopped          string
	BranchSyncStackUpdateCurrentStopped   string
	BranchSyncStackUpdatePreflightFailed  string
	BranchSyncStackUpdateIgnoredApproved  string
	BranchSyncStackUpdateCurrentLast      string
	BranchSyncStackUpdateCurrentLabel     string
	BranchSyncStackUpdateNoopLabel        string
	BranchSyncStackUpdateChoice           string
	BranchSyncFastForwardSelected         string
	BranchSyncFastForwardTooltip          string
	BranchSyncFastForwardSkippedTitle     string
	BranchSyncFastForwardNoUpstream       string
	BranchSyncFastForwardMissingRef       string
	BranchSyncFastForwardDiverged         string
	BranchSyncFastForwardIgnoredPrompt    string
	OpenMergeToolSelected                 string
	StageConflictsRangeDisabledNoKey      string
	CheckoutAsBranch                      string
	CreateMoveBranchHere                  string
	CreateBranchAndCheckout               string
	RebaseOntoSelectedCommit              string
	SelectedRebaseTooltip                 string
	SelectedRebaseSelectCommit            string
	SelectedRebaseRepositoryChanged       string
	SelectedRebaseOrdinary                string
	SelectedRebasePreserveMerges          string
	SelectedRebaseConfirmTitle            string
	SelectedRebaseConfirmPrompt           string
	SelectedRebaseChangedTitle            string
	SelectedRebaseChangedNote             string
	SelectedRebaseChecking                string
	SelectedRebaseApplying                string
	SelectedRebaseIgnoredTitle            string
	SelectedRebaseIgnoredPrompt           string
	SelectedRebaseApproveIgnored          string
	NamedCheckoutNamePrompt               string
	NamedCheckoutTooltip                  string
	NamedCheckoutMoveTitle                string
	NamedCheckoutMovePrompt               string
	NamedCheckoutChangedTitle             string
	NamedCheckoutChangedNote              string
	NamedCheckoutCurrentStatePrompt       string
	NamedCheckoutCreatePrompt             string
	NamedCheckoutAttachPrompt             string
	NamedCheckoutWorkingStatus            string
	NamedCheckoutApplyingStatus           string
	NamedCheckoutSelectCommit             string
	NamedCheckoutRepositoryChanged        string
	PlaceBranchNamePrompt                 string
	PlaceBranchTooltip                    string
	PlaceBranchMoveTitle                  string
	PlaceBranchMovePrompt                 string
	PlaceBranchChangedTitle               string
	PlaceBranchChangedNote                string
	PlaceBranchCreateAgainPrompt          string
	PlaceBranchAlreadyHereTitle           string
	PlaceBranchAlreadyHerePrompt          string
	PlaceBranchWorkingStatus              string
	PlaceBranchApplyingStatus             string
	PlaceBranchSelectCommit               string
	PlaceBranchRepositoryChanged          string
	BranchSyncRepositoryChanged           string
	ExternalToolChooseDiffTitle           string
	ExternalToolChooseMergeTitle          string
	ExternalToolUseTitle                  string
	ExternalToolUseOnce                   string
	ExternalToolSaveDiff                  string
	ExternalToolSaveMerge                 string
	ExternalToolNotFoundDiff              string
	ExternalToolNotFoundMerge             string
	ExternalToolSearching                 string
	BranchSyncNoLongerCurrent             string
	BranchSyncInspecting                  string
	BranchSyncPushMenu                    string
	BranchSyncPushSummary                 string
	BranchSyncPushThisBranch              string
	BranchSyncPushDiverged                string
	BranchSyncChooseRemote                string
	BranchSyncChooseDestination           string
	BranchSyncSetUpstream                 string
	BranchSyncUpstreamReplacement         string
	BranchSyncUpstreamAmbiguousTitle      string
	BranchSyncUpstreamAmbiguousPrompt     string
	BranchSyncPushWithoutTracking         string
	BranchSyncCheckSubmodules             string
	BranchSyncForceLease                  string
	BranchSyncRemotePrompt                string
	BranchSyncNamedRemoteRequired         string
	BranchSyncDestinationPrompt           string
	BranchSyncReviewTitle                 string
	BranchSyncReviewPrompt                string
	BranchSyncPushStatus                  string
	BranchSyncLeasePrompt                 string
	BranchSyncLeaseConfirmTitle           string
	BranchSyncLeaseConfirmPrompt          string
	BranchSyncMultipleURLsTitle           string
	BranchSyncMultipleURLsPrompt          string
	BranchSyncApproveMultipleURLs         string
	BranchSyncForceConfirm                string
	BranchSyncPushSelected                string
	BranchSyncPushCurrent                 string
	BranchSyncPullSelected                string
	BranchSyncPullCurrent                 string
	BranchSyncSelectOneLocalBranch        string
	BranchSyncNoCurrentLocalBranch        string
	BranchSyncCaptureTooltip              string
	BranchSyncUpdateMenu                  string
	BranchSyncUpdateSummary               string
	BranchSyncFetchForReview              string
	BranchSyncReviewLocalUpdate           string
	BranchSyncChooseUpstreamBranch        string
	BranchSyncUpstreamBranchPrompt        string
	BranchSyncChooseTrackingRef           string
	BranchSyncTrackingRefPrompt           string
	BranchSyncFetchReviewTitle            string
	BranchSyncFetchReviewPrompt           string
	BranchSyncUpdateReviewTitle           string
	BranchSyncUpdateReviewPrompt          string
	BranchSyncAdvanceBranch               string
	BranchSyncAlreadyAhead                string
	BranchSyncOffBranchOnly               string
	BranchSyncUpdateDiverged              string
	BranchSyncFetchStatus                 string
	BranchSyncUpdateStatus                string
	ContinueOperation                     string
	ContinueOperationHint                 string
	ContinueUnmergedFiles                 string
	SkipCommittedPickDisabled             string
	RecallOlderCommitDraft                string
	RecallNewerCommitDraft                string
	CommitDraftPosition                   string
	RewordFailed                          string
	RewordFailedRebaseStopped             string
	RewordFailedRebaseStoppedNoKey        string
	RewordFailedMessageKept               string
	PatchSourceNotInCommits               string
	FindingCommitStatus                   string
	LoadingCommitsStatus                  string
	CommandPaletteFocusedSectionTitle     string
	CommandPaletteAvailableIn             string
	DiffWithWorkingTree                   string
	DiffWithPreviousCommit                string
	DiffWithSelectedCommit                string
	HistoricalDiffCommitPrompt            string
	DiffWithWorkingFile                   string
	HistoricalDiffMissingSelection        string
	HistoricalDiffRootCommit              string
	HistoricalDiffSingleCommitOnly        string
	HistoricalDiffFileOnly                string
	HistoricalDiffWorkingFilePrompt       string
	HistoricalDiffMissingWorkingFile      string
	HistoricalDiffDeletedFile             string
	HistoricalDiffStickyTool              string
	DiffWorkingFileWithHead               string
	DiffWorkingFileWithStaged             string
	DiffWorkingFileWithCommit             string
	WorkingFileCommitPrompt               string
	WorkingFileUntracked                  string
	WorkingFileNoUnstagedChanges          string
	WorkingFileUnbornHead                 string
	WorkingFileNoPathsMatchFilter         string
	CopyCommitHash                        string
	CopyAbbreviatedCommitHash             string
	CopyCommitSubject                     string
	CopyCommitMessage                     string
	CopyCommitMessageBody                 string
	CopyCommitURL                         string
	CopyCommitDiff                        string
	CopyCommitAuthor                      string
	CopyCommitTags                        string
	CopyFileNameToClipboard               string
	CopyAbsoluteFilePathToClipboard       string
	CopyFileDiffToClipboard               string
	CopyAllFilesDiffToClipboard           string
	CopyFileContentToClipboard            string
	ForkDiscardChangePrompt               string
	RepoSwitchLoading                     string
	RepoSwitchOpenedToast                 string
	CommandUsageReset                     string
	IgnoredKeyConfig                      string
	DiffRendererDeltaNotFound             string
	ResetCommandUsageCounts               string
	ResetCommandUsageCountsPrompt         string
	OpenRepoByPath                        string
	InspectFile                           string
	InspectFileTooltip                    string
	CheckoutPreviousBranchNamed           string
	NoPreviousBranch                      string
	RepoPathPrompt                        string
	ErrRepoPathNotDirectory               string
	ErrRepoPathNotGitRepo                 string
	NoConflictedFilesSelected             string
	ForkCannotSelfUpdate                  string
}

// forkIntroPopupMessage is the fork's first-run popup. It only says that this is
// a fork, and keeps upstream's note on thanking lazygit's author.
const forkIntroPopupMessage = `
lazygitk is a fork of lazygit: https://github.com/jesseduffield/lazygit

If lazygit has made your life easier, you can say thanks to its author by
clicking the donate button at the bottom right. Donation does not grant
priority support, but it is much appreciated.

Press {{confirmationKey}} to get started.
`

// forkEnglishTranslationSet applies the fork's English strings and its changes
// to upstream English values. It runs after EnglishTranslationSet has built the
// upstream table.
func forkEnglishTranslationSet(set *TranslationSet) {
	set.ForkTranslationSet = ForkTranslationSet{
		FocusPanel:                            "Focus %s panel",
		ForkToggleFileStage:                   "Toggle staging of selected files",
		ForkUnstageSelection:                  "Unstage selected changes",
		ForkRemovePatchSelection:              "Remove selected lines from custom patch",
		CheckoutDetached:                      "Checkout detached HEAD",
		CherryPickChangedNote:                 "The destination or ignored-file risks changed. Review the fresh state before proceeding.",
		CombineSelected:                       "Combine selected commits",
		CombineSelectCommits:                  "Select at least two full commit IDs to combine.",
		CombineChecking:                       "Inspecting selected commits",
		CombineIgnoredTitle:                   "Review ignored files before combining commits",
		CombineIgnoredPrompt:                  "Git may overwrite these ignored files while rewriting history. Cancel or explicitly approve this risk:\n%s",
		CombineApproveIgnored:                 "Proceed and accept the reviewed ignored-file risks",
		CombineReviewTitle:                    "Review commit combination",
		CombineReviewPrompt:                   "Current HEAD: %s at %s\n\nKeeper (earliest selected ancestor): %s\nSelected commits, ancestor first:\n%s\n\nGit will rewrite the keeper through HEAD. Unselected commits remain separate. Git may stop for conflicts or hooks. No automatic stash, retry or rollback will run.",
		CombineChanged:                        "The captured history or ignored-file risks changed. Review the fresh state before proceeding.",
		CombineFreshReview:                    "Capture and review current history again using the same selected commit IDs",
		CombineApplying:                       "Combining selected commits",
		CombineRepositoryChanged:              "The repository changed while reviewing the commit combination.",
		CombineKeeperTitle:                    "Combined commit summary",
		CombineDescriptionTitle:               "Combined commit message (all selected messages, ancestor first)",
		CombineErrorTitle:                     "Error",
		CherryPickMainlineTitle:               "Choose the merge parent to compare against",
		CherryPickMainlinePrompt:              "The selected commits include merges. Choose the parent number used as the baseline for each merge. Pick merges separately if they require different parent numbers.",
		CherryPickMainlineChoice:              "Use parent %d of each selected merge",
		CherryPickMergeParentDetail:           " (relative to parent %d: %s)",
		CherryPickIgnoredPrompt:               "Cherry-picking may overwrite these ignored paths. Approve this risk to proceed to the commit review.",
		CherryPickReviewPrompt:                "Current destination: %s at %s\n\nCherry-pick these commits in this order:\n%s\n\nGit may stop for conflicts. No automatic stash, retry or rollback will run.",
		CollapseTreeNode:                      "Collapse directory or select parent",
		ExpandTreeNode:                        "Expand directory or select first child",
		BranchActionResetReviewPrompt:         "Current HEAD: %q at %s\nSelected source: %q (%s) at %s\n\nReset affects current HEAD, with index/worktree effects determined by the chosen mode. It does not reset another selected branch.",
		BranchActionResetHardWarning:          "Are you sure you want to do a hard reset? This discards staged and unstaged tracked changes and may overwrite untracked or ignored paths that obstruct the selected commit. No automatic retry or rollback will run.",
		BranchActionDetachedHead:              "HEAD (detached)",
		BranchActionUnbornHead:                "(unborn)",
		BranchActionRebaseDefaultRange:        "No marked base (use the selected upstream)",
		BranchActionRebaseReviewPrompt:        "Rebase current branch %q at %s onto selected source %q (%s) at %s?\n\nMode: %s\nMarked range base: %s\n\nOnly the current branch is rebased. Commit tracked changes before proceeding.",
		BranchActionRebasePausedError:         "The reviewed rebase is paused without unresolved conflicts. Inspect the pending commits, then open Rebase options to choose continue, skip or abort.",
		BranchActionMergeReviewPrompt:         "Current branch: %q at %s\nSelected source: %q (%s) at %s\n\nMode: %s\n\nThe selected source is merged into the current branch. Its ref is not moved.",
		BranchActionMergeConfigured:           "Merge using configured options",
		BranchActionMergeAllowFF:              "Merge (allow fast-forward)",
		BranchActionMergeNoFF:                 "Merge (no fast-forward)",
		BranchActionMergeDefaultMessage:       "Merge branch %q into %q",
		BranchActionMergeOptions:              "\n\nConfigured merge arguments: %s",
		BranchActionMergeMessageTitle:         "Squash commit message",
		BranchActionMergeChosenMessage:        "Chosen commit message: %q",
		BranchActionMergeStagedTitle:          "Include already-staged changes?",
		BranchActionMergeStagedPrompt:         "The files below already have staged changes. Squash-and-commit would include those staged changes as well as the merge result. Cancel to manage them first, or explicitly approve their inclusion. Git may still refuse the merge if staged changes obstruct it.",
		BranchActionMergeApproveStaged:        "Include the reviewed staged changes",
		BranchActionMergeIgnoredTitle:         "Review ignored files before merging",
		BranchActionMergeIgnoredPrompt:        "Git may overwrite the ignored files listed below while applying this merge. The list can include files Git leaves unchanged. Cancel to move or back up files first; approval explicitly accepts this risk.",
		BranchActionMergeAllIgnoredPrompt:     "The configured merge strategy has no exact path preview here. All ignored files are listed as potential overwrite risks. Cancel to inspect or back up files first, or explicitly approve this conservative risk list. The configured strategy will be retained.",
		BranchActionMergeApproveIgnored:       "Proceed and accept the reviewed ignored-file risks",
		BranchActionMergeSquashConflicts:      "The squash left unresolved conflicts. Resolve them and explicitly commit the staged result. Git did not start a normal merge, so merge --abort is unavailable. No automatic commit ran.",
		BranchActionCreateTitle:               "Create branch and check it out",
		BranchActionCreateReview:              "Create branch %q and check it out.\nSource: %s (%s)\nCurrent HEAD: %s (%s)\nChoose upstream tracking:",
		BranchActionCreateNoTracking:          "Do not set up upstream tracking",
		BranchActionCreateConfiguredTracking:  "Use Git's configured tracking policy",
		BranchActionCreateDirectTracking:      "Track the selected source branch directly",
		BranchActionCreateDirectUnavailable:   "Direct tracking needs a local or remote branch source.",
		BranchActionTagCreateReview:           "Tag %q currently names object %s.\nSource: %s (%s).\nOverwrite this tag with the reviewed source commit?",
		BranchActionDeleteSources:             "Selected refs:\n%s",
		BranchActionDeleteHead:                "\n\nCurrent HEAD: %q at %s",
		BranchActionDeleteReserved:            "%s is reserved by %s in Git directory %q. Finish the operation or manage that worktree explicitly before deleting its branch.",
		BranchActionDeleteNoRemote:            "No configured remote is available. Configure a remote before deleting a server branch.",
		BranchActionDeleteChooseRemoteRef:     "Choose remote branch to delete",
		BranchActionDeleteNotBranch:           "This ref is not a remote branch under refs/heads/.",
		BranchActionDeleteManualTitle:         "Choose an exact remote destination",
		BranchActionDeleteManualAction:        "Choose remote and branch explicitly",
		BranchActionDeleteRemoteNameTitle:     "Configured remote name",
		BranchActionDeleteRemoteRefTitle:      "Full remote branch ref (refs/heads/...)",
		BranchActionDeleteRemotePrompt:        "\n\nDelete these server branches?\n%s\n\nCached commit IDs may differ from the server's current tips. This deletes the named server branches, including any newer commits they contain.",
		BranchActionDeleteLocalUpstream:       "This upstream is a local ref. Select it in the branches pane to delete it, or choose a server branch explicitly.",
		BranchActionDeleteForcePrompt:         "\n\nThese branches have history not preserved by the remaining reviewed HEAD, upstream or main refs:\n%s\n\nDelete these local branches anyway?",
		FilterCommitAncestors:                 "Show selected commits and their ancestors",
		FilterCommitDescendants:               "Show selected commits and their descendants",
		CommitScopeSelectCommits:              "Select only commits to filter by their ancestors or descendants.",
		ShowingCommitAncestorsOne:             "Showing ancestors of %d selected commit",
		ShowingCommitAncestorsMany:            "Showing ancestors of %d selected commits",
		ShowingCommitDescendantsOne:           "Showing descendants of %d selected commit",
		ShowingCommitDescendantsMany:          "Showing descendants of %d selected commits",
		FilterRef:                             "Show branch / tag",
		ShowInCommits:                         "Show in Commits",
		CommitNotInCommitsLog:                 "%s is not in the Commits log: no branch, tag or HEAD reaches it.",
		FilterByPath:                          "Filter by path",
		FilterByAuthor:                        "Filter by author",
		ClearLogFilter:                        "Clear the log filter",
		ToggleCommitSelection:                 "Toggle selection",
		ClearCommitSelection:                  "Clear selection",
		NoCommitsSelected:                     "No commits selected",
		ClearCommitSelectionFirst:             "Clear the selection first (Escape)",
		CommitScopeNoRef:                      "No branch or tag points at this commit.",
		CommitScopeSelectBranch:               "Select a local branch to show it with its upstream.",
		ShowingBranchWithUpstream:             "Showing %s and %s",
		ShowingBranchNoUpstream:               "Showing %s (no tracking branch)",
		ShowingRemoteBranchHistory:            "Showing %s and its history",
		ShowingTagHistory:                     "Showing tag %s and its history",
		RefScopeLocalWithUpstream:             "%s and %s",
		RefScopeTagLabel:                      "tag %s",
		BranchSyncCurrentPullSummary:          "Checked-out branch %q at %s\nUpstream remote: %q\nUpstream branch: %q\nLocal tracking storage: %q\n\nFetch first, then review the exact incoming commit and integration method before changing files. Saved upstream settings will not change.",
		BranchSyncIntegrationMenu:             "Integrate reviewed incoming commit",
		BranchSyncIntegrationSummary:          "Checked-out branch %q at %s\nReviewed incoming commit: %s\n\nChoose an integration method. The default uses Git configuration; explicit choices replace the corresponding defaults. No second network pull or autostash will run.",
		BranchSyncIntegrationDefault:          "Use Git's configured pull method",
		BranchSyncIntegrationFFOnly:           "Fast-forward only",
		BranchSyncIntegrationMerge:            "Merge (fast-forward if possible)",
		BranchSyncIntegrationNoFF:             "Merge (create a merge commit)",
		BranchSyncIntegrationRebase:           "Rebase (linear history)",
		BranchSyncIntegrationMerges:           "Rebase preserving merges",
		BranchSyncIntegrationInteractive:      "Interactive rebase",
		BranchSyncIntegrationConfiguredMerge:  "Configured merge",
		BranchSyncIntegrationConfiguredRebase: "Configured rebase",
		BranchSyncIntegrationReviewTitle:      "Review current-branch integration",
		BranchSyncIntegrationReviewPrompt:     "Integrate reviewed commit %s into checked-out branch %q at %s using %s?\n\nThis worktree's HEAD, index and files may change; conflicts may need resolution. Other local branch tips and nested submodule checkouts will not be updated automatically. No autostash or second network pull will run. Explicit merge choices replace branch.mergeOptions for this invocation only.",
		BranchSyncIntegrationMergeOptions:     "\n\nConfigured branch.mergeOptions: %q\npull.ff override: %q",
		BranchSyncIntegrationRebaseOptions:    "\n\nUsing configured rebase topology/autosquash options. Automatic updates of other branch refs remain disabled.",
		BranchSyncIntegrationReplayBase:       "\n\nReplay base: %s (the captured old upstream boundary). Only local commits after this boundary will be replayed.",
		BranchSyncIntegrationDefaultError:     "\n\nConfigured method unavailable: %s\nChoose an explicit method or cancel.",
		BranchSyncIntegrationIgnoredTitle:     "Ignored files may be overwritten",
		BranchSyncIntegrationIgnoredPrompt:    "\n\nGit may overwrite the listed ignored files even with a clean working tree. Move or back up files first, or explicitly approve this risk.",
		BranchSyncIntegrationApproveIgnored:   "Approve ignored-file overwrite risk and integrate",
		BranchSyncIntegrationStatus:           "Integrating reviewed commit into branch %q",
		BranchSyncMergeSelectedIntoCurrent:    "Merge selected branch into current…",
		BranchSyncRebaseCurrentOntoSelected:   "Rebase current onto selected branch…",
		BranchSyncResetCurrentToSelected:      "Reset checked-out branch to selected ref…",
		BranchRenameReviewPrompt:              "Rename local branch %q to %q at %s?\n\nOnly branch metadata changes. Checked-out commits, index contents and files stay in place; linked worktrees using this branch will use its new name.",
		BranchUpstreamRefPrompt:               "Full upstream tracking ref (e.g. refs/remotes/origin/main or refs/heads/main):",
		BranchUpstreamMenuPrompt:              "Selected branch %q at %s\nUpstream: %s",
		BranchUpstreamReviewTitle:             "Review selected branch upstream",
		BranchUpstreamSetReviewPrompt:         "Set upstream of local branch %q at %s to %q at %s?\n\nPrevious upstream remote: %s\nPrevious merge refs: %q\n\nOnly saved tracking metadata changes. No fetch, push, checkout, integration or tip change will run. Existing rebase preferences stay unchanged.",
		BranchUpstreamUnsetReviewPrompt:       "Unset upstream of local branch %q at %s?\n\nPrevious upstream remote: %s\nPrevious merge refs: %q\n\nOnly saved tracking metadata changes. Branch tips, index and files stay in place. Existing rebase preferences stay unchanged.",
		BranchUpstreamStatus:                  "Updating upstream metadata for branch %q",
		BranchSyncStackPushChoice:             "Review push for this branch and branches stacked below…",
		BranchSyncStackPushTitle:              "Review stack push",
		BranchSyncStackPushPrompt:             "Review every source and destination below. No branch has been pushed yet.\n\nExecute pushes these branches in the listed order. Git's normal concurrency checks apply; this is not an atomic transaction. On failure, completed pushes remain and later pushes stop. There is no automatic retry or rollback.",
		BranchSyncStackPushExecute:            "Push all reviewed branches",
		BranchSyncStackReviewStatus:           "Reviewing push for branch %q",
		BranchSyncStackPushStatus:             "Pushing reviewed branches",
		BranchSyncStackNoMembers:              "No other unmerged local branch tips were found below the selected branch. Nothing was pushed; use the one-branch push action instead.",
		BranchSyncStackPushFailed:             "Stack push stopped at branch %q. Completed branches: %q. Later branches were not attempted. Inspect remote state before retrying; Git may have partly pushed the failing branch.\n\n%w",
		BranchSyncStackPreparePush:            "Review this branch's push",
		BranchSyncStackMemberReview:           "Stack review: branch %d of %d. Approval here only collects a plan; no branch is pushed until the final stack approval.\n\n",
		BranchSyncStackContinue:               "Continue reviewing remaining branches",
		BranchSyncStackUpdateTitle:            "Review branch updates",
		BranchSyncStackUpdateContext:          "Update review: branch %d of %d. Approved fetches may update tracking refs. Local branch tips and worktree integration wait for final batch approval.\n\n",
		BranchSyncStackUpdatePrompt:           "Approved fetches may have changed tracking refs; local tips and files have not changed.\n\nApply ref-only fast-forwards first, then the checked-out integration, in the order below. Targets stay pinned: updating a selected local upstream does not retarget another plan.\n\nNot atomic: earlier advances remain after failure. No automatic retries, rollback, stash, divergent resets or updates of other worktrees.",
		BranchSyncStackUpdateExecute:          "Apply all reviewed branch updates",
		BranchSyncStackUpdateStatus:           "Applying reviewed branch updates",
		BranchSyncStackUpdateNoMembers:        "No other unmerged local branch tips were found below the selected branch. No fetch or update ran; use the one-branch update action instead.",
		BranchSyncStackUpdateStopped:          "Branch updates stopped at %q. Ref-only branches already advanced: %q. Later updates were not attempted. Inspect refs before retrying; a failing native ref transaction may have committed.\n\n%w",
		BranchSyncStackUpdateCurrentStopped:   "Checked-out branch integration for %q stopped. Ref-only branches already advanced: %q. Do not repeat the batch automatically; inspect the current state and review any remaining work explicitly.\n\n%w",
		BranchSyncStackUpdatePreflightFailed:  "Update preflight failed for %q. No local branch tip or worktree integration was changed by this batch; approved fetches may have updated tracking refs.\n\n%w",
		BranchSyncStackUpdateIgnoredApproved:  "\n\nIgnored-file overwrite risk was explicitly approved for %d paths. Changed contents invalidate that approval before execution.",
		BranchSyncStackUpdateCurrentLast:      " (checked-out integration, last)",
		BranchSyncStackUpdateCurrentLabel:     "%q: integrate %s into %s",
		BranchSyncStackUpdateNoopLabel:        "%q: keep %s (already contains %s)",
		BranchSyncStackUpdateChoice:           "Review updates for this branch and branches stacked below…",
		BranchSyncFastForwardSelected:         "Fast-forward selected branches",
		BranchSyncFastForwardTooltip:          "Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end.",
		BranchSyncFastForwardSkippedTitle:     "Not fast-forwarded",
		BranchSyncFastForwardNoUpstream:       "no upstream",
		BranchSyncFastForwardMissingRef:       "%s does not exist locally; fetch first",
		BranchSyncFastForwardDiverged:         "has diverged from %s",
		BranchSyncFastForwardIgnoredPrompt:    "Fast-forward checked-out branch %q from %s to %s.",
		OpenMergeToolSelected:                 "Open merge tool",
		StageConflictsRangeDisabledNoKey:      "Cannot stage a selection that includes files with merge conflicts; resolve them individually first.",
		CreateMoveBranchHere:                  "Create/move branch here…",
		CreateBranchAndCheckout:               "Create branch and check it out…",
		RebaseOntoSelectedCommit:              "Rebase current branch onto this commit…",
		SelectedRebaseTooltip:                 "Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base.",
		SelectedRebaseSelectCommit:            "Select one actual commit to rebase the current branch onto.",
		SelectedRebaseRepositoryChanged:       "The repository changed. Start the rebase action again.",
		SelectedRebaseOrdinary:                "Ordinary rebase (linear history)",
		SelectedRebasePreserveMerges:          "Rebase preserving merge topology",
		SelectedRebaseConfirmTitle:            "Rebase current branch?",
		SelectedRebaseConfirmPrompt:           "Rebase current branch %q at %s onto commit %s?\n\n%s\n\nThis may rewrite history. No changes will be stashed and no additional branch refs will be updated. The marked base is not used. Coordinate other Git writers while rebase runs.",
		SelectedRebaseChangedTitle:            "Rebase changed; review again",
		SelectedRebaseChangedNote:             "The source branch, tip or potentially affected ignored files changed. Review the fresh state; previous approvals no longer apply.",
		SelectedRebaseChecking:                "Checking rebase",
		SelectedRebaseApplying:                "Rebasing current branch",
		SelectedRebaseIgnoredTitle:            "Ignored files may be overwritten",
		SelectedRebaseIgnoredPrompt:           "Git may overwrite the ignored files listed below while checking out the target or replaying commits. This conservative list can include files Git leaves unchanged. Scroll to review all paths. Cancel to move or back up files first; approval explicitly accepts this risk.",
		SelectedRebaseApproveIgnored:          "Approve ignored-file overwrite risk and rebase",
		CheckoutAsBranch:                      "Checkout as branch",
		NamedCheckoutNamePrompt:               "Local branch name for checkout at %s",
		NamedCheckoutTooltip:                  "Check out the selected commit under a literal new or existing local branch name. Git carries dirty changes when safe; files are never forced and changes are never stashed. Coordinate with other Git writers while checkout runs.",
		NamedCheckoutMoveTitle:                "Move branch and check out?",
		NamedCheckoutMovePrompt:               "Move branch %q from %s to %s and check it out?\n\nGit preserves local changes when safe; no files will be forced or changes stashed. Existing upstream settings are preserved.",
		NamedCheckoutChangedTitle:             "Checkout changed; review again",
		NamedCheckoutChangedNote:              "The destination branch or current checkout changed while this dialog was open. Review the fresh state before proceeding.",
		NamedCheckoutCurrentStatePrompt:       "Current checkout: %s.",
		NamedCheckoutCreatePrompt:             "Create and check out branch %q at %s? No changes will be stashed and no files will be forced.",
		NamedCheckoutAttachPrompt:             "Check out existing branch %q at %s? Its tip will not change.",
		NamedCheckoutWorkingStatus:            "Checking checkout",
		NamedCheckoutApplyingStatus:           "Checking out named branch",
		NamedCheckoutSelectCommit:             "Select one actual commit to check out under a branch name.",
		NamedCheckoutRepositoryChanged:        "The repository changed. Start the named checkout again.",
		PlaceBranchNamePrompt:                 "Local branch name at %s (without checkout)",
		PlaceBranchTooltip:                    "Create or move a local branch to the selected commit without changing HEAD, the index or working files.",
		PlaceBranchMoveTitle:                  "Move existing branch?",
		PlaceBranchMovePrompt:                 "Move branch %q from %s to %s?\n\nThis changes only the branch tip; it does not check out the branch. Existing upstream settings are preserved.",
		PlaceBranchChangedTitle:               "Branch changed; review again",
		PlaceBranchChangedNote:                "The branch changed while this dialog was open. Review its current state before proceeding.",
		PlaceBranchCreateAgainPrompt:          "Branch %q no longer exists. Create it at %s without checkout?",
		PlaceBranchAlreadyHereTitle:           "Branch already at selected commit",
		PlaceBranchAlreadyHerePrompt:          "Branch %q already points to %s. Nothing changed.",
		PlaceBranchWorkingStatus:              "Checking branch",
		PlaceBranchApplyingStatus:             "Placing branch",
		PlaceBranchSelectCommit:               "Select exactly one commit to create or move a branch.",
		PlaceBranchRepositoryChanged:          "The repository changed. Start the branch placement again.",
		BranchSyncRepositoryChanged:           "The repository changed. Start this action again.",
		ExternalToolChooseDiffTitle:           "No diff tool is configured",
		ExternalToolChooseMergeTitle:          "No merge tool is configured",
		ExternalToolUseTitle:                  "Use %s",
		ExternalToolUseOnce:                   "Just this time",
		ExternalToolSaveDiff:                  "Save as Git's diff.tool in your global config, and use it",
		ExternalToolSaveMerge:                 "Save as Git's merge.tool in your global config, and use it",
		ExternalToolNotFoundDiff:              "No diff tool was found. Install one, or set diff.tool in Git's config.",
		ExternalToolNotFoundMerge:             "No merge tool was found. Install one, or set merge.tool in Git's config.",
		ExternalToolSearching:                 "Looking for tools Git can use",
		BranchSyncNoLongerCurrent:             "Branch %q is no longer the checked-out branch. Start the action again.",
		BranchSyncInspecting:                  "Inspecting branch target",
		BranchSyncPushMenu:                    "Push selected branch",
		BranchSyncPushSummary:                 "Source %q at %s\nRemote: %q\nDestination: %q\nSet upstream for selected branch: %t\nCheck submodules: %t\nConfigured push mappings beyond this explicit destination will not be used.",
		BranchSyncPushThisBranch:              "Push this branch",
		BranchSyncPushDiverged:                "%s has commits that %s does not have; use Force with reviewed lease…",
		BranchSyncChooseRemote:                "Choose remote…",
		BranchSyncChooseDestination:           "Choose destination branch…",
		BranchSyncSetUpstream:                 "Toggle set upstream for selected local branch",
		BranchSyncUpstreamReplacement:         "\n\nExisting upstream settings before this push: remote %s; merge refs %q.",
		BranchSyncUpstreamAmbiguousTitle:      "Upstream settings need attention",
		BranchSyncUpstreamAmbiguousPrompt:     "Source %q at %s, remote %q, destination %q.\n\nThis branch has multiple, inherited, or invalid upstream settings that Git cannot safely replace with this push. No push has run and tracking settings are unchanged. Cancel, or prepare a new push review without changing tracking.",
		BranchSyncPushWithoutTracking:         "Push without changing tracking",
		BranchSyncCheckSubmodules:             "Toggle check submodules vs no recursion",
		BranchSyncForceLease:                  "Force with reviewed lease…",
		BranchSyncRemotePrompt:                "Remote name",
		BranchSyncNamedRemoteRequired:         "The configured push target is a URL. Enter a configured remote name to continue.",
		BranchSyncDestinationPrompt:           "Destination branch name or refs/heads/...",
		BranchSyncReviewTitle:                 "Review branch push",
		BranchSyncReviewPrompt:                "Push source %q at %s to remote %q, destination %q?\n\nSet upstream for selected local branch: %t\nCheck submodules: %t\n\nOnly this branch ref will be pushed. Configuration-wide push mappings, extra heads, and tags will not be pushed.",
		BranchSyncPushStatus:                  "Pushing selected branch %q",
		BranchSyncLeasePrompt:                 "Full locally available lease reference",
		BranchSyncLeaseConfirmTitle:           "Confirm force with lease",
		BranchSyncLeaseConfirmPrompt:          "Force push source %q at %s to %q using lease %q at %s?\n\nCancel to leave the remote unchanged.",
		BranchSyncMultipleURLsTitle:           "Approve multiple push destinations",
		BranchSyncMultipleURLsPrompt:          "The reviewed push will contact every destination shown below. Git may partially succeed when some destinations fail. No extra heads or tags are included.",
		BranchSyncApproveMultipleURLs:         "Approve push to all listed destinations",
		BranchSyncForceConfirm:                "Push with reviewed lease",
		BranchSyncPushSelected:                "Push selected branch…",
		BranchSyncPushCurrent:                 "Push checked-out branch",
		BranchSyncPullSelected:                "Pull/update selected branch…",
		BranchSyncPullCurrent:                 "Pull checked-out branch",
		BranchSyncSelectOneLocalBranch:        "Select exactly one local branch.",
		BranchSyncNoCurrentLocalBranch:        "There is no checked-out local branch.",
		BranchSyncCaptureTooltip:              "Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation.",
		BranchSyncUpdateMenu:                  "Update selected branch without checkout",
		BranchSyncUpdateSummary:               "Local branch %q at %s\nUpstream remote: %q\nUpstream branch: %q\nLocal tracking storage: %q\n\nFetching does not move the local branch. A fast-forward needs a separate review. Saved upstream settings will not change.",
		BranchSyncFetchForReview:              "Fetch upstream for review",
		BranchSyncReviewLocalUpdate:           "Review local upstream update",
		BranchSyncChooseUpstreamBranch:        "Choose upstream branch…",
		BranchSyncUpstreamBranchPrompt:        "Upstream branch (full refs/heads/ ref or branch name)",
		BranchSyncChooseTrackingRef:           "Choose local tracking storage…",
		BranchSyncTrackingRefPrompt:           "Local tracking storage (full ref; never a local branch or tag)",
		BranchSyncFetchReviewTitle:            "Review upstream fetch",
		BranchSyncFetchReviewPrompt:           "Fetch remote %q, branch %q into local tracking ref %q for local branch %q at %s?\n\nOnly this tracking ref will be updated. No checkout, branch advancement, integration, pruning, tag or submodule update will run. Review the fetched commit separately before moving the local branch.",
		BranchSyncUpdateReviewTitle:           "Review branch fast-forward",
		BranchSyncUpdateReviewPrompt:          "Advance local branch %q from %s to reviewed commit %s?\n\nThis is a verified fast-forward. HEAD, index, files and unrelated local branch tips will not change. Moving the upstream later will not substitute a different commit.",
		BranchSyncAdvanceBranch:               "Fast-forward this branch",
		BranchSyncAlreadyAhead:                "Local branch %q at %s already contains the reviewed upstream commit %s. No branch update is needed; it will not be rewound.",
		BranchSyncOffBranchOnly:               "Branch %q is checked out here; use the current-branch fetch and integration review instead.",
		BranchSyncUpdateDiverged:              "The upstream was inspected but local branch %q has diverged. No local branch was moved. Integration needs an explicit checkout; this action will not reset, merge, rebase, stash or create a worktree.",
		BranchSyncFetchStatus:                 "Fetching upstream for selected branch %q",
		BranchSyncUpdateStatus:                "Fast-forwarding selected branch %q",
		ContinueOperation:                     "Continue the operation in progress",
		ContinueOperationHint:                 "Continue %s",
		ContinueUnmergedFiles:                 "Cannot continue yet. These files are still unmerged; resolve them and stage them first:\n\n- %s",
		SkipCommittedPickDisabled:             "This pick is already committed, so there is nothing to skip",
		RecallOlderCommitDraft:                "Recall older cancelled message",
		RecallNewerCommitDraft:                "Recall newer cancelled message",
		CommitDraftPosition:                   "Draft {{.index}} of {{.total}}",
		RewordFailed:                          "Rewording failed: {{.error}}",
		RewordFailedRebaseStopped:             "The rebase is still stopped. Press {{.key}} to abort or continue it before you try again.",
		RewordFailedRebaseStoppedNoKey:        "The rebase is still stopped. Abort or continue it before you try again.",
		RewordFailedMessageKept:               "Your message is saved. Press Up in the message box to get it back.",
		PatchSourceNotInCommits:               "The commit this patch was built from is not in the Commits list. Clear the filter, or reset the patch.",
		CommandPaletteFocusedSectionTitle:     "%s and global",
		CommandPaletteAvailableIn:             "Available in %s",
		FindingCommitStatus:                   "Finding commit",
		LoadingCommitsStatus:                  "Loading commits",
		DiffWithWorkingTree:                   "Diff with working tree",
		DiffWithPreviousCommit:                "Diff with previous commit",
		DiffWithSelectedCommit:                "Diff with commit…",
		HistoricalDiffCommitPrompt:            "Compare selected %s (left) with commit hash (right)",
		DiffWithWorkingFile:                   "Diff with working file…",
		HistoricalDiffMissingSelection:        "Select a commit to compare.",
		HistoricalDiffRootCommit:              "This commit has no available first parent to compare.",
		HistoricalDiffSingleCommitOnly:        "Select exactly one commit to compare.",
		HistoricalDiffFileOnly:                "Select a single historical file.",
		HistoricalDiffWorkingFilePrompt:       "Compare %s:%s with working file (inside worktree)",
		HistoricalDiffMissingWorkingFile:      "The working file is missing. Use Diff with working file… to choose another file.",
		HistoricalDiffDeletedFile:             "This file was deleted in the selected commit; there is no historical blob to compare.",
		HistoricalDiffStickyTool:              "Diff using current comparison mode",
		DiffWorkingFileWithHead:               "Diff with HEAD",
		DiffWorkingFileWithStaged:             "Diff with staged version",
		DiffWorkingFileWithCommit:             "Diff with commit…",
		WorkingFileCommitPrompt:               "Compare commit hash (left) with working file (right)",
		WorkingFileUntracked:                  "untracked file: Git has no base version to compare",
		WorkingFileNoUnstagedChanges:          "no unstaged changes",
		WorkingFileUnbornHead:                 "HEAD has no commit yet",
		WorkingFileNoPathsMatchFilter:         "no files in this directory match the current filter",
		CopyCommitHash:                        "Copy commit hash",
		CopyAbbreviatedCommitHash:             "Copy abbreviated commit hash",
		CopyCommitSubject:                     "Copy commit subject",
		CopyCommitMessage:                     "Copy commit message",
		CopyCommitMessageBody:                 "Copy commit message body",
		CopyCommitURL:                         "Copy commit URL",
		CopyCommitDiff:                        "Copy commit diff",
		CopyCommitAuthor:                      "Copy commit author",
		CopyCommitTags:                        "Copy commit tags",
		CopyFileNameToClipboard:               "Copy file name",
		CopyAbsoluteFilePathToClipboard:       "Copy absolute path",
		CopyFileDiffToClipboard:               "Copy file diff",
		CopyAllFilesDiffToClipboard:           "Copy all files diff",
		CopyFileContentToClipboard:            "Copy file content",
		ForkDiscardChangePrompt:               "Discard the selected unstaged changes from the working tree? This cannot be undone.",
		RepoSwitchLoading:                     "Loading…",
		RepoSwitchOpenedToast:                 "Opened {{.repo}}",
		CommandUsageReset:                     "Command usage counts were unreadable and have been reset.",
		IgnoredKeyConfig:                      "Key settings are fixed in lazygitk; ignored %s in %s",
		DiffRendererDeltaNotFound:             "delta was not found on PATH; showing git's own diff",
		ResetCommandUsageCounts:               "Reset command usage counts",
		ResetCommandUsageCountsPrompt:         "Reset all command usage counts?",
		OpenRepoByPath:                        "Open repository by path",
		InspectFile:                           "Inspect file",
		InspectFileTooltip:                    "Focus the selected file's diff. A conflicted file opens its resolution, and a submodule entry enters the submodule.",
		CheckoutPreviousBranchNamed:           "Switch to previous branch: %s",
		NoPreviousBranch:                      "No branch was checked out before the current one.",
		RepoPathPrompt:                        "Repository path",
		ErrRepoPathNotDirectory:               "Not a directory: {{.path}}",
		ErrRepoPathNotGitRepo:                 "Not a Git repository: {{.path}}",
		NoConflictedFilesSelected:             "No conflicted files selected",
		ForkCannotSelfUpdate:                  "This build of lazygitk cannot update itself. Rebuild it from source to update.",
	}
	set.MustExitFilterModePrompt = "Command not available while the commit log is filtered. Exit filter mode?"
	set.IntroPopupMessage = forkIntroPopupMessage
}
