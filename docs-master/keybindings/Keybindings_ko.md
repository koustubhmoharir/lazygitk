_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit 키 바인딩

## 글로벌 키 바인딩

| Key | Action | Info |
|-----|--------|-------------|
| `` s r `` | 최근에 사용한 저장소로 전환 |  |
| `` s n `` | Open repository by path |  |
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | 메인 패널을 위로 스크롤 |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | 메인 패널을 아래로로 스크롤 |  |
| `` t l c `` | 명령어 로그 메뉴 열기 | View options for the command log e.g. show/hide the command log and focus the command log. |
| `` p `` | 푸시 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | 업데이트 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Diff 보기의 변경 사항 주위에 표시되는 컨텍스트의 크기를 늘리기 | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Diff 보기의 변경 사항 주위에 표시되는 컨텍스트 크기 줄이기 | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | 체리픽 |  |
| `` : `` | Execute shell command | Bring up a prompt where you can enter a shell command to execute. |
| `` g h o `` | 커스텀 Patch 옵션 보기 |  |
| `` g o `` | View merge/rebase options | View options to abort/continue/skip the current merge/rebase. |
| `` <f5> `` | 새로고침 | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` t s n `` | 다음 스크린 모드 (normal/half/fullscreen) |  |
| `` t s p `` | 이전 스크린 모드 |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | 취소 |  |
| `` ? `` | 매뉴 열기 |  |
| `` t f o `` | View filter-by-path options | View options for filtering the commit log, so that only commits matching the filter are shown. |
| `` t d c `` | Diff 메뉴 열기 | View options relating to diffing two refs e.g. diffing against selected ref, entering ref to diff against, and reversing the diff direction. |
| `` q `` | 종료 |  |
| `` t a s `` | Suspend the application |  |
| `` t d w `` | 공백문자를 Diff 뷰에서 표시 여부 전환 | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | 설정 파일 수정 | Open file in external editor. |
| `` u u `` | 되돌리기 (reflog) (실험적) | The reflog will be used to determine what git command to run to undo the last git command. This does not include changes to the working tree; only commits are taken into consideration. |
| `` u r `` | 다시 실행 (reflog) (실험적) | The reflog will be used to determine what git command to run to redo the last git command. This does not include changes to the working tree; only commits are taken into consideration. |

## List panel navigation

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus 상태 panel |  |
| `` 2 `` | Focus 파일 panel |  |
| `` 3 `` | Focus 브랜치 panel |  |
| `` 4 `` | Focus 커밋 panel |  |
| `` 5 `` | Focus Stash panel |  |
| `` <pgup> `` | 이전 페이지 |  |
| `` <pgdown> `` | 다음 페이지 |  |
| `` <home> `` | 맨 위로 스크롤  |  |
| `` <end> `` | 맨 아래로 스크롤  |  |
| `` t h r `` | 드래그 선택 전환 |  |
| `` <shift+down> `` | Range select down |  |
| `` <shift+up> `` | Range select up |  |
| `` / `` | 검색 시작 |  |
| `` , `` | 이전 페이지 |  |
| `` . `` | 다음 페이지 |  |
| `` <, <home> `` | 맨 위로 스크롤  |  |
| `` >, <end> `` | 맨 아래로 스크롤  |  |
| `` <alt+left> `` | 우 스크롤 |  |
| `` <alt+right> `` | 좌 스크롤 |  |
| `` <tab> `` | 이전 탭 |  |
| `` <backtab> `` | 다음 탭 |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 확인 |  |
| `` <esc> `` | 닫기/취소 |  |

## Reflog

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 체크아웃 | Checkout the selected commit as a detached HEAD. |
| `` o `` | 브라우저에서 커밋 열기 |  |
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
| `` t f t `` | Filter the current view by text |  |

## Secondary

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | 드래그 선택 전환 |  |
| `` e `` | 파일 편집 | Open file in external editor. |
| `` g h s `` | Staged 전환 | 선택한 행을 staged / unstaged |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | Line(s)을 패치에 추가/삭제 |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 변경을 삭제 (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | 이전 hunk를 선택 |  |
| `` t n h `` | 다음 hunk를 선택 |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` g e c `` | Git 편집기를 사용하여 변경 내용을 커밋합니다. |  |
| `` / `` | 검색 시작 |  |

## Stash

| Key | Action | Info |
|-----|--------|-------------|
| `` g a `` | 적용 | Apply the stash entry to your working directory. |
| `` g s p `` | Pop | Apply the stash entry to your working directory and remove the stash entry. |
| `` <delete> `` | Drop | Remove the stash entry from the stash list. |
| `` n w `` | New worktree |  |
| `` <f2> `` | Rename stash |  |
| `` v, <space> `` | View selected item's files |  |
| `` t f t `` | Filter the current view by text |  |

## Sub-commits

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 체크아웃 | Checkout the selected commit as a detached HEAD. |
| `` o `` | 브라우저에서 커밋 열기 |  |
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
| `` v, <space> `` | View selected item's files |  |
| `` / `` | 검색 시작 |  |

## Worktrees

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | New worktree |  |
| `` t r w `` | Switch | Switch to the selected worktree. |
| `` o `` | Open in editor |  |
| `` <delete> `` | Remove | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` t f t `` | Filter the current view by text |  |

## 메뉴

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 실행 |  |
| `` <esc> `` | 닫기/취소 |  |
| `` / `` | Filter the current view by text |  |

## 메인 패널 (Merging)

| Key | Action | Info |
|-----|--------|-------------|
| `` g h p `` | Pick hunk |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | 이전 hunk를 선택 |  |
| `` j, <down> `` | 다음 hunk를 선택 |  |
| `` t p c `` | 이전 충돌을 선택 |  |
| `` t n c `` | 다음 충돌을 선택 |  |
| `` u u `` | 되돌리기 | Undo last merge conflict resolution. |
| `` e `` | 파일 편집 | Open file in external editor. |
| `` o `` | 파일 닫기 | Open file in default application. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | 파일 목록으로 돌아가기 |  |

## 메인 패널 (Normal)

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | 아래로 스크롤 |  |
| `` <mouse wheel up> (fn+down) `` | 위로 스크롤 |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | 드래그 선택 전환 |  |
| `` e `` | 파일 편집 | Open file in external editor. |
| `` g h s `` | Staged 전환 | 선택한 행을 staged / unstaged |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | Line(s)을 패치에 추가/삭제 |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 변경을 삭제 (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | 이전 hunk를 선택 |  |
| `` t n h `` | 다음 hunk를 선택 |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` g e c `` | Git 편집기를 사용하여 변경 내용을 커밋합니다. |  |
| `` / `` | 검색 시작 |  |

## 브랜치

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 브랜치명을 클립보드에 복사 |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` g y `` | Git-flow 옵션 보기 |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | 체크아웃 | Checkout selected item. |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` g p c `` | 풀 리퀘스트 생성 |  |
| `` g p o `` | 풀 리퀘스트 생성 옵션 |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | 풀 리퀘스트 URL을 클립보드에 복사 |  |
| `` <delete> `` | 삭제 | View delete options for local/remote branch. |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` m `` | Merge selected branch into current… | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` g f `` | Fast-forward selected branches… | Review an upstream fetch and a forward-only update of the captured selected branches. Divergent or rewritten history is never reset; tips occupied by another worktree are not moved. A selected checked-out branch uses native fast-forward integration with explicit review. Multiple selections require final batch approval. Saved upstream settings stay unchanged. |
| `` n t `` | 태그를 생성 |  |
| `` t s o `` | Sort order |  |
| `` <f2> `` | 브랜치 이름 변경 |  |
| `` g t `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` v, <space> `` | 커밋 보기 |  |
| `` t f t `` | Filter the current view by text |  |

## 상태

| Key | Action | Info |
|-----|--------|-------------|
| `` t a c `` | 설정 파일 수정 | Open file in external editor. |
| `` t a u `` | 업데이트 확인 |  |
| `` s r `` | 최근에 사용한 저장소로 전환 |  |
| `` t l n `` | Show/cycle all branch logs |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space> `` | Focus main view |  |

## 서브모듈

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 서브모듈 이름을 클립보드에 복사 |  |
| `` t r s `` | Enter | 서브모듈 열기 |
| `` <delete> `` | Remove | Remove the selected submodule and its corresponding directory. |
| `` g u s `` | Update | 서브모듈 업데이트 |
| `` n s `` | 새로운 서브모듈 추가 |  |
| `` e `` | 서브모듈의 URL을 수정 |  |
| `` g u i `` | Initialize | 서브모듈 초기화 |
| `` g u b `` | View bulk submodule options |  |
| `` t a g `` | 이스터 에그 |  |
| `` t f t `` | Filter the current view by text |  |

## 원격

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | View branches |  |
| `` n r `` | 새로운 Remote 추가 |  |
| `` <delete> `` | Remove | Remove the selected remote. Any local branches tracking a remote branch from the remote will be unaffected. |
| `` e `` | Edit | Remote를 수정 |
| `` f `` | Fetch | 원격을 업데이트 |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` t f t `` | Filter the current view by text |  |

## 원격 브랜치

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 브랜치명을 클립보드에 복사 |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | 체크아웃 | Checkout a new local branch based on the selected remote branch, or the remote branch as a detached head. |
| `` n w `` | New worktree |  |
| `` m `` | 현재 브랜치에 병합 | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` <delete> `` | 삭제 | Delete the remote branch from the remote. |
| `` g t `` | Set as upstream | Set the selected remote branch as the upstream of the checked-out branch. |
| `` t s o `` | Sort order |  |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` v, <space> `` | 커밋 보기 |  |
| `` t f t `` | Filter the current view by text |  |

## 커밋

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | Bisect 옵션 보기 |  |
| `` <f2> `` | 커밋메시지 변경 | Reword the selected commit's message. |
| `` g e r `` | 에디터에서 커밋메시지 수정 |  |
| `` <delete> `` | 커밋 삭제 | Drop the selected commit. This will remove the commit from the branch via a rebase. If the commit makes changes that later commits depend on, you may need to resolve merge conflicts. |
| `` e `` | Edit (start interactive rebase) | 커밋을 편집 |
| `` g i `` | Start interactive rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` g r p `` | Pick | Pick commit (when mid-rebase) |
| `` g m d `` | 커밋을 1개 아래로 이동 |  |
| `` g m u `` | 커밋을 1개 위로 이동 |  |
| `` c p `` | 체리픽 |  |
| `` g r b `` | Mark as base commit for rebase | Select a base commit for the next rebase. When you rebase onto a branch, only commits above the base commit will be brought across. This uses the `git rebase --onto` command. |
| `` a `` | Amend | Amend commit with staged changes |
| `` g e a `` | Amend commit attribute | Set/Reset commit author or set co-author. |
| `` g v c `` | Revert | Create a revert commit for the selected commit, which applies the selected commit's changes in reverse. |
| `` n t `` | Tag commit | Create a new tag pointing at the selected commit. You'll be prompted to enter a tag name and optional description. |
| `` t l o `` | 로그 메뉴 열기 | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
| `` g p v `` | Open pull request in browser |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 체크아웃 | Checkout the selected commit as a detached HEAD. |
| `` o `` | 브라우저에서 커밋 열기 |  |
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
| `` v, <space> `` | View selected item's files |  |
| `` / `` | 검색 시작 |  |

## 커밋 파일

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 파일명을 클립보드에 복사 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | 체크아웃 | Checkout file |
| `` x `` | View 'discard changes' options | Discard this commit's changes to this file |
| `` o `` | 파일 닫기 | Open file in default application. |
| `` e `` | Edit | Open file in external editor. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Toggle file included in patch | Toggle whether the file is included in the custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Toggle all files included in patch | Add/remove all commit's files to custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | 파일 트리뷰로 전환 | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Expand all files | Expand all directories in the file tree |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` t f t `` | Filter the current view by text |  |

## 커밋메시지

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 확인 |  |
| `` <esc> `` | 닫기 |  |

## 태그

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
| `` s c `` | 체크아웃 | Checkout the selected tag as a detached HEAD. |
| `` n t `` | 태그를 생성 | Create new tag from current commit. You'll be prompted to enter a tag name and optional description. |
| `` n w `` | New worktree |  |
| `` <delete> `` | 삭제 | View delete options for local/remote tag. |
| `` p `` | 태그를 push | Push the selected tag to a remote. You'll be prompted to select a remote. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` v, <space> `` | 커밋 보기 |  |
| `` t f t `` | Filter the current view by text |  |

## 파일

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 파일명을 클립보드에 복사 |  |
| `` <space> `` | Toggle staging of selected files | Toggle staged for selected file. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` t f s `` | 파일을 필터하기 (Staged/unstaged) |  |
| `` <enter> `` | 커밋 변경내용 | 스테이징된 변경 사항 커밋. |
| `` g e n `` | Commit changes without pre-commit hook |  |
| `` a `` | 마지맛 커밋 수정 |  |
| `` g e c `` | Git 편집기를 사용하여 변경 내용을 커밋합니다. |  |
| `` e `` | Edit | Open file in external editor. |
| `` o `` | 파일 닫기 | Open file in default application. |
| `` i `` | Ignore file |  |
| `` <f5> `` | 새로고침 | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` g s s `` | Stash | Stash all changes. For other variations of stashing, use the view stash options keybinding. |
| `` g s o `` | Stash 옵션 보기 | View stash options (e.g. stash all, stash staged, stash unstaged). |
| `` x `` | View 'discard changes' options | View options for discarding changes to the selected file. |
| `` g z u `` | View upstream reset options |  |
| `` g z w `` | 초기화 | View reset options for working tree (e.g. nuking the working tree). |
| `` t t t `` | 파일 트리뷰로 전환 | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` f `` | Fetch | Fetch changes from remote. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Expand all files | Expand all directories in the file tree |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y w `` | Copy all files diff |  |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` v `` | Focus main view |  |
| `` t f t `` | Filter the current view by text |  |

## 확인 패널

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 확인 |  |
| `` <esc> `` | 닫기/취소 |  |
| `` <ctrl+o> `` | 클립보드에 복사 |  |
