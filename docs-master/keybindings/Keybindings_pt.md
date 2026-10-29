_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Atalhos do teclado

## Combinações globais de teclas

| Key | Action | Info |
|-----|--------|-------------|
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | Rolar janela principal para cima |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | Rolar a janela principal para baixo |  |
| `` t l c `` | View command log options | View options for the command log e.g. show/hide the command log and focus the command log. |
| `` p `` | Empurre (Push) | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | Puxar (Pull) | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | Increase diff context size | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | Decrease diff context size | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | cherry-pick |  |
| `` f `` | Buscar | Buscar alterações do controle remoto. |
| `` : `` | Executar comando da shell | Traga um prompt onde você pode digitar um comando shell para executar. |
| `` g h o `` | Ver opções de patch personalizadas |  |
| `` g o `` | Ver opções de mesclar/rebase | Ver opções para abortar/continuar/pular o merge/rebase atual. |
| `` <f5> `` | Atualizar | Atualize o estado do git (ou seja, execute `git status`, `git branch`, etc em segundo plano para atualizar o conteúdo de painéis). Isso não executa `git fetch`. |
| `` =, + `` | Modo de tela seguinte (normal/metade/tela cheia) |  |
| `` - `` | Modo de tela anterior |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Cancelar |  |
| `` ? `` | Abrir o menu de atalhos do teclado |  |
| `` t d c `` | View diffing options | View options relating to diffing two refs e.g. diffing against selected ref, entering ref to diff against, and reversing the diff direction. |
| `` q `` | Sair |  |
| `` t a s `` | Suspender a aplicação |  |
| `` t d w `` | Toggle whitespace | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` w p `` | Filter by path |  |
| `` w u `` | Filter by author |  |
| `` w c `` | Clear the log filter |  |
| `` u u `` | Desfazer | O reflog será usado para determinar qual comando git para executar para desfazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |
| `` u r `` | Refazer | O reflog será usado para determinar qual comando git para executar para refazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |

## List panel navigation

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus Status panel |  |
| `` 2 `` | Focus Arquivos panel |  |
| `` 3 `` | Focus Branches panel |  |
| `` 4 `` | Focus Commits panel |  |
| `` 5 `` | Focus Stash panel |  |
| `` <pgup> `` | Aba anterior |  |
| `` <pgdown> `` | Próxima aba |  |
| `` <home> `` | Voltar ao topo |  |
| `` <end> `` | Ir para o final |  |
| `` t h r `` | Toggle range select |  |
| `` <shift+down>, J `` | Range select down |  |
| `` <shift+up>, K `` | Range select up |  |
| `` / `` | Pesquisar na visualização atual por texto |  |
| `` , `` | Aba anterior |  |
| `` . `` | Próxima aba |  |
| `` <, <home> `` | Voltar ao topo |  |
| `` >, <end> `` | Ir para o final |  |
| `` <alt+left> `` | Rolar à esquerda |  |
| `` <alt+right> `` | Scroll para a direita |  |
| `` <tab> `` | Próxima aba |  |
| `` <backtab> `` | Aba anterior |  |

## Arquivos

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Copiar caminho para área de transferência |  |
| `` <space> `` | Toggle staging of selected files | Alternar para staging para o arquivo selecionado. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` w s `` | Filtrar arquivos por status |  |
| `` <enter> `` | Commit | Submeter mudanças em staging |
| `` g e n `` | Fazer commit de alterações sem pré-commit |  |
| `` a `` | Alterar último commit |  |
| `` g e c `` | Enviar alteração usando um editor Git |  |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` i `` | Ignore or exclude file |  |
| `` <f5> `` | Atualizar | Atualize o estado do git (ou seja, execute `git status`, `git branch`, etc em segundo plano para atualizar o conteúdo de painéis). Isso não executa `git fetch`. |
| `` g s s `` | Stash | Stash todas as alterações. Para outras variações de armazenamento, use a fixação de teclas de armazenamento. |
| `` g s o `` | Ver opções de stash | Ver opções de stash (por exemplo, trash all, stash staged, stash unsttued). |
| `` x `` | Descartar | Exibir opções para descartar alterações para o arquivo selecionado. |
| `` g z u `` | View upstream reset options |  |
| `` g z w `` | Restaurar | Opções de redefinição de exibição para árvore de trabalho (por exemplo, nukando a árvore de trabalho). |
| `` t t t `` | Alternar exibição de árvore de arquivo | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t d e `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` t t c `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` t t e `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
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
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches locais

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copiar nome da branch para área de transferência |  |
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
| `` g y `` | Exibir opções do git-flow |  |
| `` s h `` | Hard reset |  |
| `` s m `` | Mixed reset |  |
| `` s s `` | Soft reset |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` w b `` | Show branch / tag |  |
| `` s c `` | Verificar | Checar item selecionado |
| `` g m n `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nova árvore de trabalho |  |
| `` g p c `` | Criar solicitação de pull |  |
| `` g p o `` | View create pull request options |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | Copiar URL do pull request para área de transferência |  |
| `` <delete> `` | Apagar | Ver opções de exclusão para a branch local/remoto. |
| `` r `` | Rebase current onto selected branch… | Refazer a branch checada na branch selecionada |
| `` m `` | Merge selected branch into current… | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | Nova etiqueta |  |
| `` t s o `` | Sort order |  |
| `` <f2> `` | Renomear branch |  |
| `` g t `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` t d e `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` v, <space> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches remotos

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copiar nome da branch para área de transferência |  |
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
| `` w b `` | Show branch / tag |  |
| `` s c `` | Verificar | Checar a nova branch baseada na brach remota selecionada, ou a branch remota como HEAD, desanexado |
| `` n w `` | Nova árvore de trabalho |  |
| `` m `` | Mesclar | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` r `` | Rebase current onto selected branch… | Refazer a branch checada na branch selecionada |
| `` <delete> `` | Apagar | Excluir o branch remoto do controle remoto. |
| `` g t `` | Definir como upstream | Definir o ramo remoto selecionado como fluxo do branch check-out. |
| `` t s o `` | Sort order |  |
| `` t d e `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` v, <space> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Commit arquivos

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | Copiar caminho para área de transferência |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | Verificar | Arquivo de check-out. Isso substitui o arquivo em sua árvore de trabalho com a versão do commit selecionado. |
| `` x `` | Descartar | Descartar as alterações desse commit para este arquivo. Isso executa uma rebase interativa em segundo plano, então você pode ter um conflito de merge se um commit posterior também alterar este arquivo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | Alternar entre o arquivo incluído no patch | Alternar se o arquivo está incluído no patch personalizado. Veja https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` g h a `` | Alternar todos os arquivos | Adicionar/remover todos os arquivos de commit para atualização personalizada. Consulte https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | Alternar exibição de árvore de arquivo | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` t t c `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` t t e `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Commits

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | Ver opções de bissecção |  |
| `` <f2> `` | Reword | Repetir a mensagem de submissão selecionada. |
| `` g e r `` | Republicar com o editor |  |
| `` <delete> `` | Descartar | Solte o commit selecionado. Isso irá remover o commit do branch através de uma rebase. Se o commit faz com que as alterações em commits posteriores dependem, você pode precisar resolver conflitos de merge. |
| `` e `` | Editar (iniciar rebase interativa) | Editar o commit selecionado. Use isto para iniciar uma rebase interativa a partir do commit selecionado. Quando já estiver no meio da reconstrução, isto irá marcar o commit selecionado para edição, o que significa que ao continuar com a reformulação. a rebase irá pausar no commit selecionado para permitir que você faça alterações. |
| `` g i `` | Start interactive rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` g r p `` | Escolher | Marque o commit selecionado para ser escolhido (quando meados da base). Isso significa que o commit será mantido ao continuar o rebase. |
| `` > `` | Mover commit um para baixo |  |
| `` < `` | Mover o commit um para cima |  |
| `` c p `` | cherry-pick |  |
| `` g r b `` | Mark as base commit for rebase | Select a base commit for the next rebase. When you rebase onto a branch, only commits above the base commit will be brought across. This uses the `git rebase --onto` command. |
| `` a `` | Modificar | Alterar o commit com mudanças em sted. Se o commit selecionado for o commit HEAD, ele executará o `git commit --amend`. Caso contrário, o compromisso será alterado por meio de uma base de apoio. |
| `` g e a `` | Alterar atributo de commit | Definir/Redefinir autor de submissão ou co-autor definido. |
| `` g v c `` | Reverter | Crie um commit reverter para o commit selecionado, que aplica as alterações do commit selecionado em reverso. |
| `` n t `` | Etiquetar commit | Create a new tag pointing at the selected commit. You'll be prompted to enter a tag name and optional description. |
| `` t l o `` | View log options | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
| `` g p v `` | Open pull request in browser |  |
| `` w a `` | Show selected commits and their ancestors |  |
| `` w d `` | Show selected commits and their descendants |  |
| `` w b `` | Show branch / tag |  |
| `` <space> `` | Toggle selection |  |
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
| `` s c `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` o `` | Abrir commit no navegador |  |
| `` g m n `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nova árvore de trabalho |  |
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
| `` 0 `` | Focar visualização principal |  |
| `` v `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Etiquetas

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copiar etiqueta para área de transferência |  |
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
| `` w b `` | Show branch / tag |  |
| `` s c `` | Verificar | Checar a tag selecionada como um HEAD, desanexado |
| `` n t `` | Nova etiqueta | Crie uma nova etiqueta a partir do commit atual. Você será solicitado a digitar um nome e uma descrição opcional. |
| `` n w `` | Nova árvore de trabalho |  |
| `` <delete> `` | Apagar | Ver opções de exclusão para tag local/remoto. |
| `` p `` | Empurrar etiqueta | Push the selected tag to a remote. You'll be prompted to select a remote. |
| `` t d e `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` v, <space> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar/Cancelar |  |

## Menu

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Executar |  |
| `` <esc> `` | Fechar/Cancelar |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Painel Principal (Normal)

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Rolar para baixo |  |
| `` <mouse wheel up> (fn+down) `` | Rolar para cima |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` t h r `` | Toggle range select |  |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space>, g h s `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Alternar linhas no caminho |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` x `` | Remover linhas do commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Ir para o local anterior |  |
| `` ], t n h `` | Ir para o próximo trecho |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Fazer commit de alterações sem pré-commit |  |
| `` g e c `` | Enviar alteração usando um editor Git |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Painel de confirmação

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar/Cancelar |  |
| `` <ctrl+o> `` | Copy to clipboard |  |

## Painel principal (mesclagem)

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | Escolha o local |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | Trecho anterior |  |
| `` j, <down> `` | Próximo trecho |  |
| `` [, t p c `` | Conflito anterior |  |
| `` ], t n c `` | Próximo conflito |  |
| `` u u `` | Desfazer | Desfazer resolução de conflitos de última mesclagem. |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | Retornar ao painel de arquivos |  |

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
| `` s c `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` o `` | Abrir commit no navegador |  |
| `` g m n `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nova árvore de trabalho |  |
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
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Remotes

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | Ver branches |  |
| `` n r `` | Novo controle |  |
| `` <delete> `` | Remover | Remover o controle remoto. Quaisquer ramificações locais de rastreamento de um ramo remoto do controle não serão afetadas. |
| `` e `` | Editar | Edit the selected remote's name or URL. |
| `` f `` | Buscar | Fetch updates from the remote repository. This retrieves new commits and branches without merging them into your local branches. |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | Filtrar a visualização atual por texto |  |

## Secundário

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | Toggle hunk selection | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` t h r `` | Toggle range select |  |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space>, g h s `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | Alternar linhas no caminho |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` x `` | Remover linhas do commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | Ir para o local anterior |  |
| `` ], t n h `` | Ir para o próximo trecho |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` g e n `` | Fazer commit de alterações sem pré-commit |  |
| `` g e c `` | Enviar alteração usando um editor Git |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Stash

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | Aplicar | Aplique o stash no seu diretório de trabalho. |
| `` g s p `` | Pop | Aplique a entrada de stash no seu diretório de trabalho e remova a entrada de stash. |
| `` <delete> `` | Descartar | Remova a entrada do stash da lista de armazenamento. |
| `` n w `` | Nova árvore de trabalho |  |
| `` <f2> `` | Renomear o stash |  |
| `` 0 `` | Focar visualização principal |  |
| `` v, <space> `` | Ver arquivos |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Status

| Key | Action | Info |
|-----|--------|-------------|
| `` s n `` | Open repository by path |  |
| `` t a c `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` t a u `` | Verificar atualização |  |
| `` s r `` | Mudar para um repositório recente |  |
| `` t l n `` | Mostrar/ciclo todos os logs de filiais |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space>, 0 `` | Focar visualização principal |  |

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
| `` s c `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` o `` | Abrir commit no navegador |  |
| `` g m n `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | Nova árvore de trabalho |  |
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
| `` 0 `` | Focar visualização principal |  |
| `` v, <space> `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Submódulos

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | Copiar o nome do submódulo para área de transferência |  |
| `` v, <space> `` | Enter | Enter submodule. After entering the submodule, you can press `<esc>` to escape back to the parent repo. |
| `` <delete> `` | Remover | Remova o submódulo selecionado e o diretório correspondente. |
| `` g u s `` | Atualizar | Atualizar submódulo selecionado. |
| `` n s `` | Novo submódulo |  |
| `` e `` | Atualizar URL do submódulo |  |
| `` g u i `` | Inicializar | Initialize the selected submodule to prepare for fetching. You probably want to follow this up by invoking the 'update' action to fetch the submodule. |
| `` g u b `` | View bulk submodule options |  |
| `` t a g `` | Easter Egg |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Sumário do commit

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar |  |

## Árvores de trabalho

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | Nova árvore de trabalho |  |
| `` s w `` | Switch | Mudar para a árvore de trabalho selecionada. |
| `` o `` | Abrir no editor |  |
| `` <delete> `` | Remover | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` v, <space> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |
