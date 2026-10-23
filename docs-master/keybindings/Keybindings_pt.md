_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Atalhos do teclado

## Combinações globais de teclas

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+r> `` | Mudar para um repositório recente |  |
| `` <pgup>, K, <ctrl+u> (fn+up/shift+k) `` | Rolar janela principal para cima |  |
| `` <pgdown>, J, <ctrl+d> (fn+down/shift+j) `` | Rolar a janela principal para baixo |  |
| `` @ `` | View command log options | View options for the command log e.g. show/hide the command log and focus the command log. |
| `` P `` | Empurre (Push) | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` p `` | Puxar (Pull) | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` ) `` | Increase rename similarity threshold | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` ( `` | Decrease rename similarity threshold | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` } `` | Increase diff context size | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` { `` | Decrease diff context size | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` : `` | Executar comando da shell | Traga um prompt onde você pode digitar um comando shell para executar. |
| `` <ctrl+p> `` | Ver opções de patch personalizadas |  |
| `` m `` | Ver opções de mesclar/rebase | Ver opções para abortar/continuar/pular o merge/rebase atual. |
| `` R `` | Atualizar | Atualize o estado do git (ou seja, execute `git status`, `git branch`, etc em segundo plano para atualizar o conteúdo de painéis). Isso não executa `git fetch`. |
| `` + `` | Modo de tela seguinte (normal/metade/tela cheia) |  |
| `` _ `` | Modo de tela anterior |  |
| `` \| `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` \ `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` <ctrl+g> `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | Cancelar |  |
| `` ? `` | Abrir o menu de atalhos do teclado |  |
| `` <ctrl+s> `` | Ver opções de filtro | View options for filtering the commit log, so that only commits matching the filter are shown. |
| `` q, <ctrl+c> `` | Sair |  |
| `` <ctrl+z> `` | Suspender a aplicação |  |
| `` <ctrl+w> `` | Toggle whitespace | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` <alt+shift+c> `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` z `` | Desfazer | O reflog será usado para determinar qual comando git para executar para desfazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |
| `` Z `` | Refazer | O reflog será usado para determinar qual comando git para executar para refazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |

## List panel navigation

| Key | Action | Info |
|-----|--------|-------------|
| `` , `` | Aba anterior |  |
| `` . `` | Próxima aba |  |
| `` <, <home> `` | Voltar ao topo |  |
| `` >, <end> `` | Ir para o final |  |
| `` <shift+down> `` | Range select down |  |
| `` <shift+up> `` | Range select up |  |
| `` / `` | Pesquisar na visualização atual por texto |  |
| `` H `` | Rolar à esquerda |  |
| `` L `` | Scroll para a direita |  |
| `` ] `` | Próxima aba |  |
| `` [ `` | Aba anterior |  |

## Arquivos

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar caminho para área de transferência |  |
| `` <space> `` | Etapa | Alternar para staging para o arquivo selecionado. |
| `` <ctrl+b> `` | Filtrar arquivos por status |  |
| `` y `` | Copy to clipboard |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` A `` | Alterar último commit |  |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` i `` | Ignore or exclude file |  |
| `` r `` | Atualizar arquivos |  |
| `` s `` | Stash | Stash todas as alterações. Para outras variações de armazenamento, use a fixação de teclas de armazenamento. |
| `` S `` | Ver opções de stash | Ver opções de stash (por exemplo, trash all, stash staged, stash unsttued). |
| `` a `` | Stage completo | Alternar para todos os arquivos na árvore de trabalho |
| `` <enter> `` | Focus file diff / Collapse directory | If the selected item is a file, focus its diff so you can act on individual hunks or lines. If it is a directory, collapse or expand it. |
| `` d `` | Descartar | Exibir opções para descartar alterações para o arquivo selecionado. |
| `` g `` | View upstream reset options |  |
| `` D `` | Restaurar | Opções de redefinição de exibição para árvore de trabalho (por exemplo, nukando a árvore de trabalho). |
| `` ` `` | Alternar exibição de árvore de arquivo | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` M `` | View merge conflict options | View options for resolving merge conflicts. |
| `` f `` | Buscar | Buscar alterações do controle remoto. |
| `` - `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` = `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches locais

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar nome da branch para área de transferência |  |
| `` i `` | Exibir opções do git-flow |  |
| `` <space> `` | Verificar | Checar item selecionado |
| `` n `` | Nova branch |  |
| `` N `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` o `` | Criar solicitação de pull |  |
| `` O `` | View create pull request options |  |
| `` G `` | Open pull request in browser |  |
| `` <ctrl+y> `` | Copiar URL do pull request para área de transferência |  |
| `` c `` | Checar por nome | Checar por nome. Na caixa de entrada você pode inserir '-' para trocar para a última branch  |
| `` - `` | Checkout da branch anterior |  |
| `` F `` | Forçar checagem | Forçar checagem da branch selecionada. Isso irá descartar todas as mudanças no seu diretório de trabalho antes cheque a branch selecionada   |
| `` d `` | Apagar | Ver opções de exclusão para a branch local/remoto. |
| `` r `` | Rebase current onto selected branch… | Refazer a branch checada na branch selecionada |
| `` M `` | Merge selected branch into current… | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` f `` | Fast-forward selected branches… | Review an upstream fetch and a forward-only update of the captured selected branches. Divergent or rewritten history is never reset; tips occupied by another worktree are not moved. A selected checked-out branch uses native fast-forward integration with explicit review. Multiple selections require final batch approval. Saved upstream settings stay unchanged. |
| `` T `` | Nova etiqueta |  |
| `` s `` | Sort order |  |
| `` g `` | Reset checked-out branch to selected ref… |  |
| `` R `` | Renomear branch |  |
| `` u `` | View upstream options | View options relating to the branch's upstream e.g. setting/unsetting the upstream and resetting to the upstream. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches remotos

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar nome da branch para área de transferência |  |
| `` <space> `` | Verificar | Checar a nova branch baseada na brach remota selecionada, ou a branch remota como HEAD, desanexado |
| `` n `` | Nova branch |  |
| `` M `` | Mesclar | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` r `` | Refazer | Refazer a branch checada na branch selecionada |
| `` d `` | Apagar | Excluir o branch remoto do controle remoto. |
| `` u `` | Definir como upstream | Definir o ramo remoto selecionado como fluxo do branch check-out. |
| `` s `` | Sort order |  |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Commit arquivos

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar caminho para área de transferência |  |
| `` w, W `` | Diff with working tree |  |
| `` v, V `` | Diff with previous commit |  |
| `` y `` | Copy to clipboard |  |
| `` c `` | Verificar | Arquivo de check-out. Isso substitui o arquivo em sua árvore de trabalho com a versão do commit selecionado. |
| `` d `` | Descartar | Descartar as alterações desse commit para este arquivo. Isso executa uma rebase interativa em segundo plano, então você pode ter um conflito de merge se um commit posterior também alterar este arquivo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` <ctrl+t> `` | Diff using current comparison mode |  |
| `` <space> `` | Alternar entre o arquivo incluído no patch | Alternar se o arquivo está incluído no patch personalizado. Veja https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` a `` | Alternar todos os arquivos | Adicionar/remover todos os arquivos de commit para atualização personalizada. Consulte https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` <enter> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` ` `` | Alternar exibição de árvore de arquivo | Toggle file view between flat and tree layout. Flat layout shows all file paths in a single list, tree layout groups files by directory.<br><br>The default can be changed in the config file with the key 'gui.showFileTree'. |
| `` - `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` = `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Commits

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copy abbreviated commit hash to clipboard |  |
| `` <ctrl+r> `` | Reset copied (cherry-picked) commits selection |  |
| `` b `` | Ver opções de bissecção |  |
| `` s `` | Squash | Squash o commit selecionado no commit abaixo dele. A mensagem do commit selecionado será anexada ao commit abaixo dele. |
| `` f `` | Corrigir | Faça o commit selecionado no commit abaixo dele. Semelhante para o squash, mas a mensagem do commit selecionado será descartada. |
| `` c `` | Configurar mensagem de correção | Defina a opção de mensagem para o commit de correção. A opção -C significa usar a mensagem deste commit em vez da mensagem do commit alvo. |
| `` r `` | Reword | Repetir a mensagem de submissão selecionada. |
| `` R `` | Republicar com o editor |  |
| `` d `` | Descartar | Solte o commit selecionado. Isso irá remover o commit do branch através de uma rebase. Se o commit faz com que as alterações em commits posteriores dependem, você pode precisar resolver conflitos de merge. |
| `` e `` | Editar (iniciar rebase interativa) | Editar o commit selecionado. Use isto para iniciar uma rebase interativa a partir do commit selecionado. Quando já estiver no meio da reconstrução, isto irá marcar o commit selecionado para edição, o que significa que ao continuar com a reformulação. a rebase irá pausar no commit selecionado para permitir que você faça alterações. |
| `` i `` | Start interactive rebase | Start an interactive rebase for the commits on your branch. This will include all commits from the HEAD commit down to the first merge commit or main branch commit.<br>If you would instead like to start an interactive rebase from the selected commit, press `e`. |
| `` p `` | Escolher | Marque o commit selecionado para ser escolhido (quando meados da base). Isso significa que o commit será mantido ao continuar o rebase. |
| `` F `` | Criar commit de correção | Crie o commit 'correção!' para o commit selecionado. Mais tarde, você pode pressionar `S` neste mesmo commit para aplicar todas os commits de correção acima. |
| `` S `` | Aplicar commits de correções | Aplicar Squash all 'correção!', seja acima do commit selecionado, ou tudo no branch atual (autosquash). |
| `` <ctrl+j>, <alt+down> `` | Mover commit um para baixo |  |
| `` <ctrl+k>, <alt+up> `` | Mover o commit um para cima |  |
| `` B `` | Mark as base commit for rebase | Select a base commit for the next rebase. When you rebase onto a branch, only commits above the base commit will be brought across. This uses the `git rebase --onto` command. |
| `` A `` | Modificar | Alterar o commit com mudanças em sted. Se o commit selecionado for o commit HEAD, ele executará o `git commit --amend`. Caso contrário, o compromisso será alterado por meio de uma base de apoio. |
| `` a `` | Alterar atributo de commit | Definir/Redefinir autor de submissão ou co-autor definido. |
| `` t `` | Reverter | Crie um commit reverter para o commit selecionado, que aplica as alterações do commit selecionado em reverso. |
| `` T `` | Etiquetar commit | Create a new tag pointing at the selected commit. You'll be prompted to enter a tag name and optional description. |
| `` <ctrl+l> `` | View log options | View options for commit log e.g. changing sort order, hiding the git graph, showing the whole git graph. |
| `` G `` | Open pull request in browser |  |
| `` w, W `` | Diff with working tree |  |
| `` v, V `` | Diff with previous commit |  |
| `` <space> `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` y `` | Copy commit attribute to clipboard | Copy commit attribute to clipboard (e.g. hash, URL, diff, message, author). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Create branch and check it out… |  |
| `` N `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Mark commit as copied. Then, in the local commits view, choose Paste (cherry-pick) from the command palette. Use `<esc>` to cancel the selection. |
| `` <ctrl+t> `` | Diff using current comparison mode |  |
| `` * `` | Select commits of current branch |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Etiquetas

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar etiqueta para área de transferência |  |
| `` <space> `` | Verificar | Checar a tag selecionada como um HEAD, desanexado |
| `` n `` | Nova etiqueta | Crie uma nova etiqueta a partir do commit atual. Você será solicitado a digitar um nome e uma descrição opcional. |
| `` d `` | Apagar | Ver opções de exclusão para tag local/remoto. |
| `` P `` | Empurrar etiqueta | Push the selected tag to a remote. You'll be prompted to select a remote. |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
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
| `` <tab> `` | Switch diff pane | Switch to the other focused diff pane. |
| `` a `` | Toggle hunk selection | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space> `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` d `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` E `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` <ctrl+o> `` | Copy selected diff lines to clipboard |  |
| `` <left>, h `` | Ir para o local anterior |  |
| `` <right>, l `` | Ir para o próximo trecho |  |
| `` N `` | Go to previous file |  |
| `` n `` | Go to next file |  |
| `` <ctrl+g> `` | Jump to file |  |
| `` G `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
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
| `` <space> `` | Escolha o local |  |
| `` b `` | Pick both hunks |  |
| `` <up>, k `` | Trecho anterior |  |
| `` <down>, j `` | Próximo trecho |  |
| `` <left>, h `` | Conflito anterior |  |
| `` <right>, l `` | Próximo conflito |  |
| `` z `` | Desfazer | Desfazer resolução de conflitos de última mesclagem. |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` M `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | Retornar ao painel de arquivos |  |

## Reflog

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copy abbreviated commit hash to clipboard |  |
| `` w, W `` | Diff with working tree |  |
| `` v, V `` | Diff with previous commit |  |
| `` <space> `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` y `` | Copy commit attribute to clipboard | Copy commit attribute to clipboard (e.g. hash, URL, diff, message, author). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Create branch and check it out… |  |
| `` N `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Mark commit as copied. Then, in the local commits view, choose Paste (cherry-pick) from the command palette. Use `<esc>` to cancel the selection. |
| `` <ctrl+r> `` | Reset copied (cherry-picked) commits selection |  |
| `` <ctrl+t> `` | Diff using current comparison mode |  |
| `` * `` | Select commits of current branch |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Remotes

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Ver branches |  |
| `` n `` | Novo controle |  |
| `` d `` | Remover | Remover o controle remoto. Quaisquer ramificações locais de rastreamento de um ramo remoto do controle não serão afetadas. |
| `` e `` | Editar | Edit the selected remote's name or URL. |
| `` f `` | Buscar | Fetch updates from the remote repository. This retrieves new commits and branches without merging them into your local branches. |
| `` F `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | Filtrar a visualização atual por texto |  |

## Secundário

| Key | Action | Info |
|-----|--------|-------------|
| `` <tab> `` | Switch diff pane | Switch to the other focused diff pane. |
| `` a `` | Toggle hunk selection | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space> `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` d `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` E `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` <ctrl+o> `` | Copy selected diff lines to clipboard |  |
| `` <left>, h `` | Ir para o local anterior |  |
| `` <right>, l `` | Ir para o próximo trecho |  |
| `` N `` | Go to previous file |  |
| `` n `` | Go to next file |  |
| `` <ctrl+g> `` | Jump to file |  |
| `` G `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | Exit back to side panel |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Stash

| Key | Action | Info |
|-----|--------|-------------|
| `` <space> `` | Aplicar | Aplique o stash no seu diretório de trabalho. |
| `` g `` | Pop | Aplique a entrada de stash no seu diretório de trabalho e remova a entrada de stash. |
| `` d `` | Descartar | Remova a entrada do stash da lista de armazenamento. |
| `` n `` | Nova branch | Criar um novo ramo a partir da entrada de lixo selecionada. Isso funciona verificando o commit do qual a entrada de lixo foi criada, criar um novo branch a partir desse commit e, em seguida, aplicar a entrada de lixo ao novo branch como um commit adicional. |
| `` r `` | Renomear o stash |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Status

| Key | Action | Info |
|-----|--------|-------------|
| `` e `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` u `` | Verificar atualização |  |
| `` <enter> `` | Mudar para um repositório recente |  |
| `` a `` | Mostrar/ciclo todos os logs de filiais |  |
| `` A `` | Show/cycle all branch logs (reverse) |  |
| `` 0 `` | Focar visualização principal |  |

## Sub-commits

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copy abbreviated commit hash to clipboard |  |
| `` w, W `` | Diff with working tree |  |
| `` v, V `` | Diff with previous commit |  |
| `` <space> `` | Verificar | Checkout the selected commit as a detached HEAD. |
| `` y `` | Copy commit attribute to clipboard | Copy commit attribute to clipboard (e.g. hash, URL, diff, message, author). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Create branch and check it out… |  |
| `` N `` | Mover commits para uma nova branch | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Mark commit as copied. Then, in the local commits view, choose Paste (cherry-pick) from the command palette. Use `<esc>` to cancel the selection. |
| `` <ctrl+r> `` | Reset copied (cherry-picked) commits selection |  |
| `` <ctrl+t> `` | Diff using current comparison mode |  |
| `` * `` | Select commits of current branch |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Submódulos

| Key | Action | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar o nome do submódulo para área de transferência |  |
| `` <enter> `` | Enter | Enter submodule. After entering the submodule, you can press `<esc>` to escape back to the parent repo. |
| `` d `` | Remover | Remova o submódulo selecionado e o diretório correspondente. |
| `` u `` | Atualizar | Atualizar submódulo selecionado. |
| `` n `` | Novo submódulo |  |
| `` e `` | Atualizar URL do submódulo |  |
| `` i `` | Inicializar | Initialize the selected submodule to prepare for fetching. You probably want to follow this up by invoking the 'update' action to fetch the submodule. |
| `` b `` | View bulk submodule options |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Sumário do commit

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar |  |

## Árvores de trabalho

| Key | Action | Info |
|-----|--------|-------------|
| `` n `` | Nova árvore de trabalho |  |
| `` <space> `` | Switch | Mudar para a árvore de trabalho selecionada. |
| `` o `` | Abrir no editor |  |
| `` d `` | Remover | Remove the selected worktree. This will both delete the worktree's directory, as well as metadata about the worktree in the .git directory. |
| `` / `` | Filtrar a visualização atual por texto |  |
