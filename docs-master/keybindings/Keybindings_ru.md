_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Связки клавиш

## Глобальные сочетания клавиш

| Key | Action | Info |
|-----|--------|-------------|
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | Прокрутить вверх главную панель |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | Прокрутить вниз главную панель |  |
| `` t l c `` | Открыть меню журнала команд | View options for the command log e.g. show/hide the command log and focus the command log. |
| `` p `` | Отправить изменения | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | Получить и слить изменения | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Увеличить размер контекста, отображаемого вокруг изменений в просмотрщике сравнении | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Уменьшите размер контекста, отображаемого вокруг изменений в просмотрщике сравнении | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | Выборочная отборка (Cherry-Pick) |  |
| `` f `` | Получить изменения | Fetch changes from remote. |
| `` : `` | Execute shell command | Bring up a prompt where you can enter a shell command to execute. |
| `` g h o `` | Просмотреть пользовательские параметры патча |  |
| `` g o `` | Просмотреть параметры слияния/перебазирования | View options to abort/continue/skip the current merge/rebase. |
| `` <f5> `` | Обновить | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` =, + `` | Следующий режим экрана (нормальный/полуэкранный/полноэкранный) |  |
| `` - `` | Предыдущий режим экрана |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Отменить |  |
| `` ? `` | Открыть меню |  |
| `` w o `` | Просмотреть параметры фильтрации по пути | View options for filtering the commit log, so that only commits matching the filter are shown. |
| `` t d c `` | Открыть меню сравнении | View options relating to diffing two refs e.g. diffing against selected ref, entering ref to diff against, and reversing the diff direction. |
| `` q `` | Выйти |  |
| `` t a s `` | Suspend the application |  |
| `` t d w `` | Переключить отображение изменении пробелов в просмотрщике сравнении | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | Редактировать файл конфигурации | Open file in external editor. |
| `` u u `` | Отменить (через reflog) (экспериментальный) | Журнал ссылок (reflog) будет использоваться для определения того, какую команду git запустить, чтобы отменить последнюю команду git. Сюда не входят изменения в рабочем дереве; учитываются только коммиты. |
| `` u r `` | Повторить (через reflog) (экспериментальный) | Журнал ссылок (reflog) будет использоваться для определения того, какую команду git нужно запустить, чтобы повторить последнюю команду git. Сюда не входят изменения в рабочем дереве; учитываются только коммиты. |

## Навигация по панели списка

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus Статус panel |  |
| `` 2 `` | Focus Файлы panel |  |
| `` 3 `` | Focus Ветки panel |  |
| `` 4 `` | Focus Коммиты panel |  |
| `` 5 `` | Focus Хранилище panel |  |
| `` <pgup> `` | Предыдущая страница |  |
| `` <pgdown> `` | Следующая страница |  |
| `` <home> `` | Пролистать наверх |  |
| `` <end> `` | Прокрутить вниз |  |
| `` t h r `` | Переключить выборку перетаскивания |  |
| `` <shift+down>, J `` | Range select down |  |
| `` <shift+up>, K `` | Range select up |  |
| `` / `` | Найти |  |
| `` , `` | Предыдущая страница |  |
| `` . `` | Следующая страница |  |
| `` <, <home> `` | Пролистать наверх |  |
| `` >, <end> `` | Прокрутить вниз |  |
| `` <alt+left> `` | Прокрутить влево |  |
| `` <alt+right> `` | Прокрутить вправо |  |
| `` <tab> `` | Следующая вкладка |  |
| `` <backtab> `` | Предыдущая вкладка |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Подтвердить |  |
| `` <esc> `` | Закрыть/отменить |  |

## Worktrees

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | New worktree |  |
| `` s w `` | Switch | Switch to the selected worktree. |
| `` o `` | Open in editor |  |
| `` <delete> `` | Remove | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` v, <space> `` | Просмотреть коммиты |  |
| `` / `` | Filter the current view by text |  |

## Вторичный

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Переключить выборку перетаскивания |  |
| `` e `` | Редактировать файл | Open file in external editor. |
| `` <space>, g h s `` | Переключить индекс | Переключить строку в проиндексированные / непроиндексированные |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Добавить/удалить строку(и) для патча |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Отменить изменение (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Изменить эту часть | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Выбрать предыдущую часть |  |
| `` ], t n h `` | Выбрать следующую часть |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Закоммитить изменения без предварительного хука коммита |  |
| `` g e c `` | Сохранить изменения с помощью редактора git |  |
| `` / `` | Найти |  |

## Главная панель (Обычный)

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Прокрутить вниз |  |
| `` <mouse wheel up> (fn+down) `` | Прокрутить вверх |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Переключить выборку перетаскивания |  |
| `` e `` | Редактировать файл | Open file in external editor. |
| `` <space>, g h s `` | Переключить индекс | Переключить строку в проиндексированные / непроиндексированные |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Добавить/удалить строку(и) для патча |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Отменить изменение (git reset) | When unstaged change is selected, discard the change using `git reset`. When staged change is selected, unstage the change. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Изменить эту часть | Edit selected hunk in external editor. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Выбрать предыдущую часть |  |
| `` ], t n h `` | Выбрать следующую часть |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Закоммитить изменения без предварительного хука коммита |  |
| `` g e c `` | Сохранить изменения с помощью редактора git |  |
| `` / `` | Найти |  |

## Главная панель (Слияние)

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | Выбрать эту часть |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | Выбрать предыдущую часть |  |
| `` j, <down> `` | Выбрать следующую часть |  |
| `` [, t p c `` | Выбрать предыдущий конфликт |  |
| `` ], t n c `` | Выбрать следующий конфликт |  |
| `` u u `` | Отменить | Undo last merge conflict resolution. |
| `` e `` | Редактировать файл | Open file in external editor. |
| `` o `` | Открыть файл | Open file in default application. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | Вернуться к панели файлов |  |

## Журнал ссылок (Reflog)

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Жёсткий сброс |  |
| `` s m `` | Смешанный сброс |  |
| `` s s `` | Мягкий сброс |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Переключить | Checkout the selected commit as a detached HEAD. |
| `` o `` | Открыть коммит в браузере |  |
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

## Коммиты

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | Просмотреть параметры бинарного поиска |  |
| `` <f2> `` | Перефразировать коммит | Reword the selected commit's message. |
| `` g e r `` | Переписать коммит с помощью редактора |  |
| `` <delete> `` | Удалить коммит | Drop the selected commit. This will remove the commit from the branch via a rebase. If the commit makes changes that later commits depend on, you may need to resolve merge conflicts. |
| `` e `` | Edit (start interactive rebase) | Изменить коммит |
| `` g i `` | Start interactive rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` g r p `` | Pick | Выбрать коммит (в середине перебазирования) |
| `` > `` | Переместить коммит вниз на один |  |
| `` < `` | Переместить коммит вверх на один |  |
| `` c p `` | Выборочная отборка (Cherry-Pick) |  |
| `` g r b `` | Mark as base commit for rebase | Select a base commit for the next rebase. When you rebase onto a branch, only commits above the base commit will be brought across. This uses the `git rebase --onto` command. |
| `` a `` | Amend | Править последний коммит с проиндексированными изменениями |
| `` g e a `` | Установить/убрать автора коммита | Set/Reset commit author or set co-author. |
| `` g v c `` | Revert | Create a revert commit for the selected commit, which applies the selected commit's changes in reverse. |
| `` n t `` | Пометить коммит тегом | Create a new tag pointing at the selected commit. You'll be prompted to enter a tag name and optional description. |
| `` t l o `` | Открыть меню журнала | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
| `` g p v `` | Open pull request in browser |  |
| `` w a `` | Show selected commits and their ancestors |  |
| `` w d `` | Show selected commits and their descendants |  |
| `` w b `` | Show branch and its upstream |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Жёсткий сброс |  |
| `` s m `` | Смешанный сброс |  |
| `` s s `` | Мягкий сброс |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Переключить | Checkout the selected commit as a detached HEAD. |
| `` o `` | Открыть коммит в браузере |  |
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
| `` v, <space> `` | Просмотреть файлы выбранного элемента |  |
| `` / `` | Найти |  |

## Локальные Ветки

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Скопировать название ветки в буфер обмена |  |
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
| `` g y `` | Показать параметры git-flow |  |
| `` s h `` | Жёсткий сброс |  |
| `` s m `` | Смешанный сброс |  |
| `` s s `` | Мягкий сброс |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` w b `` | Show branch and its upstream |  |
| `` s c `` | Переключить | Checkout selected item. |
| `` g m n `` | Move commits to new branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | New worktree |  |
| `` g p c `` | Создать запрос на принятие изменений |  |
| `` g p o `` | Создать параметры запроса принятие изменений |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | Скопировать URL запроса на принятие изменений в буфер обмена |  |
| `` <delete> `` | Delete | View delete options for local/remote branch. |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` m `` | Merge selected branch into current… | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | Создать тег |  |
| `` t s o `` | Порядок сортировки |  |
| `` <f2> `` | Переименовать ветку |  |
| `` g t `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Просмотреть коммиты |  |
| `` / `` | Filter the current view by text |  |

## Меню

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Выполнить |  |
| `` <esc> `` | Закрыть/отменить |  |
| `` / `` | Filter the current view by text |  |

## Панель Подтверждения

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Подтвердить |  |
| `` <esc> `` | Закрыть/отменить |  |
| `` <ctrl+o> `` | Copy to clipboard |  |

## Подкоммиты

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | Жёсткий сброс |  |
| `` s m `` | Смешанный сброс |  |
| `` s s `` | Мягкий сброс |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Переключить | Checkout the selected commit as a detached HEAD. |
| `` o `` | Открыть коммит в браузере |  |
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
| `` v, <space> `` | Просмотреть файлы выбранного элемента |  |
| `` / `` | Найти |  |

## Подмодули

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Скопировать название подмодуля в буфер обмена |  |
| `` v, <space> `` | Enter | Ввести подмодуль |
| `` <delete> `` | Remove | Remove the selected submodule and its corresponding directory. |
| `` g u s `` | Update | Обновить подмодуль |
| `` n s `` | Добавить новый подмодуль |  |
| `` e `` | Обновить URL подмодуля |  |
| `` g u i `` | Initialize | Инициализировать подмодуль |
| `` g u b `` | Просмотреть параметры массового подмодуля |  |
| `` t a g `` | Пасхалка |  |
| `` / `` | Filter the current view by text |  |

## Сводка коммита

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Подтвердить |  |
| `` <esc> `` | Закрыть |  |

## Сохранить Изменения Файлов

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Скопировать название файла в буфер обмена |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | Переключить | Переключить файл |
| `` x `` | Просмотреть параметры «отмены изменении» | Отменить изменения коммита в этом файле |
| `` o `` | Открыть файл | Open file in default application. |
| `` e `` | Edit | Open file in external editor. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Переключить файлы включённые в патч | Toggle whether the file is included in the custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Переключить все файлы, включённые в патч | Add/remove all commit's files to custom patch. See https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | Переключить вид дерева файлов | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
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

## Статус

| Key | Action | Info |
|-----|--------|-------------|
| `` s n `` | Open repository by path |  |
| `` t a c `` | Редактировать файл конфигурации | Open file in external editor. |
| `` t a u `` | Проверить обновления |  |
| `` s r `` | Переключиться на последний репозиторий |  |
| `` t l n `` | Show/cycle all branch logs |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space>, 0 `` | Focus main view |  |

## Теги

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
| `` s c `` | Переключить | Checkout the selected tag as a detached HEAD. |
| `` n t `` | Создать тег | Create new tag from current commit. You'll be prompted to enter a tag name and optional description. |
| `` n w `` | New worktree |  |
| `` <delete> `` | Delete | View delete options for local/remote tag. |
| `` p `` | Отправить тег | Push the selected tag to a remote. You'll be prompted to select a remote. |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Просмотреть коммиты |  |
| `` / `` | Filter the current view by text |  |

## Удалённые ветки

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Скопировать название ветки в буфер обмена |  |
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
| `` s c `` | Переключить | Checkout a new local branch based on the selected remote branch, or the remote branch as a detached head. |
| `` n w `` | New worktree |  |
| `` m `` | Слияние с текущей переключённой веткой | View options for merging the selected item into the current branch (regular merge, squash merge) |
| `` r `` | Rebase current onto selected branch… | Rebase the checked-out branch onto the selected branch. |
| `` <delete> `` | Delete | Delete the remote branch from the remote. |
| `` g t `` | Set as upstream | Установить как upstream-ветку переключённую ветку |
| `` t s o `` | Порядок сортировки |  |
| `` t d e `` | Open external diff tool (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Просмотреть коммиты |  |
| `` / `` | Filter the current view by text |  |

## Удалённые репозитории

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | View branches |  |
| `` n r `` | Добавить новую удалённую ветку |  |
| `` <delete> `` | Remove | Remove the selected remote. Any local branches tracking a remote branch from the remote will be unaffected. |
| `` e `` | Edit | Редактировать удалённый репозитории |
| `` f `` | Получить изменения | Получение изменения из удалённого репозитория |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | Filter the current view by text |  |

## Файлы

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Скопировать название файла в буфер обмена |  |
| `` <space> `` | Toggle staging of selected files | Toggle staged for selected file. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` w s `` | Фильтровать файлы (проиндексированные/непроиндексированные) |  |
| `` <enter> `` | Сохранить изменения | Commit staged changes. |
| `` g e n `` | Закоммитить изменения без предварительного хука коммита |  |
| `` a `` | Правка последнего коммита |  |
| `` g e c `` | Сохранить изменения с помощью редактора git |  |
| `` e `` | Edit | Open file in external editor. |
| `` o `` | Открыть файл | Open file in default application. |
| `` i `` | Игнорировать или исключить файл |  |
| `` <f5> `` | Обновить | Refresh the git state (i.e. run `git status`, `git branch`, etc in background to update the contents of panels). This does not run `git fetch`. |
| `` g s s `` | Stash | Stash all changes. For other variations of stashing, use the view stash options keybinding. |
| `` g s o `` | Просмотреть параметры хранилища | View stash options (e.g. stash all, stash staged, stash unstaged). |
| `` x `` | Просмотреть параметры «отмены изменении» | View options for discarding changes to the selected file. |
| `` g z u `` | Просмотреть параметры сброса upstream-ветки |  |
| `` g z w `` | Reset | View reset options for working tree (e.g. nuking the working tree). |
| `` t t t `` | Переключить вид дерева файлов | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
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

## Хранилище

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | Применить припрятанные изменения | Apply the stash entry to your working directory. |
| `` g s p `` | Применить припрятанные изменения и тут же удалить их из хранилища | Apply the stash entry to your working directory and remove the stash entry. |
| `` <delete> `` | Удалить припрятанные изменения из хранилища | Remove the stash entry from the stash list. |
| `` n w `` | New worktree |  |
| `` <f2> `` | Переименовать хранилище |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Просмотреть файлы выбранного элемента |  |
| `` / `` | Filter the current view by text |  |
