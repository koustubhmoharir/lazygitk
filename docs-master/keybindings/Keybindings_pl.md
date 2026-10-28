_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Skróty klawiszowe

## Globalne skróty klawiszowe

| Key | Action | Info |
|-----|--------|-------------|
| `` s r `` | Przełącz na ostatnie repozytorium |  |
| `` s n `` | Open repository by path |  |
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | Przewiń główne okno w górę |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | Przewiń główne okno w dół |  |
| `` t l c `` | Pokaż opcje dziennika poleceń | Pokaż opcje dla dziennika poleceń, np. pokazywanie/ukrywanie dziennika poleceń i skupienie na dzienniku poleceń. |
| `` p `` | Wypchnij | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | Pociągnij | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Zwiększ rozmiar kontekstu w widoku różnic | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Zmniejsz rozmiar kontekstu w widoku różnic | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | Cherry-pick |  |
| `` f `` | Pobierz | Pobierz zmiany ze zdalnego serwera. |
| `` : `` | Wykonaj polecenie w powłoce | Bring up a prompt where you can enter a shell command to execute. |
| `` g h o `` | Wyświetl opcje niestandardowej łatki |  |
| `` g o `` | Pokaż opcje scalania/rebase | Pokaż opcje do przerwania/kontynuowania/pominięcia bieżącego scalania/rebase. |
| `` <f5> `` | Odśwież | Odśwież stan git (tj. uruchom `git status`, `git branch`, itp. w tle, aby zaktualizować zawartość paneli). To nie uruchamia `git fetch`. |
| `` =, + `` | Następny tryb ekranu (normalny/półpełny/pełnoekranowy) |  |
| `` - `` | Poprzedni tryb ekranu |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Anuluj |  |
| `` ? `` | Otwórz menu przypisań klawiszy |  |
| `` t f o `` | Pokaż opcje filtrowania | Pokaż opcje filtrowania dziennika commitów, tak aby pokazywane były tylko commity pasujące do filtra. |
| `` t d c `` | Pokaż opcje różnicowania | Pokaż opcje dotyczące różnicowania dwóch refów, np. różnicowanie względem wybranego refa, wprowadzanie refa do różnicowania i odwracanie kierunku różnic. |
| `` q `` | Wyjdź |  |
| `` t a s `` | Suspend the application |  |
| `` t d w `` | Przełącz białe znaki | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | Edytuj plik konfiguracyjny | Otwórz plik w zewnętrznym edytorze. |
| `` u u `` | Cofnij | Dziennik reflog zostanie użyty do określenia, jakie polecenie git należy uruchomić, aby cofnąć ostatnie polecenie git. Nie obejmuje to zmian w drzewie roboczym; brane są pod uwagę tylko commity. |
| `` u r `` | Ponów | Dziennik reflog zostanie użyty do określenia, jakie polecenie git należy uruchomić, aby ponowić ostatnie polecenie git. Nie obejmuje to zmian w drzewie roboczym; brane są pod uwagę tylko commity. |

## Nawigacja panelu listy

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus Status panel |  |
| `` 2 `` | Focus Pliki panel |  |
| `` 3 `` | Focus Gałęzie panel |  |
| `` 4 `` | Focus Commity panel |  |
| `` 5 `` | Focus Schowek panel |  |
| `` <pgup> `` | Poprzednia strona |  |
| `` <pgdown> `` | Następna strona |  |
| `` <home> `` | Przewiń do góry |  |
| `` <end> `` | Przewiń do dołu |  |
| `` t h r `` | Przełącz zaznaczenie zakresu |  |
| `` <shift+down> `` | Zaznacz zakres w dół |  |
| `` <shift+up> `` | Zaznacz zakres w górę |  |
| `` / `` | Szukaj w bieżącym widoku po tekście |  |
| `` , `` | Poprzednia strona |  |
| `` . `` | Następna strona |  |
| `` <, <home> `` | Przewiń do góry |  |
| `` >, <end> `` | Przewiń do dołu |  |
| `` <alt+left> `` | Przewiń w lewo |  |
| `` <alt+right> `` | Przewiń w prawo |  |
| `` <tab> `` | Następna zakładka |  |
| `` <backtab> `` | Poprzednia zakładka |  |

## Commity

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | Zobacz opcje bisect |  |
| `` <f2> `` | Przeformułuj | Przeformułuj wiadomość wybranego commita. |
| `` g e r `` | Przeformułuj za pomocą edytora |  |
| `` <delete> `` | Usuń | Usuń wybrany commit. To usunie commit z gałęzi za pomocą przebazowania. Jeśli commit wprowadza zmiany, od których zależą późniejsze commity, być może będziesz musiał rozwiązać konflikty scalania. |
| `` e `` | Edytuj (rozpocznij interaktywne przebazowanie) | Edytuj wybrany commit. Użyj tego, aby rozpocząć interaktywne przebazowanie od wybranego commita. Podczas trwania przebazowania, to oznaczy wybrany commit do edycji, co oznacza, że po kontynuacji przebazowania, przebazowanie zostanie wstrzymane na wybranym commicie, aby umożliwić wprowadzenie zmian. |
| `` g i `` | Rozpocznij interaktywne przebazowanie | Rozpocznij interaktywne przebazowanie dla commitów na twojej gałęzi. To będzie zawierać wszystkie commity od HEAD do pierwszego commita scalenia lub commita głównej gałęzi.<br>Jeśli zamiast tego chcesz rozpocząć interaktywne przebazowanie od wybranego commita, naciśnij `e`. |
| `` g r p `` | Wybierz | Oznacz wybrany commit do wybrania (podczas przebazowania). Oznacza to, że commit zostanie zachowany po kontynuacji przebazowania. |
| `` > `` | Przesuń commit w dół |  |
| `` < `` | Przesuń commit w górę |  |
| `` c p `` | Cherry-pick |  |
| `` g r b `` | Oznacz jako bazowy commit dla przebazowania | Wybierz bazowy commit dla następnego przebazowania. Kiedy robisz przebazowanie na gałąź, tylko commity powyżej bazowego commita zostaną przeniesione. Używa to polecenia `git rebase --onto`. |
| `` a `` | Popraw | Popraw commit ze zmianami zatwierdzonymi. Jeśli wybrany commit jest commit HEAD, to wykona `git commit --amend`. W przeciwnym razie commit zostanie poprawiony za pomocą przebazowania. |
| `` g e a `` | Popraw atrybut commita | Ustaw/Resetuj autora commita lub ustaw współautora. |
| `` g v c `` | Cofnij | Utwórz commit cofający dla wybranego commita, który stosuje zmiany wybranego commita w odwrotnej kolejności. |
| `` n t `` | Otaguj commit | Utwórz nowy tag wskazujący na wybrany commit. Zostaniesz poproszony o wprowadzenie nazwy tagu i opcjonalnego opisu. |
| `` t l o `` | Zobacz opcje logów | Zobacz opcje dla logów commitów, np. zmiana kolejności sortowania, ukrywanie grafu gita, pokazywanie całego grafu gita. |
| `` g p v `` | Otwórz żądanie ściągnięcia w przeglądarce |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Twardy reset |  |
| `` s m `` | Mieszany reset |  |
| `` s s `` | Miękki reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Przełącz | Przełącz wybrany commit jako odłączoną HEAD. |
| `` o `` | Otwórz commit w przeglądarce |  |
| `` g m n `` | Przenieś commity do nowej gałęzi | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nowe drzewo pracy |  |
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
| `` v, <space> `` | Wyświetl pliki |  |
| `` / `` | Szukaj w bieżącym widoku po tekście |  |

## Dodatkowy

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Przełącz zaznaczenie zakresu |  |
| `` e `` | Edytuj plik | Otwórz plik w zewnętrznym edytorze. |
| `` <space>, g h s `` | Zatwierdź | Przełącz zaznaczenie zatwierdzone/niezatwierdzone. |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Przełącz linie w łatce |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Odrzuć | Gdy zaznaczona jest niezatwierdzona zmiana, odrzuć ją używając `git reset`. Gdy zaznaczona jest zatwierdzona zmiana, cofnij zatwierdzenie. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edytuj fragment | Edytuj wybrany fragment w zewnętrznym edytorze. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Idź do poprzedniego fragmentu |  |
| `` ], t n h `` | Idź do następnego fragmentu |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Zatwierdź zmiany bez hooka pre-commit |  |
| `` g e c `` | Zatwierdź zmiany używając edytora git |  |
| `` / `` | Szukaj w bieżącym widoku po tekście |  |

## Drzewa pracy

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | Nowe drzewo pracy |  |
| `` s w `` | Przełącz | Przełącz do wybranego drzewa pracy. |
| `` o `` | Otwórz w edytorze |  |
| `` <delete> `` | Usuń | Usuń wybrane drzewo pracy. To usunie zarówno katalog drzewa pracy, jak i metadane o drzewie pracy w katalogu .git. |
| `` v, <space> `` | Pokaż commity |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Dziennik reflog

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Twardy reset |  |
| `` s m `` | Mieszany reset |  |
| `` s s `` | Miękki reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Przełącz | Przełącz wybrany commit jako odłączoną HEAD. |
| `` o `` | Otwórz commit w przeglądarce |  |
| `` g m n `` | Przenieś commity do nowej gałęzi | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nowe drzewo pracy |  |
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
| `` / `` | Filtruj bieżący widok po tekście |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Potwierdź |  |
| `` <esc> `` | Zamknij/Anuluj |  |

## Lokalne gałęzie

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopiuj nazwę gałęzi do schowka |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` g y `` | Pokaż opcje git-flow |  |
| `` s h `` | Twardy reset |  |
| `` s m `` | Mieszany reset |  |
| `` s s `` | Miękki reset |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | Przełącz | Przełącz wybrany element. |
| `` g m n `` | Przenieś commity do nowej gałęzi | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nowe drzewo pracy |  |
| `` g p c `` | Utwórz żądanie ściągnięcia |  |
| `` g p o `` | Zobacz opcje tworzenia pull requesta |  |
| `` g p v `` | Otwórz żądanie ściągnięcia w przeglądarce |  |
| `` y r `` | Kopiuj adres URL żądania ściągnięcia do schowka |  |
| `` s b `` | Przełącz według nazwy | Przełącz według nazwy. W polu wprowadzania możesz wpisać '-' aby przełączyć się na ostatnią gałąź. |
| `` <delete> `` | Usuń | Wyświetl opcje usuwania lokalnej/odległej gałęzi. |
| `` r `` | Rebase current onto selected branch… | Przebazuj przełączoną gałąź na wybraną gałąź. |
| `` m `` | Merge selected branch into current… | Scal wybraną gałąź z aktualnie sprawdzoną gałęzią. |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | Nowy tag |  |
| `` t s o `` | Kolejność sortowania |  |
| `` <f2> `` | Zmień nazwę gałęzi |  |
| `` g t `` | Pokaż opcje upstream | Pokaż opcje dotyczące upstream gałęzi, np. ustawianie/usuwanie upstream i resetowanie do upstream. |
| `` t d e `` | Otwórz zewnętrzne narzędzie różnic (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Pokaż commity |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Menu

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Wykonaj |  |
| `` <esc> `` | Zamknij/Anuluj |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Panel główny (normalny)

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Przewiń w dół |  |
| `` <mouse wheel up> (fn+down) `` | Przewiń w górę |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | Przełącz zaznaczenie zakresu |  |
| `` e `` | Edytuj plik | Otwórz plik w zewnętrznym edytorze. |
| `` <space>, g h s `` | Zatwierdź | Przełącz zaznaczenie zatwierdzone/niezatwierdzone. |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Przełącz linie w łatce |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Odrzuć | Gdy zaznaczona jest niezatwierdzona zmiana, odrzuć ją używając `git reset`. Gdy zaznaczona jest zatwierdzona zmiana, cofnij zatwierdzenie. |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Edytuj fragment | Edytuj wybrany fragment w zewnętrznym edytorze. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Idź do poprzedniego fragmentu |  |
| `` ], t n h `` | Idź do następnego fragmentu |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Zatwierdź zmiany bez hooka pre-commit |  |
| `` g e c `` | Zatwierdź zmiany używając edytora git |  |
| `` / `` | Szukaj w bieżącym widoku po tekście |  |

## Panel główny (scalanie)

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | Wybierz fragment |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | Poprzedni fragment |  |
| `` j, <down> `` | Następny fragment |  |
| `` [, t p c `` | Poprzedni konflikt |  |
| `` ], t n c `` | Następny konflikt |  |
| `` u u `` | Cofnij | Cofnij ostatnie rozwiązanie konfliktu scalania. |
| `` e `` | Edytuj plik | Otwórz plik w zewnętrznym edytorze. |
| `` o `` | Otwórz plik | Otwórz plik w domyślnej aplikacji. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | Wróć do panelu plików |  |

## Panel potwierdzenia

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Potwierdź |  |
| `` <esc> `` | Zamknij/Anuluj |  |
| `` <ctrl+o> `` | Kopiuj do schowka |  |

## Pliki

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Kopiuj ścieżkę do schowka |  |
| `` <space> `` | Toggle staging of selected files | Przełącz zatwierdzenie dla wybranego pliku. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` t f s `` | Filtruj pliki według statusu |  |
| `` <enter> `` | Commit | Zatwierdź zmiany zatwierdzone. |
| `` g e n `` | Zatwierdź zmiany bez hooka pre-commit |  |
| `` a `` | Popraw ostatni commit |  |
| `` g e c `` | Zatwierdź zmiany używając edytora git |  |
| `` e `` | Edytuj | Otwórz plik w zewnętrznym edytorze. |
| `` o `` | Otwórz plik | Otwórz plik w domyślnej aplikacji. |
| `` i `` | Ignoruj lub wyklucz plik |  |
| `` <f5> `` | Odśwież | Odśwież stan git (tj. uruchom `git status`, `git branch`, itp. w tle, aby zaktualizować zawartość paneli). To nie uruchamia `git fetch`. |
| `` g s s `` | Schowaj | Schowaj wszystkie zmiany. Dla innych wariantów schowania, użyj klawisza wyświetlania opcji schowka. |
| `` g s o `` | Wyświetl opcje schowka | Wyświetl opcje schowka (np. schowaj wszystko, schowaj zatwierdzone, schowaj niezatwierdzone). |
| `` x `` | Odrzuć | Wyświetl opcje odrzucania zmian w wybranym pliku. |
| `` g z u `` | Pokaż opcje resetowania do upstream |  |
| `` g z w `` | Reset | Wyświetl opcje resetu dla drzewa roboczego (np. zniszczenie drzewa roboczego). |
| `` t t t `` | Przełącz widok drzewa plików | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t d e `` | Otwórz zewnętrzne narzędzie różnic (git difftool) |  |
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
| `` / `` | Filtruj bieżący widok po tekście |  |

## Pliki commita

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Kopiuj ścieżkę do schowka |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | Przełącz | Przełącz plik. Zastępuje plik w twoim drzewie roboczym wersją z wybranego commita. |
| `` x `` | Odrzuć | Odrzuć zmiany w tym pliku z tego commita. Uruchamia interaktywne przebazowanie w tle, więc możesz otrzymać konflikt scalania, jeśli późniejszy commit również zmienia ten plik. |
| `` o `` | Otwórz plik | Otwórz plik w domyślnej aplikacji. |
| `` e `` | Edytuj | Otwórz plik w zewnętrznym edytorze. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Przełącz plik włączony w łatkę | Przełącz, czy plik jest włączony w niestandardową łatkę. Zobacz https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Przełącz wszystkie pliki | Dodaj/usuń wszystkie pliki commita do niestandardowej łatki. Zobacz https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | Przełącz widok drzewa plików | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t t c `` | Collapse all files | Collapse all directories in the files tree |
| `` t t e `` | Expand all files | Expand all directories in the file tree |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` 0 `` | Focus main view |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Podsumowanie commita

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Potwierdź |  |
| `` <esc> `` | Zamknij |  |

## Schowek

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | Zastosuj | Zastosuj wpis schowka do katalogu roboczego. |
| `` g s p `` | Wyciągnij | Zastosuj wpis schowka do katalogu roboczego i usuń wpis schowka. |
| `` <delete> `` | Usuń | Usuń wpis schowka z listy schowka. |
| `` n w `` | Nowe drzewo pracy |  |
| `` <f2> `` | Zmień nazwę schowka |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Wyświetl pliki |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Status

| Key | Action | Info |
|-----|--------|-------------|
| `` t a c `` | Edytuj plik konfiguracyjny | Otwórz plik w zewnętrznym edytorze. |
| `` t a u `` | Sprawdź aktualizacje |  |
| `` s r `` | Przełącz na ostatnie repozytorium |  |
| `` t l n `` | Show/cycle all branch logs |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space>, 0 `` | Focus main view |  |

## Sub-commity

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | Twardy reset |  |
| `` s m `` | Mieszany reset |  |
| `` s s `` | Miękki reset |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | Przełącz | Przełącz wybrany commit jako odłączoną HEAD. |
| `` o `` | Otwórz commit w przeglądarce |  |
| `` g m n `` | Przenieś commity do nowej gałęzi | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nowe drzewo pracy |  |
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
| `` v, <space> `` | Wyświetl pliki |  |
| `` / `` | Szukaj w bieżącym widoku po tekście |  |

## Submoduły

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopiuj nazwę submodułu do schowka |  |
| `` v, <space> `` | Wejdź | Wejdź do submodułu. Po wejściu do submodułu możesz nacisnąć `<esc>`, aby wrócić do repozytorium nadrzędnego. |
| `` <delete> `` | Usuń | Usuń wybrany submoduł i odpowiadający mu katalog. |
| `` g u s `` | Aktualizuj | Aktualizuj wybrany submoduł. |
| `` n s `` | Nowy submoduł |  |
| `` e `` | Zaktualizuj URL submodułu |  |
| `` g u i `` | Zainicjuj | Zainicjuj wybrany submoduł, aby przygotować do pobrania. Prawdopodobnie chcesz to kontynuować, wywołując akcję 'update', aby pobrać submoduł. |
| `` g u b `` | Pokaż opcje masowych operacji na submodułach |  |
| `` t a g `` | Jajko wielkanocne |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Tagi

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Skopiuj tag do schowka |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` s c `` | Przełącz | Przełącz wybrany tag jako odłączoną głowę (detached HEAD). |
| `` n t `` | Nowy tag | Utwórz nowy tag z bieżącego commita. Zostaniesz poproszony o wprowadzenie nazwy tagu i opcjonalnego opisu. |
| `` n w `` | Nowe drzewo pracy |  |
| `` <delete> `` | Usuń | Wyświetl opcje usuwania lokalnego/odległego tagu. |
| `` p `` | Wyślij tag | Wyślij wybrany tag do zdalnego. Zostaniesz poproszony o wybranie zdalnego. |
| `` t d e `` | Otwórz zewnętrzne narzędzie różnic (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Pokaż commity |  |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Zdalne

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | Wyświetl gałęzie |  |
| `` n r `` | Nowy zdalny |  |
| `` <delete> `` | Usuń | Usuń wybrany zdalny. Wszelkie lokalne gałęzie śledzące gałąź zdalną z tego zdalnego nie zostaną dotknięte. |
| `` e `` | Edytuj | Edytuj nazwę lub URL wybranego zdalnego. |
| `` f `` | Pobierz | Pobierz aktualizacje z zdalnego repozytorium. Pobiera nowe commity i gałęzie bez scalania ich z lokalnymi gałęziami. |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | Filtruj bieżący widok po tekście |  |

## Zdalne gałęzie

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Kopiuj nazwę gałęzi do schowka |  |
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
| `` s c `` | Przełącz | Przełącz na nową lokalną gałąź na podstawie wybranej gałęzi zdalnej. Nowa gałąź będzie śledzić gałąź zdalną. |
| `` n w `` | Nowe drzewo pracy |  |
| `` m `` | Scal | Scal wybraną gałąź z aktualnie sprawdzoną gałęzią. |
| `` r `` | Rebase current onto selected branch… | Przebazuj przełączoną gałąź na wybraną gałąź. |
| `` <delete> `` | Usuń | Usuń gałąź zdalną ze zdalnego. |
| `` g t `` | Ustaw jako upstream | Ustaw wybraną gałąź zdalną jako upstream sprawdzonej gałęzi. |
| `` t s o `` | Kolejność sortowania |  |
| `` t d e `` | Otwórz zewnętrzne narzędzie różnic (git difftool) |  |
| `` 0 `` | Focus main view |  |
| `` v, <space> `` | Pokaż commity |  |
| `` / `` | Filtruj bieżący widok po tekście |  |
