_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Keybindings

## Global keybindings

| Key | Action | Info |
|-----|--------|-------------|
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | Scroll up main window |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | Scroll down main window |  |
| `` t l c `` | View command log options | View options for the command log e.g. show/hide the command log and focus the command log. |
| `` p `` | Push | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | Pull | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Increase diff context size | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Decrease diff context size | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | Cherry-pick |  |
| `` f `` | Fetch | Fetch changes from remote. |
| `` : `` | Execute shell command | Bring up a prompt where you can enter a shell command to execute. |
| `` g h o `` | View custom patch options |  |
| `` g o `` | View merge/rebase options | View options to abort/continue/skip the current merge/rebase. |
| `` <f5> `` | Refresh | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` =, + `` | Next screen mode (normal/half/fullscreen) |  |
| `` - `` | Prev screen mode |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Cancel |  |
| `` ? `` | Open keybindings menu |  |
| `` w o `` | View filter options | View options for filtering the commit log, so that only commits matching the filter are shown. |
| `` t d c `` | View diffing options | View options relating to diffing two refs e.g. diffing against selected ref, entering ref to diff against, and reversing the diff direction. |
| `` q `` | Quit |  |
| `` t a s `` | Suspend the application |  |
| `` t d w `` | Toggle whitespace | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | Edit config file | Open file in external editor. |
| `` u u `` | Undo | The reflog will be used to determine what git command to run to undo the last git command. This does not include changes to the working tree; only commits are taken into consideration. |
| `` u r `` | Redo | The reflog will be used to determine what git command to run to redo the last git command. This does not include changes to the working tree; only commits are taken into consideration. |

## List panel navigation

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus Status panel |  |
| `` 2 `` | Focus Files panel |  |
| `` 3 `` | Focus Branches panel |  |
| `` 4 `` | Focus Commits panel |  |
| `` 5 `` | Focus Stash panel |  |
| `` <pgup> `` | Previous page |  |
| `` <pgdown> `` | Next page |  |
| `` <home> `` | Scroll to top |  |
| `` <end> `` | Scroll to bottom |  |
| `` t h r `` | Toggle range select |  |
| `` <shift+down>, J `` | Range select down |  |
| `` <shift+up>, K `` | Range select up |  |
| `` / `` | Search the current view by text |  |
| `` , `` | Previous page |  |
| `` . `` | Next page |  |
| `` <, <home> `` | Scroll to top |  |
| `` >, <end> `` | Scroll to bottom |  |
| `` <alt+left> `` | Scroll left |  |
| `` <alt+right> `` | Scroll right |  |
| `` <tab> `` | Next tab |  |
| `` <backtab> `` | Previous tab |  |

## Commit files

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Copy path to clipboard |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | Checkout | Checkout file. This replaces the file in your working tree with the version from the selected commit. |
| `` x `` | Discard | Discard this commit's changes to this file. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes this file. |
| `` o `` | Open file | Open file in default application. |
| `` e `` | Edit | Open file in external editor. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Toggle file included in patch | Toggle whether the file is included in the custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Toggle all files | Add/remove all commit's files to custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | Toggle file tree view | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Expand all files | Expand all directories in the file tree |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` 0 `` | Focus main view |  |
| `` / `` | Filter the current view by text |  |

## Commit summary

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirm |  |
| `` <esc> `` | Close |  |

## Commits

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | View bisect options |  |
| `` <f2> `` | Reword | Reword the selected commit's message. |
| `` g e r `` | Reword with editor |  |
| `` <delete> `` | Drop | Drop the selected commit. This will remove the commit from the branch via a rebase. If the commit makes changes that later commits depend on, you may need to resolve merge conflicts. |
| `` e `` | Edit (start interactive rebase) | Edit the selected commit. Use this to start an interactive rebase from the selected commit. When already mid-rebase, this will mark the selected commit for editing, which means that upon continuing the rebase, the rebase will pause at the selected commit to allow you to make changes. |
| `` g i `` | Start interactive rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` g r p `` | Pick | Mark the selected commit to be picked (when mid-rebase). This means that the commit will be retained upon continuing the rebase. |
| `` > `` | Move commit down one |  |
| `` < `` | Move commit up one |  |
| `` c p `` | Cherry-pick |  |
| `` g r b `` | Mark as base commit for rebase | Select a base commit for the next rebase. When you rebase onto a branch, only commits above the base commit will be brought across. This uses the `git rebase --onto` command. |
| `` a `` | Amend | Amend commit with staged changes. If the selected commit is the HEAD commit, this will perform `git commit --amend`. Otherwise the commit will be amended via a rebase. |
| `` g e a `` | Amend commit attribute | Set/Reset commit author or set co-author. |
| `` g v c `` | Revert | Create a revert commit for the selected commit, which applies the selected commit's changes in reverse. |
| `` n t `` | Tag commit | Create a new tag pointing at the selected commit. You'll be prompted to enter a tag name and optional description. |
| `` t l o `` | View log options | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
| `` g p v `` | Open pull request in browser |  |
| `` w a `` | Show selected commits and their ancestors |  |
| `` w d `` | Show selected commits and their descendants |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Checkout | Checkout the selected commit as a detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | Select commits of current branch |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View files |  |
| `` / `` | Search the current view by text |  |

## Confirmation panel

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirm |  |
| `` <esc> `` | Close/Cancel |  |
| `` <ctrl+o> `` | Copy to clipboard |  |

## Files

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Copy path to clipboard |  |
| `` <space> `` | Toggle staging of selected files | Toggle staged for selected file. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` w s `` | Filter files by status |  |
| `` <enter> `` | Commit | Commit staged changes. |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` a `` | Amend last commit |  |
| `` g e c `` | Commit changes using git editor |  |
| `` e `` | Edit | Open file in external editor. |
| `` o `` | Open file | Open file in default application. |
| `` i `` | Ignore or exclude file |  |
| `` <f5> `` | Refresh | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` g s s `` | Stash | Stash all changes. For other variations of stashing, use the view stash options keybinding. |
| `` g s o `` | View stash options | View stash options (e.g. stash all, stash staged, stash unstaged). |
| `` x `` | Discard | View options for discarding changes to the selected file. |
| `` g z u `` | View upstream reset options |  |
| `` g z w `` | Reset | View reset options for working tree (e.g. nuking the working tree). |
| `` t t t `` | Toggle file tree view | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Expand all files | Expand all directories in the file tree |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y w `` | Copy all files diff |  |
| `` d w `` | Diff with HEAD |  |
| `` d s `` | Diff with staged version |  |
| `` d c `` | Diff with commit… |  |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` v `` | Inspect file | Focus the selected file's diff. A conflicted file opens its resolution, and a submodule entry enters the submodule. |
| `` 0 `` | Focus main view |  |
| `` / `` | Filter the current view by text |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirm |  |
| `` <esc> `` | Close/Cancel |  |

## Local branches

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copy branch name to clipboard |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` s d `` | Checkout detached HEAD |  |
| `` g y `` | Show git-flow options |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | Checkout | Checkout selected item. |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` g p c `` | Create pull request |  |
| `` g p o `` | View create pull request options |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | Copy pull request URL to clipboard |  |
| `` <delete> `` | Delete | View delete options for local/remote branch. |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` m `` | Merge selected branch into current… | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | New tag |  |
| `` t s o `` | Sort order |  |
| `` <f2> `` | Rename branch |  |
| `` g t `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View commits |  |
| `` / `` | Filter the current view by text |  |

## Main panel (merging)

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | Pick hunk |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | Previous hunk |  |
| `` j, <down> `` | Next hunk |  |
| `` [, t p c `` | Previous conflict |  |
| `` ], t n c `` | Next conflict |  |
| `` u u `` | Undo | Undo last merge conflict resolution. |
| `` e `` | Edit file | Open file in external editor. |
| `` o `` | Open file | Open file in default application. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | Return to files panel |  |

## Main panel (normal)

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Scroll down |  |
| `` <mouse wheel up> (fn+down) `` | Scroll up |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Toggle range select |  |
| `` e `` | Edit file | Open file in external editor. |
| `` <space>, g h s `` | Stage | Toggle selection staged / unstaged. |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Toggle lines in patch |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Discard | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Go to previous hunk |  |
| `` ], t n h `` | Go to next hunk |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` g e c `` | Commit changes using git editor |  |
| `` / `` | Search the current view by text |  |

## Menu

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Execute |  |
| `` <esc> `` | Close/Cancel |  |
| `` / `` | Filter the current view by text |  |

## Reflog

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Checkout | Checkout the selected commit as a detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | Select commits of current branch |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | Focus main view |  |
| `` / `` | Filter the current view by text |  |

## Remote branches

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copy branch name to clipboard |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` s d `` | Checkout detached HEAD |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | Checkout | Checkout a new local branch based on the selected remote branch, or the remote branch as a detached head. |
| `` n w `` | New worktree |  |
| `` m `` | Merge | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` <delete> `` | Delete | Delete the remote branch from the remote. |
| `` g t `` | Set as upstream | Set the selected remote branch as the upstream of the checked-out branch. |
| `` t s o `` | Sort order |  |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View commits |  |
| `` / `` | Filter the current view by text |  |

## Remotes

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | View branches |  |
| `` n r `` | New remote |  |
| `` <delete> `` | Remove | Remove the selected remote. Any local branches tracking a remote branch from the remote will be unaffected. |
| `` e `` | Edit | Edit the selected remote's name or URL. |
| `` f `` | Fetch | Fetch updates from the remote repository. This retrieves new commits and branches without merging them into your local branches. |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | Filter the current view by text |  |

## Secondary

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Toggle range select |  |
| `` e `` | Edit file | Open file in external editor. |
| `` <space>, g h s `` | Stage | Toggle selection staged / unstaged. |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Toggle lines in patch |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Discard | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Go to previous hunk |  |
| `` ], t n h `` | Go to next hunk |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` g e c `` | Commit changes using git editor |  |
| `` / `` | Search the current view by text |  |

## Stash

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | Apply | Apply the stash entry to your working directory. |
| `` g s p `` | Pop | Apply the stash entry to your working directory and remove the stash entry. |
| `` <delete> `` | Drop | Remove the stash entry from the stash list. |
| `` n w `` | New worktree |  |
| `` <f2> `` | Rename stash |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View files |  |
| `` / `` | Filter the current view by text |  |

## Status

| Key | Action | Info |
|-----|--------|-------------|
| `` s n `` | Open repository by path |  |
| `` t a c `` | Edit config file | Open file in external editor. |
| `` t a u `` | Check for update |  |
| `` s r `` | Switch to a recent repo |  |
| `` t l n `` | Show/cycle all branch logs |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space>, 0 `` | Focus main view |  |

## Sub-commits

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Checkout | Checkout the selected commit as a detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | Select commits of current branch |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View files |  |
| `` / `` | Search the current view by text |  |

## Submodules

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copy submodule name to clipboard |  |
| `` v, <space> `` | Enter | Enter submodule. After entering the submodule, you can press `<esc>` to escape back to the parent repo. |
| `` <delete> `` | Remove | Remove the selected submodule and its corresponding directory. |
| `` g u s `` | Update | Update selected submodule. |
| `` n s `` | New submodule |  |
| `` e `` | Update submodule URL |  |
| `` g u i `` | Initialize | Initialize the selected submodule to prepare for fetching. You probably want to follow this up by invoking the 'update' action to fetch the submodule. |
| `` g u b `` | View bulk submodule options |  |
| `` t a g `` | Easter egg |  |
| `` / `` | Filter the current view by text |  |

## Tags

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copy tag to clipboard |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s c `` | Checkout | Checkout the selected tag as a detached HEAD. |
| `` n t `` | New tag | Create new tag from current commit. You'll be prompted to enter a tag name and optional description. |
| `` n w `` | New worktree |  |
| `` <delete> `` | Delete | View delete options for local/remote tag. |
| `` p `` | Push tag | Push the selected tag to a remote. You'll be prompted to select a remote. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | View commits |  |
| `` / `` | Filter the current view by text |  |

## Worktrees

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | New worktree |  |
| `` s w `` | Switch | Switch to the selected worktree. |
| `` o `` | Open in editor |  |
| `` <delete> `` | Remove | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` v, <space> `` | View commits |  |
| `` / `` | Filter the current view by text |  |
