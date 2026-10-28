_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit キーバインディング

## グローバルキーバインド

| Key | Action | Info |
|-----|--------|-------------|
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | メインウィンドウを上にスクロール |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | メインウィンドウを下にスクロール |  |
| `` t l c `` | コマンドログオプションを表示 | コマンドログのオプションを表示します（例：コマンドログの表示/非表示、コマンドログへのフォーカスなど）。 |
| `` p `` | プッシュ | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | プル | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | リネーム検出の類似度しきい値を上げる | Increase the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d < `` | リネーム検出の類似度しきい値を下げる | Decrease the similarity threshold for a deletion and addition pair to be treated as a rename.<br><br>The default can be changed in the config file with the key 'git.renameSimilarityThreshold'. |
| `` t d + `` | 差分コンテキストサイズを増やす | Increase the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` t d - `` | 差分コンテキストサイズを減らす | Decrease the amount of the context shown around changes in the diff view.<br><br>The default can be changed in the config file with the key 'git.diffContextSize'. |
| `` c p `` | チェリーピック |  |
| `` f `` | フェッチ | リモートから変更をフェッチします。 |
| `` : `` | シェルコマンドを実行 | 実行するシェルコマンドを入力するプロンプトを表示します。 |
| `` g h o `` | カスタムパッチオプションを表示 |  |
| `` g o `` | マージ/リベースオプションを表示 | 現在のマージ/リベースを中止/継続/スキップするオプションを表示します。 |
| `` <f5> `` | 更新 | Gitの状態を更新します（`git status`、`git branch`などをバックグラウンドで実行してパネルの内容を更新します）。これは`git fetch`を実行しません。 |
| `` =, + `` | 次の画面モード（通常/半分/全画面） |  |
| `` - `` | 前の画面モード |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | キャンセル |  |
| `` ? `` | キーバインディングメニューを開く |  |
| `` w o `` | フィルターオプションを表示 | コミットログのフィルタリングオプションを表示し、フィルタに一致するコミットのみを表示します。 |
| `` t d c `` | 差分オプションを表示 | ２つのrefの差分に関連するオプションを表示します（例：選択したrefとの差分表示、差分を取るrefの入力、差分方向の反転など）。 |
| `` q `` | 終了 |  |
| `` t a s `` | Suspend the application |  |
| `` t d w `` | 空白表示の切り替え | Toggle whether or not whitespace changes are shown in the diff view.<br><br>The default can be changed in the config file with the key 'git.ignoreWhitespaceInDiffView'. |
| `` t a c `` | 設定ファイルを編集 | 外部エディタでファイルを開きます。 |
| `` u u `` | 元に戻す | 最後のgitコマンドを元に戻すために実行するgitコマンドを決定するためにreflogが使用されます。これにはワーキングツリーへの変更は含まれません。コミットのみが考慮されます。 |
| `` u r `` | やり直す | 最後のgitコマンドをやり直すために実行するgitコマンドを決定するためにreflogが使用されます。これにはワーキングツリーへの変更は含まれません。コミットのみが考慮されます。 |

## リストパネルのナビゲーション

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus ステータス panel |  |
| `` 2 `` | Focus ファイル panel |  |
| `` 3 `` | Focus ブランチ panel |  |
| `` 4 `` | Focus コミット panel |  |
| `` 5 `` | Focus スタッシュ panel |  |
| `` <pgup> `` | 前のページ |  |
| `` <pgdown> `` | 次のページ |  |
| `` <home> `` | 先頭にスクロール |  |
| `` <end> `` | 末尾にスクロール |  |
| `` t h r `` | 範囲選択を切り替え |  |
| `` <shift+down>, J `` | 範囲選択を下に |  |
| `` <shift+up>, K `` | 範囲選択を上に |  |
| `` / `` | 現在のビューをテキストで検索 |  |
| `` , `` | 前のページ |  |
| `` . `` | 次のページ |  |
| `` <, <home> `` | 先頭にスクロール |  |
| `` >, <end> `` | 末尾にスクロール |  |
| `` <alt+left> `` | 左にスクロール |  |
| `` <alt+right> `` | 右にスクロール |  |
| `` <tab> `` | 次のタブ |  |
| `` <backtab> `` | 前のタブ |  |

## Input prompt

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 閉じる/キャンセル |  |

## コミット

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | bisectオプションを表示 |  |
| `` <f2> `` | メッセージ変更 | 選択したコミットのメッセージを変更します。 |
| `` g e r `` | エディタでメッセージ変更 |  |
| `` <delete> `` | 削除 | 選択したコミットを削除します。これはリベースを通じてブランチからコミットを削除します。コミットが後続のコミットが依存する変更を行っている場合、マージコンフリクトを解決する必要があるかもしれません。 |
| `` e `` | 編集（対話型リベースを開始） | 選択したコミットを編集します。これを使用して、選択したコミットから対話型リベースを開始します。すでにリベース中の場合、これは選択したコミットを編集用にマークし、リベースを続行すると、リベースは選択したコミットで一時停止して変更を行えるようにします。 |
| `` g i `` | 対話的リベースを開始 | ブランチ上のコミットの対話的リベースを開始します。これには、HEADコミットから最初のマージコミットまたはメインブランチのコミットまでのすべてのコミットが含まれます。<br>選択したコミットから対話的リベースを開始したい場合は、代わりに `e` を押してください。 |
| `` g r p `` | ピック | 選択したコミットをピックするようにマークします（リベース中）。これは、リベースを続行すると、コミットが保持されることを意味します。 |
| `` > `` | コミットを1つ下に移動 |  |
| `` < `` | コミットを1つ上に移動 |  |
| `` c p `` | チェリーピック |  |
| `` g r b `` | リベース用のベースコミットとしてマーク | 次のリベース用のベースコミットを選択します。ブランチにリベースするとき、ベースコミットより上のコミットのみが持ち込まれます。これは `git rebase --onto` コマンドを使用します。 |
| `` a `` | 修正 | ステージされた変更でコミットを修正します。選択したコミットがHEADコミットの場合、これは `git commit --amend` を実行します。それ以外の場合、コミットはリベースを通じて修正されます。 |
| `` g e a `` | コミット属性を修正 | コミット作者の設定/リセットまたは共同作者の設定を行います。 |
| `` g v c `` | リバート | 選択したコミットの変更を逆に適用する、リバートコミットを作成します。 |
| `` n t `` | コミットにタグを付ける | 選択したコミットを指すタグを新規作成します。タグ名とオプションの説明を入力するよう促されます。 |
| `` t l o `` | ログオプションを表示 | コミットログのオプションを表示します（例：並び順の変更、Gitグラフの非表示、Gitグラフ全体の表示）。 |
| `` g p v `` | Open pull request in browser |  |
| `` w a `` | Show selected commits and their ancestors |  |
| `` w d `` | Show selected commits and their descendants |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | ハードリセット |  |
| `` s m `` | ミックスリセット |  |
| `` s s `` | ソフトリセット（変更を保持してステージ） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択したコミットをデタッチドヘッド（特定のブランチに属さない状態）としてチェックアウトします。 |
| `` o `` | ブラウザでコミットを開く |  |
| `` g m n `` | コミットを新しいブランチに移動 | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | 新しいワークツリー |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 現在のブランチのコミットを選択 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | ファイルを表示 |  |
| `` / `` | 現在のビューをテキストで検索 |  |

## コミットファイル

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | パスをクリップボードにコピー |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | チェックアウト（ブランチの切り替え） | ファイルをチェックアウトします。これにより、作業ツリー内のファイルが選択したコミットのバージョンに置き換えられます。 |
| `` x `` | 破棄 | このコミットのこのファイルへの変更を破棄します。これはバックグラウンドで対話的なリベースを実行するため、後のコミットでもこのファイルが変更されている場合、マージコンフリクトが発生する可能性があります。 |
| `` o `` | ファイルを開く | デフォルトのアプリケーションでファイルを開きます。 |
| `` e `` | 編集 | 外部エディタでファイルを開きます。 |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | パッチに含めるファイルを切り替え | ファイルがカスタムパッチに含まれるかどうかを切り替えます。https://github.com/jesseduffield/lazygit#rebase-magic-custom-patchesを参照してください。 |
| `` g h a `` | すべてのファイルを切り替え | コミットのすべてのファイルをカスタムパッチに追加/削除します。https://github.com/jesseduffield/lazygit#rebase-magic-custom-patchesを参照してください。 |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | ファイルツリービューを切り替え | ファイル表示をフラット表示とツリー表示で切り替えます。フラット表示はすべてのファイルパスを一覧で表示し、ツリー表示はディレクトリごとにファイルをグループ化します。<br><br>デフォルトは設定ファイル内の 'gui.showFileTree' キーで変更できます。 |
| `` t t c `` | すべてのファイルを折りたたむ | ファイルツリー内のすべてのディレクトリを折りたたみます |
| `` t t e `` | すべてのファイルを展開 | ファイルツリー内のすべてのディレクトリを展開します |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## コミット概要

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 閉じる |  |

## サブコミット

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | ハードリセット |  |
| `` s m `` | ミックスリセット |  |
| `` s s `` | ソフトリセット（変更を保持してステージ） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択したコミットをデタッチドヘッド（特定のブランチに属さない状態）としてチェックアウトします。 |
| `` o `` | ブラウザでコミットを開く |  |
| `` g m n `` | コミットを新しいブランチに移動 | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | 新しいワークツリー |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 現在のブランチのコミットを選択 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | ファイルを表示 |  |
| `` / `` | 現在のビューをテキストで検索 |  |

## サブモジュール

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | サブモジュール名をクリップボードにコピー |  |
| `` v, <space> `` | 入る | サブモジュールに入ります。サブモジュールに入った後、`<esc>`を押して親リポジトリに戻ることができます。 |
| `` <delete> `` | 削除 | 選択したサブモジュールとそれに対応するディレクトリを削除します。 |
| `` g u s `` | 更新 | 選択したサブモジュールを更新します。 |
| `` n s `` | 新しいサブモジュール |  |
| `` e `` | サブモジュールURLを更新 |  |
| `` g u i `` | 初期化 | 選択したサブモジュールを初期化してフェッチの準備をします。おそらく、続いて「更新」アクションを呼び出してサブモジュールをフェッチしたいでしょう。 |
| `` g u b `` | 一括サブモジュールオプションを表示 |  |
| `` t a g `` | イースターエッグ |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## スタッシュ

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | 適用 | スタッシュエントリをワーキングディレクトリに適用します。 |
| `` g s p `` | ポップ | スタッシュエントリをワーキングディレクトリに適用し、スタッシュエントリを削除します。 |
| `` <delete> `` | 削除 | スタッシュリストからスタッシュエントリを削除します。 |
| `` n w `` | 新しいワークツリー |  |
| `` <f2> `` | スタッシュの名前を変更 |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | ファイルを表示 |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## ステータス

| Key | Action | Info |
|-----|--------|-------------|
| `` s n `` | Open repository by path |  |
| `` t a c `` | 設定ファイルを編集 | 外部エディタでファイルを開きます。 |
| `` t a u `` | 更新を確認 |  |
| `` s r `` | 最近のリポジトリをチェックアウト |  |
| `` t l n `` | ブランチログの表示モードを順に切り替え |  |
| `` t l p `` | Show/cycle all branch logs (reverse) |  |
| `` v, <space>, 0 `` | メインビューにフォーカス |  |

## セカンダリ

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | ハンクの選択を切り替える | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | 範囲選択を切り替え |  |
| `` e `` | ファイルを編集 | 外部エディタでファイルを開きます。 |
| `` <space>, g h s `` | ステージ | 選択された部分のステージ / アンステージを切り替えます。 |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | パッチ内の行を切り替え |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 破棄 | ステージされていない変更が選択されている場合、`git reset`を使用して変更を破棄します。ステージされた変更が選択されている場合、変更をアンステージします。 |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | ハンクを編集 | 選択したハンクを外部エディタで編集します。 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | 前のハンクに移動 |  |
| `` ], t n h `` | 次のハンクに移動 |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | サイドパネルに戻る |  |
| `` g e n `` | pre-commitフックなしで変更をコミット |  |
| `` g e c `` | Gitエディタを使用して変更をコミット |  |
| `` / `` | 現在のビューをテキストで検索 |  |

## タグ

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | タグをクリップボードにコピー |  |
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
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択したタグをデタッチドHEADとしてチェックアウトします。 |
| `` n t `` | 新しいタグを作成 | 現在のコミットから新しいタグを作成します。タグ名とオプションの説明を入力するよう促されます。 |
| `` n w `` | 新しいワークツリー |  |
| `` <delete> `` | 削除 | ローカル/リモートタグの削除オプションを表示します。 |
| `` p `` | タグをプッシュ | 選択したタグをリモートにプッシュします。リモートを選択するよう促されます。 |
| `` t d e `` | 外部差分ツールを開く（git difftool） |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | コミットを表示 |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## ファイル

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | パスをクリップボードにコピー |  |
| `` <space> `` | Toggle staging of selected files | 選択したファイルのステージ状態を切り替えます。 |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` w s `` | ステータスでファイルをフィルタリング |  |
| `` <enter> `` | コミット | ステージされた変更をコミットします。 |
| `` g e n `` | pre-commitフックなしで変更をコミット |  |
| `` a `` | 直前のコミットを修正 |  |
| `` g e c `` | Gitエディタを使用して変更をコミット |  |
| `` e `` | 編集 | 外部エディタでファイルを開きます。 |
| `` o `` | ファイルを開く | デフォルトのアプリケーションでファイルを開きます。 |
| `` i `` | ファイルを無視または除外 |  |
| `` <f5> `` | 更新 | Gitの状態を更新します（`git status`、`git branch`などをバックグラウンドで実行してパネルの内容を更新します）。これは`git fetch`を実行しません。 |
| `` g s s `` | スタッシュ | すべての変更をスタッシュします。スタッシュの他のバリエーションについては、スタッシュオプションを表示するキーバインディングを使用してください。 |
| `` g s o `` | スタッシュオプションを表示 | スタッシュオプション（すべてをスタッシュ、ステージされた変更をスタッシュ、ステージされていない変更をスタッシュなど）を表示します。 |
| `` x `` | 破棄 | 選択したファイルの変更を破棄するオプションを表示します。 |
| `` g z u `` | アップストリームへのリセットオプションを表示 |  |
| `` g z w `` | リセット | 作業ツリーのリセットオプション（例：作業ツリーの完全破棄）を表示します。 |
| `` t t t `` | ファイルツリービューを切り替え | ファイル表示をフラット表示とツリー表示で切り替えます。フラット表示はすべてのファイルパスを一覧で表示し、ツリー表示はディレクトリごとにファイルをグループ化します。<br><br>デフォルトは設定ファイル内の 'gui.showFileTree' キーで変更できます。 |
| `` t d e `` | 外部差分ツールを開く（git difftool） |  |
| `` t t c `` | すべてのファイルを折りたたむ | ファイルツリー内のすべてのディレクトリを折りたたみます |
| `` t t e `` | すべてのファイルを展開 | ファイルツリー内のすべてのディレクトリを展開します |
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
| `` 0 `` | メインビューにフォーカス |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## メインパネル（マージ中）

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | ハンクを選択 |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | 前のハンク |  |
| `` j, <down> `` | 次のハンク |  |
| `` [, t p c `` | 前のコンフリクト |  |
| `` ], t n c `` | 次のコンフリクト |  |
| `` u u `` | 元に戻す | 最後のマージコンフリクト解決を元に戻します。 |
| `` e `` | ファイルを編集 | 外部エディタでファイルを開きます。 |
| `` o `` | ファイルを開く | デフォルトのアプリケーションでファイルを開きます。 |
| `` g h c `` | View merge conflict options | View options for resolving merge conflicts. |
| `` <esc> `` | ファイルパネルに戻る |  |

## メインパネル（通常）

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | 下にスクロール |  |
| `` <mouse wheel up> (fn+down) `` | 上にスクロール |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | ハンクの選択を切り替える | Toggle line-by-line vs. hunk selection mode. |
| `` t h r `` | 範囲選択を切り替え |  |
| `` e `` | ファイルを編集 | 外部エディタでファイルを開きます。 |
| `` <space>, g h s `` | ステージ | 選択された部分のステージ / アンステージを切り替えます。 |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | パッチ内の行を切り替え |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 破棄 | ステージされていない変更が選択されている場合、`git reset`を使用して変更を破棄します。ステージされた変更が選択されている場合、変更をアンステージします。 |
| `` x `` | Remove lines from commit | Remove the selected lines from this commit. This runs an interactive rebase in the background, so you may get a merge conflict if a later commit also changes these lines. |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | ハンクを編集 | 選択したハンクを外部エディタで編集します。 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | 前のハンクに移動 |  |
| `` ], t n h `` | 次のハンクに移動 |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | サイドパネルに戻る |  |
| `` g e n `` | pre-commitフックなしで変更をコミット |  |
| `` g e c `` | Gitエディタを使用して変更をコミット |  |
| `` / `` | 現在のビューをテキストで検索 |  |

## メニュー

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 実行 |  |
| `` <esc> `` | 閉じる/キャンセル |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## リフログ

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | ハードリセット |  |
| `` s m `` | ミックスリセット |  |
| `` s s `` | ソフトリセット（変更を保持してステージ） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択したコミットをデタッチドヘッド（特定のブランチに属さない状態）としてチェックアウトします。 |
| `` o `` | ブラウザでコミットを開く |  |
| `` g m n `` | コミットを新しいブランチに移動 | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | 新しいワークツリー |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 現在のブランチのコミットを選択 |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## リモート

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | ブランチを表示 |  |
| `` n r `` | 新しいリモート |  |
| `` <delete> `` | 削除 | 選択したリモートを削除します。そのリモートからのリモートブランチを追跡しているローカルブランチは影響を受けません。 |
| `` e `` | 編集 | 選択したリモートの名前またはURLを編集します。 |
| `` f `` | フェッチ | リモートリポジトリから更新をフェッチします。これにより、ローカルブランチにマージせずに新しいコミットとブランチを取得します。 |
| `` n f `` | Add fork remote | Quickly add a fork remote by replacing the owner in the origin URL and optionally check out a branch from new remote. |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## リモートブランチ

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | ブランチ名をクリップボードにコピー |  |
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
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択したリモートブランチに基づいて新しいローカルブランチをチェックアウトするか、リモートブランチをデタッチドヘッドとしてチェックアウトします。 |
| `` n w `` | 新しいワークツリー |  |
| `` m `` | マージ | 選択した項目を現在のブランチにマージするためのオプションを表示します（通常のマージ、スカッシュマージ） |
| `` r `` | Rebase current onto selected branch… | チェックアウトしたブランチを選択したブランチ上にリベースします。 |
| `` <delete> `` | 削除 | リモートからリモートブランチを削除します。 |
| `` g t `` | アップストリームとして設定 | 選択したリモートブランチをチェックアウトされたブランチのアップストリームとして設定します。 |
| `` t s o `` | 並び順 |  |
| `` t d e `` | 外部差分ツールを開く（git difftool） |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | コミットを表示 |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## ローカルブランチ

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | ブランチ名をクリップボードにコピー |  |
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
| `` g y `` | git-flowオプションを表示 |  |
| `` s h `` | ハードリセット |  |
| `` s m `` | ミックスリセット |  |
| `` s s `` | ソフトリセット（変更を保持してステージ） |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | チェックアウト（ブランチの切り替え） | 選択した項目をチェックアウトします。 |
| `` g m n `` | コミットを新しいブランチに移動 | Create a new branch and move the unpushed commits of the current branch to it. Useful if you meant to start new work and forgot to create a new branch first.<br><br>Note that this disregards the selection, the new branch is always created either from the main branch or stacked on top of the current branch (you get to choose which). |
| `` n w `` | 新しいワークツリー |  |
| `` g p c `` | プルリクエストを作成 |  |
| `` g p o `` | プルリクエスト作成オプションを表示 |  |
| `` g p v `` | Open pull request in browser |  |
| `` y r `` | プルリクエストURLをクリップボードにコピー |  |
| `` <delete> `` | 削除 | ローカル/リモートブランチの削除オプションを表示します。 |
| `` r `` | Rebase current onto selected branch… | チェックアウトしたブランチを選択したブランチ上にリベースします。 |
| `` m `` | Merge selected branch into current… | 選択した項目を現在のブランチにマージするためのオプションを表示します（通常のマージ、スカッシュマージ） |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | 新しいタグを作成 |  |
| `` t s o `` | 並び順 |  |
| `` <f2> `` | ブランチ名を変更 |  |
| `` g t `` | アップストリームオプションを表示 | ブランチのアップストリームに関連するオプションを表示します（例：アップストリームの設定/解除やアップストリームへのリセット）。 |
| `` t d e `` | 外部差分ツールを開く（git difftool） |  |
| `` 0 `` | メインビューにフォーカス |  |
| `` v, <space> `` | コミットを表示 |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## ワークツリー

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | 新しいワークツリー |  |
| `` s w `` | チェックアウト（切り替え） | 選択したワークツリーをチェックアウト（切り替え）します。 |
| `` o `` | エディタで開く |  |
| `` <delete> `` | 削除 | 選択したワークツリーを削除します。これはワークツリーのディレクトリとワークツリーに関するメタデータの両方を.gitディレクトリから削除します。 |
| `` v, <space> `` | コミットを表示 |  |
| `` / `` | 現在のビューをテキストでフィルタリング |  |

## 確認パネル

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 閉じる/キャンセル |  |
| `` <ctrl+o> `` | クリップボードにコピー |  |
