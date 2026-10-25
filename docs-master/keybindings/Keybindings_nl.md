_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Sneltoetsen

## Globale sneltoetsen

| Key | Action | Info |
|-----|--------|-------------|
| `` s r `` | Wissel naar een recente repo |  |
| `` s n `` | Open repository by path |  |
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | Scroll naar beneden vanaf hoofdpaneel |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | Scroll naar beneden vanaf hoofdpaneel |  |
| `` t l c `` | Commandolog opties weergeven | Bekijk commandolog opties, bijv. commandolog tonen/verbergen en focus. |
| `` p `` | Push | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | Pull | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Increase diff context size | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Decrease diff context size | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | Cherry-Pick |  |
| `` : `` | Voer shellcommando uit | Bring up a prompt where you can enter a shell command to execute. |
| `` g h o `` | Bekijk aangepaste patch opties |  |
| `` g o `` | Bekijk merge/rebase opties | Toon abort/continue/skip opties voor huidige merge/rebase. |
| `` <f5> `` | Verversen | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` t s n `` | Volgende scherm modus (normaal/half/groot) |  |
| `` t s p `` | Vorige scherm modus |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Annuleren |  |
| `` ? `` | Open menu |  |
| `` t f o `` | Bekijk scoping opties | View options for filtering the commit log, so that only commits matching the filter are shown. |
| `` t d c `` | Open diff menu | View options relating to diffing two refs e.g. diffing against selected ref, entering ref to diff against, and reversing the diff direction. |
| `` q `` | Afsluiten |  |
| `` t a s `` | Pauzeer de applicatie |  |
| `` t d w `` | Witruimte weergeven in-/uitschakelen | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | Verander config bestand | Open bestand in externe editor. |
| `` u u `` | Ongedaan maken (via reflog) (experimenteel) | The reflog will be used to determine what git command to run to undo the last git command. This does not include changes to the working tree; only commits are taken into consideration. |
| `` u r `` | Redo (via reflog) (experimenteel) | Het reflog wordt gebruikt om te bepalen welk git commando moet worden gebruikt om het laatste git commando te herhalen. Wijzigingen aan de working tree worden niet meegenomen, alleen command's zijn kandidaten. |

## Lijstpaneel navigatie

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus Status panel |  |
| `` 2 `` | Focus Bestanden panel |  |
| `` 3 `` | Focus Branches panel |  |
| `` 4 `` | Focus Commits panel |  |
| `` 5 `` | Focus Stash panel |  |
| `` <pgup> `` | Vorige pagina |  |
| `` <pgdown> `` | Volgende pagina |  |
| `` <home> `` | Scroll naar boven |  |
| `` <end> `` | Scroll naar beneden |  |
| `` t h r `` | Toggle drag selecteer |  |
| `` <shift+down> `` | Range select down |  |
| `` <shift+up> `` | Range select up |  |
| `` / `` | Start met zoeken |  |
| `` , `` | Vorige pagina |  |
| `` . `` | Volgende pagina |  |
| `` <, <home> `` | Scroll naar boven |  |
| `` >, <end> `` | Scroll naar beneden |  |
| `` <alt+left> `` | Scroll naar links |  |
| `` <alt+right> `` | Scroll naar rechts |  |
| `` <tab> `` | Volgende tabblad |  |
| `` <backtab> `` | Vorige tabblad |  |

## Bestanden

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Kopieer de bestandsnaam naar het klembord |  |
| `` <space> `` | Toggle staging of selected files | Toggle staged for selected file. |
| `` g h c `` | Bekijk merge conflict opties | Bekijk opties voor het oplossen van mergeconflicten. |
| `` t f s `` | Filter bestanden op status |  |
| `` <enter> `` | Commit veranderingen | Commit gestagede wijzigingen. |
| `` g e n `` | Commit veranderingen zonder pre-commit hook |  |
| `` a `` | Wijzig laatste commit |  |
| `` g e c `` | Commit veranderingen met de git editor |  |
| `` e `` | Edit | Open bestand in externe editor. |
| `` o `` | Open bestand | Open bestand in standaardapplicatie. |
| `` i `` | Ignore or exclude file |  |
| `` <f5> `` | Verversen | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` g s s `` | Stash | Stash all changes. For other variations of stashing, use the view stash options keybinding. |
| `` g s o `` | Bekijk stash opties | View stash options (e.g. stash all, stash staged, stash unstaged). |
| `` x `` | Bekijk 'veranderingen ongedaan maken' opties | View options for discarding changes to the selected file. |
| `` g z u `` | Bekijk upstream reset opties |  |
| `` g z w `` | Resetten | View reset options for working tree (e.g. nuking the working tree). |
| `` t t t `` | Toggle bestandsboom weergave | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t d e `` | Open externe diff applicatie (git difftool) |  |
| `` f `` | Fetch | Fetch changes from remote. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Vouw alle bestanden uit | Vouw alle mappen in de bestandsstructuur uit |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y w `` | Copy all files diff |  |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` v `` | Focus main view |  |
| `` t f t `` | Filter the current view by text |  |

## Bevestigingspaneel

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Bevestig |  |
| `` <esc> `` | Sluiten |  |
| `` <ctrl+o> `` | Kopieer naar klembord |  |

## Branches

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopieer branch name naar klembord |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` g y `` | Laat git-flow opties zien |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | Uitchecken | Geselecteerd item uitchecken. |
| `` g m n `` | Verplaats commits naar nieuwe branch | Maak een nieuwe branch en verplaats niet-gepushte commits van de huidige branch hier naar toe. Gebruik dit in het geval dat je deze commits eigenlijk op een nieuwe branch had willen maken.<br><br>Let op dat de selectie genegeerd wordt. De nieuwe branch komt ofwel bovenop de main branch, of bovenop de huidige branch (je kan kiezen). |
| `` n w `` | New worktree |  |
| `` g p c `` | Maak een pull-request |  |
| `` g p o `` | Bekijk opties voor pull-aanvraag |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | Kopieer de URL van het pull-verzoek naar het klembord |  |
| `` <delete> `` | Verwijderen | View delete options for local/remote branch. |
| `` r `` | Rebase current onto selected branch… | Rebase de uitgecheckte branch bovenop de geselecteerde branch. |
| `` m `` | Merge selected branch into current… | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` g f `` | Fast-forward selected branches… | Review an upstream fetch and a forward-only update of the captured selected branches. Divergent or rewritten history is never reset; tips occupied by another worktree are not moved. A selected checked-out branch uses native fast-forward integration with explicit review. Multiple selections require final batch approval. Saved upstream settings stay unchanged. |
| `` n t `` | Creëer tag |  |
| `` t s o `` | Sort order |  |
| `` <f2> `` | Hernoem branch |  |
| `` g t `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` t d e `` | Open externe diff applicatie (git difftool) |  |
| `` v, <space> `` | Bekijk commits |  |
| `` t f t `` | Filter the current view by text |  |

## Commit bericht

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Bevestig |  |
| `` <esc> `` | Sluiten |  |

## Commit bestanden

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Kopieer de bestandsnaam naar het klembord |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | Uitchecken | Bestand uitchecken |
| `` x `` | Bekijk 'veranderingen ongedaan maken' opties | Uitsluit deze commit zijn veranderingen aan dit bestand |
| `` o `` | Open bestand | Open bestand in standaardapplicatie. |
| `` e `` | Edit | Open bestand in externe editor. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Toggle bestand inbegrepen in patch | Toggle whether the file is included in the custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Toggle all files | Add/remove all commit's files to custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | Toggle bestandsboom weergave | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Vouw alle bestanden uit | Vouw alle mappen in de bestandsstructuur uit |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` t f t `` | Filter the current view by text |  |

## Commits

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | View bisect options |  |
| `` <f2> `` | Hernoem commit | Herschrijf de commit message van de geselecteerde commit. |
| `` g e r `` | Hernoem commit met editor |  |
| `` <delete> `` | Verwijder commit | Drop the selected commit. This will remove the commit from the branch via a rebase. If the commit makes changes that later commits depend on, you may need to resolve merge conflicts. |
| `` e `` | Bewerken (start interactieve rebase) | Wijzig commit |
| `` g i `` | Start interactieve rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` g r p `` | Pick | Kies commit (wanneer midden in rebase) |
| `` g m d `` | Verplaats commit 1 naar beneden |  |
| `` g m u `` | Verplaats commit 1 naar boven |  |
| `` c p `` | Cherry-Pick |  |
| `` g r b `` | Markeer als basiscommit voor rebase | Selecteer een basiscommit voor de volgende rebase. Als je rebased op een branch worden alleen commits boven de basiscommit meegenomen. Hiervoor wordt het `git rebase --onto` commando gebruikt. |
| `` a `` | Amend | Wijzig commit met staged veranderingen |
| `` g e a `` | Amend commit attribute | Set/Reset commit author or set co-author. |
| `` g v c `` | Revert | Maak een revert commit voor de geselecteerde commit, die de wijzigingen in deze commit terugdraait. |
| `` n t `` | Tag commit | Maak een nieuwe tag die naar de geselecteerde commit wijst. Je wordt gevraagd om een tag naam en optionele omschrijving. |
| `` t l o `` | Log opties weergeven | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
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
| `` s c `` | Uitchecken | Check de geselecteerde branch uit als een detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Verplaats commits naar nieuwe branch | Maak een nieuwe branch en verplaats niet-gepushte commits van de huidige branch hier naar toe. Gebruik dit in het geval dat je deze commits eigenlijk op een nieuwe branch had willen maken.<br><br>Let op dat de selectie genegeerd wordt. De nieuwe branch komt ofwel bovenop de main branch, of bovenop de huidige branch (je kan kiezen). |
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
| `` v, <space> `` | Bekijk gecommite bestanden |  |
| `` / `` | Start met zoeken |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Bevestig |  |
| `` <esc> `` | Sluiten |  |

## Menu

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Uitvoeren |  |
| `` <esc> `` | Sluiten |  |
| `` / `` | Filter the current view by text |  |

## Mergen

| Key | Action | Info |
|-----|--------|-------------|
| `` g h p `` | Kies stuk |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | Selecteer bovenste hunk |  |
| `` j, <down> `` | Selecteer onderste hunk |  |
| `` t p c `` | Selecteer voorgaand conflict |  |
| `` t n c `` | Selecteer volgende conflict |  |
| `` u u `` | Ongedaan maken | Undo last merge conflict resolution. |
| `` e `` | Verander bestand | Open bestand in externe editor. |
| `` o `` | Open bestand | Open bestand in standaardapplicatie. |
| `` g h c `` | Bekijk merge conflict opties | Bekijk opties voor het oplossen van mergeconflicten. |
| `` <esc> `` | Ga terug naar het bestanden paneel |  |

## Normaal

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Scroll omlaag |  |
| `` <mouse wheel up> (fn+down) `` | Scroll omhoog |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Wissel tussen hunk selectie aan of uit | Wissel tussen regel-voor-regel of hunk selectie modus. |
| `` t h r `` | Toggle drag selecteer |  |
| `` e `` | Verander bestand | Open bestand in externe editor. |
| `` g h s `` | Toggle staged | Toggle lijnen staged / unstaged |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | Voeg toe/verwijder lijn(en) in patch |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Verwijdert change (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | Selecteer de vorige hunk |  |
| `` t n h `` | Selecteer de volgende hunk |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit veranderingen zonder pre-commit hook |  |
| `` g e c `` | Commit veranderingen met de git editor |  |
| `` / `` | Start met zoeken |  |

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
| `` s c `` | Uitchecken | Check de geselecteerde branch uit als een detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Verplaats commits naar nieuwe branch | Maak een nieuwe branch en verplaats niet-gepushte commits van de huidige branch hier naar toe. Gebruik dit in het geval dat je deze commits eigenlijk op een nieuwe branch had willen maken.<br><br>Let op dat de selectie genegeerd wordt. De nieuwe branch komt ofwel bovenop de main branch, of bovenop de huidige branch (je kan kiezen). |
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

## Remote branches

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopieer branch name naar klembord |  |
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
| `` s c `` | Uitchecken | Geselecteerde remote branch uitchecken als nieuwe locale branch of als detached head. |
| `` n w `` | New worktree |  |
| `` m `` | Merge in met huidige checked out branch | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` r `` | Rebase current onto selected branch… | Rebase de uitgecheckte branch bovenop de geselecteerde branch. |
| `` <delete> `` | Verwijderen | Verwijder de remote branch van de remote. |
| `` g t `` | Instellen als upstream | Stel in als upstream van uitgecheckte branch |
| `` t s o `` | Sort order |  |
| `` t d e `` | Open externe diff applicatie (git difftool) |  |
| `` v, <space> `` | Bekijk commits |  |
| `` t f t `` | Filter the current view by text |  |

## Remotes

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | Bekijk branches |  |
| `` n r `` | Voeg een nieuwe remote toe |  |
| `` <delete> `` | Verwijderen | Verwijder de geselecteerde remote. Locale branches die een branch tracken van de remote worden niet aangepast. |
| `` e `` | Edit | Wijzig remote |
| `` f `` | Fetch | Fetch remote |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` t f t `` | Filter the current view by text |  |

## Secondary

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Wissel tussen hunk selectie aan of uit | Wissel tussen regel-voor-regel of hunk selectie modus. |
| `` t h r `` | Toggle drag selecteer |  |
| `` e `` | Verander bestand | Open bestand in externe editor. |
| `` g h s `` | Toggle staged | Toggle lijnen staged / unstaged |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | Voeg toe/verwijder lijn(en) in patch |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Verwijdert change (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edit hunk | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | Selecteer de vorige hunk |  |
| `` t n h `` | Selecteer de volgende hunk |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Commit veranderingen zonder pre-commit hook |  |
| `` g e c `` | Commit veranderingen met de git editor |  |
| `` / `` | Start met zoeken |  |

## Stash

| Key | Action | Info |
|-----|--------|-------------|
| `` g a `` | Toepassen | Apply the stash entry to your working directory. |
| `` g s p `` | Pop | Apply the stash entry to your working directory and remove the stash entry. |
| `` <delete> `` | Laten vallen | Remove the stash entry from the stash list. |
| `` n w `` | New worktree |  |
| `` <f2> `` | Hernoem stash |  |
| `` v, <space> `` | Bekijk gecommite bestanden |  |
| `` t f t `` | Filter the current view by text |  |

## Status

| Key | Action | Info |
|-----|--------|-------------|
| `` t a c `` | Verander config bestand | Open bestand in externe editor. |
| `` t a u `` | Check voor updates |  |
| `` s r `` | Wissel naar een recente repo |  |
| `` t l n `` | Show/cycle all branch logs |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space> `` | Focus main view |  |

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
| `` s c `` | Uitchecken | Check de geselecteerde branch uit als een detached HEAD. |
| `` o `` | Open commit in browser |  |
| `` g m n `` | Verplaats commits naar nieuwe branch | Maak een nieuwe branch en verplaats niet-gepushte commits van de huidige branch hier naar toe. Gebruik dit in het geval dat je deze commits eigenlijk op een nieuwe branch had willen maken.<br><br>Let op dat de selectie genegeerd wordt. De nieuwe branch komt ofwel bovenop de main branch, of bovenop de huidige branch (je kan kiezen). |
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
| `` v, <space> `` | Bekijk gecommite bestanden |  |
| `` / `` | Start met zoeken |  |

## Submodules

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopieer submodule naam naar klembord |  |
| `` t r s `` | Enter | Enter submodule |
| `` <delete> `` | Verwijderen | Remove the selected submodule and its corresponding directory. |
| `` g u s `` | Update | Update selected submodule. |
| `` n s `` | Voeg nieuwe submodule toe |  |
| `` e `` | Update submodule URL |  |
| `` g u i `` | Initialize | Initialiseer submodule |
| `` g u b `` | Bekijk bulk submodule opties |  |
| `` t a g `` | Easter egg |  |
| `` t f t `` | Filter the current view by text |  |

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
| `` s c `` | Uitchecken | Geselecteerde tag uitchecken als detached HEAD. |
| `` n t `` | Creëer tag | Maak een nieuwe tag die naar de huidige commit wijst. Je wordt gevraagd om een tag naam en optionele beschrijving. |
| `` n w `` | New worktree |  |
| `` <delete> `` | Verwijderen | View delete options for local/remote tag. |
| `` p `` | Tag pushen | Push de geselecteerde tag naar een remote. Je krijgt de optie een remote te selecteren. |
| `` t d e `` | Open externe diff applicatie (git difftool) |  |
| `` v, <space> `` | Bekijk commits |  |
| `` t f t `` | Filter the current view by text |  |

## Worktrees

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | New worktree |  |
| `` t r w `` | Switch | Switch to the selected worktree. |
| `` o `` | Openen in editor |  |
| `` <delete> `` | Verwijderen | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` t f t `` | Filter the current view by text |  |
