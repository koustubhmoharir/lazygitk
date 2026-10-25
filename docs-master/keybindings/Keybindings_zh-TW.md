_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit 鍵盤快捷鍵

## 全域快捷鍵

| Key | Action | Info |
|-----|--------|-------------|
| `` t r r `` | 切換到最近使用的版本庫 |  |
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | 向上捲動主面板 |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | 向下捲動主面板 |  |
| `` t l c `` | 開啟命令記錄選單 | 檢視命令日誌的選項，例如顯示/隱藏命令日誌以及聚焦命令日誌。 |
| `` p `` | 推送 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | 拉取 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | 提高重新命名相似度閾值 | 提高將刪除和新增對視為重新命名所需的相似度閾值。<br><br>預設值可在設定檔中透過鍵 'git.renameSimilarityThreshold' 更改。 |
| `` t d < `` | 降低重新命名相似度閾值 | 降低將刪除和新增對視為重新命名所需的相似度閾值。<br><br>預設值可在設定檔中透過鍵 'git.renameSimilarityThreshold' 更改。 |
| `` t d + `` | 增加差異檢視中顯示變更周圍上下文的大小 | 增加差異檢視中變更周圍顯示的上下文量。<br><br>預設值可在設定檔中透過鍵 'git.diffContextSize' 更改。 |
| `` t d - `` | 減小差異檢視中顯示變更周圍上下文的大小 | 減少差異檢視中變更周圍顯示的上下文量。<br><br>預設值可在設定檔中透過鍵 'git.diffContextSize' 更改。 |
| `` c p `` | 揀選 (Cherry-pick) |  |
| `` : `` | 執行 Shell 命令 | 調出可輸入shell命令執行的提示符。 |
| `` g h o `` | 檢視自訂補丁選項 |  |
| `` g o `` | 查看合併/變基選項 | 檢視目前合併或變基的中止、繼續、跳過選項。 |
| `` <f5> `` | 重新整理 | 重新整理Git狀態（即在背景執行`git status`、`git branch`等命令以更新面板內容）。此操作不會執行`git fetch`。 |
| `` t s n `` | 下一個螢幕模式（常規/半螢幕/全螢幕） |  |
| `` t s p `` | 上一個螢幕模式 |  |
| `` t d n `` | 切換差異渲染器 | 選擇已設定的差異渲染器清單中的下一個渲染器。 |
| `` t d p `` | 切換差異渲染器（反向） | 選擇已設定的差異渲染器清單中的上一個渲染器。 |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | 取消 |  |
| `` ? `` | 開啟選單 |  |
| `` t f o `` | 檢視篩選路徑選項 | 檢視用於過濾提交日誌的選項，以便僅顯示與過濾器匹配的提交。 |
| `` t d c `` | 開啟差異比較選單 | 檢視與比較兩個引用相關的選項，例如與選定的 ref 進行比較，輸入要比較的 ref，然後反轉比較方向。 |
| `` q `` | 結束 |  |
| `` t a s `` | 掛起應用程式 |  |
| `` t d w `` | 切換是否在差異檢視中顯示空格變更 | 切換是否在差異檢視中顯示空白字元更改。<br><br>預設值可在設定檔中透過鍵 'git.ignoreWhitespaceInDiffView' 更改。 |
| `` t a c `` | 編輯設定檔案 | 使用外部編輯器開啟 |
| `` u u `` | 復原 | 將使用 reflog 確任 git 指令以復原。這不包括工作區更改；只考慮提交。 |
| `` u r `` | 取消復原 | 將使用 reflog 確任 git 指令以重作。這不包括工作區更改；只考慮提交。 |

## 移動

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus 狀態 panel |  |
| `` 2 `` | Focus 檔案 panel |  |
| `` 3 `` | Focus 分支 panel |  |
| `` 4 `` | Focus 提交 panel |  |
| `` 5 `` | Focus 收藏 (Stash) panel |  |
| `` <pgup> `` | 上一頁 |  |
| `` <pgdown> `` | 下一頁 |  |
| `` <home> `` | 捲動到頂部 |  |
| `` <end> `` | 捲動到底部 |  |
| `` t h r `` | 切換拖曳選擇 |  |
| `` <shift+down> `` | 向下擴充套件選擇範圍 |  |
| `` <shift+up> `` | 向上擴充套件選擇範圍 |  |
| `` / `` | 搜尋 |  |
| `` , `` | 上一頁 |  |
| `` . `` | 下一頁 |  |
| `` <, <home> `` | 捲動到頂部 |  |
| `` >, <end> `` | 捲動到底部 |  |
| `` <alt+left> `` | 向左捲動 |  |
| `` <alt+right> `` | 向右捲動 |  |
| `` <tab> `` | 下一個索引標籤 |  |
| `` <backtab> `` | 上一個索引標籤 |  |

## 主面板（一般）

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | 向下捲動 |  |
| `` <mouse wheel up> (fn+down) `` | 向上捲動 |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | 切換程式碼塊選擇 | 切換逐行選擇與程式碼塊選擇模式。 |
| `` t h r `` | 切換拖曳選擇 |  |
| `` e `` | 編輯檔案 | 使用外部編輯器開啟 |
| `` g h s `` | 切換預存 | 切換現有行的狀態 (已預存/未預存) |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | 向 (或從) 補丁中添加/刪除行 |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 刪除變更 (git reset) | 選取未暫存的變更時，使用 `git reset` 捨棄變更。選取已暫存的變更時，取消暫存變更。 |
| `` x `` | 從提交中移除行 | 從本次提交中移除所選行。此操作會在背景執行互動式變基，因此如果後續提交也修改了這些行，您可能會遇到合併衝突。 |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | 編輯程式碼塊 | 在外部編輯器中編輯選中的程式碼塊。 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | 選擇上一段 |  |
| `` t n h `` | 選擇下一段 |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | 退出回到側邊面板 |  |
| `` g e n `` | 沒有預提交 hook 就提交更改 |  |
| `` g e c `` | 使用 git 編輯器提交變更 |  |
| `` / `` | 搜尋 |  |

## 主面板（合併）

| Key | Action | Info |
|-----|--------|-------------|
| `` g h p `` | 挑選程式碼片段 |  |
| `` g h b `` | 選取兩個區塊 |  |
| `` k, <up> `` | 選擇上一段 |  |
| `` j, <down> `` | 選擇下一段 |  |
| `` t p c `` | 選擇上一個衝突 |  |
| `` t n c `` | 選擇下一個衝突 |  |
| `` u u `` | 復原 | 撤消上次合併衝突解決。 |
| `` e `` | 編輯檔案 | 使用外部編輯器開啟 |
| `` o `` | 開啟檔案 | 使用預設軟體開啟 |
| `` g h c `` | 檢視合併衝突選項 | 檢視用於解決合併衝突的選項。 |
| `` <esc> `` | 返回檔案面板 |  |

## 功能表

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 執行 |  |
| `` <esc> `` | 關閉/取消 |  |
| `` / `` | 搜尋 |  |

## 子提交

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | 強制重設 |  |
| `` s m `` | 混合重設 |  |
| `` s s `` | 軟重設（保留變動） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 檢出 | 檢出所選擇的提交作為分離HEAD。 |
| `` o `` | 在瀏覽器中開啟提交 |  |
| `` g m n `` | 移動提交至新分支 | 建立一個新分支，並將目前分支未推送的提交移動到該分支。如果您打算開始新工作但忘記先建立新分支，這會很有用。<br><br>請注意，此操作忽略選擇，新分支總是從主分支建立或堆疊在目前分支之上（您可以選擇哪種方式）。 |
| `` n w `` | 新建工作樹 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 選擇目前分支的提交 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` v, <space> `` | 檢視所選項目的檔案 |  |
| `` / `` | 搜尋 |  |

## 子模組

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 複製子模組名稱到剪貼簿 |  |
| `` t r s `` | 進入 | 進入子模組 |
| `` <delete> `` | 刪除 | 刪除選定的子模組及其相應的目錄。 |
| `` g u s `` | 更新 | 更新子模組 |
| `` n s `` | 新增子模組 |  |
| `` e `` | 更新子模組 URL |  |
| `` g u i `` | 初始化 | 初始化子模組 |
| `` g u b `` | 查看批量子模組選項 |  |
| `` t a g `` | 彩蛋 |  |
| `` t f t `` | 搜尋 |  |

## 工作目錄

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | 新建工作樹 |  |
| `` t r w `` | 切換 | 切換到選中的工作樹。 |
| `` o `` | 在編輯器中開啟 |  |
| `` <delete> `` | 刪除 | 刪除選定的工作樹。這將刪除工作樹的目錄以及 .git 目錄中有關工作樹的後設資料。 |
| `` t f t `` | 搜尋 |  |

## 提交

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | 查看二分選項 |  |
| `` <f2> `` | 改寫提交 | 改寫選中的提交訊息 |
| `` g e r `` | 使用編輯器改寫提交 |  |
| `` <delete> `` | 刪除提交 | 刪除選中的提交。這將透過變基從分支中刪除該提交，如果該提交修改的內容依賴於後續的提交，則需要解決合併衝突。 |
| `` e `` | 編輯(開始互動變基) | 編輯提交 |
| `` g i `` | 開始互動變基 | 為分支上的提交啟動互動式變基。這將包括從 HEAD 提交到第一個合併提交或主分支提交的所有提交。<br>如果您想從所選提交啟動互動式變基，請按 `e`。 |
| `` g r p `` | 挑選 | 挑選提交 (於變基過程中) |
| `` g m d `` | 向下移動提交 |  |
| `` g m u `` | 向上移動提交 |  |
| `` c p `` | 揀選 (Cherry-pick) |  |
| `` g r b `` | 為了變基已標注提交為基準提交 | 請為了下一次變基選擇一項基準提交；此將執行 `git rebase --onto`。 |
| `` a `` | 修改 | 使用已預存的更改修正提交 |
| `` g e a `` | 設定/重設提交作者 | 設定或重置提交的作者，或新增其他作者。 |
| `` g v c `` | 還原 | 為所選提交建立還原提交，這會反向應用所選提交的更改。 |
| `` n t `` | 打標籤到提交 | 建立一個新標籤指向所選提交。您可以在彈窗中輸入標籤名稱和描述(可選)。 |
| `` t l o `` | 開啟記錄選單 | 檢視提交日誌的選項，例如更改排序順序、隱藏 git graph、顯示整個 git graph。 |
| `` g p v `` | 在瀏覽器中開啟拉取請求 |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | 強制重設 |  |
| `` s m `` | 混合重設 |  |
| `` s s `` | 軟重設（保留變動） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 檢出 | 檢出所選擇的提交作為分離HEAD。 |
| `` o `` | 在瀏覽器中開啟提交 |  |
| `` g m n `` | 移動提交至新分支 | 建立一個新分支，並將目前分支未推送的提交移動到該分支。如果您打算開始新工作但忘記先建立新分支，這會很有用。<br><br>請注意，此操作忽略選擇，新分支總是從主分支建立或堆疊在目前分支之上（您可以選擇哪種方式）。 |
| `` n w `` | 新建工作樹 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 選擇目前分支的提交 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` v, <space> `` | 檢視所選項目的檔案 |  |
| `` / `` | 搜尋 |  |

## 提交摘要

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 關閉 |  |

## 提交檔案

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 複製檔案名稱到剪貼簿 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | 檢出 | 檢出檔案 |
| `` x `` | 捨棄 | 放棄對此檔案的提交變更。 |
| `` o `` | 開啟檔案 | 使用預設軟體開啟 |
| `` e `` | 編輯 | 使用外部編輯器開啟 |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | 切換檔案是否包含在補丁中 | 切換檔案是否包含在自定義補丁中。請參閱 https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches。 |
| `` g h a `` | 切換所有檔案是否包含在補丁中 | 新增或刪除所有提交中的檔案到自定義的補丁中。請參閱 https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches。 |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | 顯示檔案樹狀視圖 | 在平面佈局和樹佈局之間切換檔案檢視。平面佈局在單個列表中顯示所有檔案路徑，樹佈局按目錄分組檔案。<br><br>可以在設定檔中使用 'gui.showFileTree' 鍵更改預設設定。 |
| `` t t c `` | 摺疊全部檔案 | 摺疊檔案樹中的全部目錄 |
| `` t t e `` | 展開全部檔案 | 展開檔案樹中的全部目錄 |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` t f t `` | 搜尋 |  |

## 收藏 (Stash)

| Key | Action | Info |
|-----|--------|-------------|
| `` g a `` | 套用 | 將貯藏項應用到您的工作目錄。 |
| `` g s p `` | 還原 | 將儲存項應用到工作目錄並刪除儲存項。 |
| `` <delete> `` | 捨棄 | 從貯藏列表中刪除該貯藏項。 |
| `` n w `` | 新建工作樹 |  |
| `` <f2> `` | 重新命名收藏 |  |
| `` v, <space> `` | 檢視所選項目的檔案 |  |
| `` t f t `` | 搜尋 |  |

## 日誌

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s h `` | 強制重設 |  |
| `` s m `` | 混合重設 |  |
| `` s s `` | 軟重設（保留變動） |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 檢出 | 檢出所選擇的提交作為分離HEAD。 |
| `` o `` | 在瀏覽器中開啟提交 |  |
| `` g m n `` | 移動提交至新分支 | 建立一個新分支，並將目前分支未推送的提交移動到該分支。如果您打算開始新工作但忘記先建立新分支，這會很有用。<br><br>請注意，此操作忽略選擇，新分支總是從主分支建立或堆疊在目前分支之上（您可以選擇哪種方式）。 |
| `` n w `` | 新建工作樹 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 選擇目前分支的提交 |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` t f t `` | 搜尋 |  |

## 本地分支

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 複製分支名稱到剪貼簿 |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` g y `` | 顯示 git-flow 選項 |  |
| `` s h `` | 強制重設 |  |
| `` s m `` | 混合重設 |  |
| `` s s `` | 軟重設（保留變動） |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | 檢出 | 檢出選定的項目。 |
| `` g m n `` | 移動提交至新分支 | 建立一個新分支，並將目前分支未推送的提交移動到該分支。如果您打算開始新工作但忘記先建立新分支，這會很有用。<br><br>請注意，此操作忽略選擇，新分支總是從主分支建立或堆疊在目前分支之上（您可以選擇哪種方式）。 |
| `` n w `` | 新建工作樹 |  |
| `` g p c `` | 建立拉取請求 |  |
| `` g p o `` | 建立拉取請求選項 |  |
| `` g p v `` | 在瀏覽器中開啟拉取請求 |  |
| `` y r `` | 複製拉取請求的 URL 到剪貼板 |  |
| `` <delete> `` | 刪除 | 檢視本地/遠端分支的刪除選項。 |
| `` r `` | Rebase current onto selected branch… | 將檢出的分支變基到所選的分支上。 |
| `` m `` | Merge selected branch into current… | 檢視將選中項合併到目前分支的選項(正常合併，壓縮合並) |
| `` g f `` | Fast-forward selected branches… | Review an upstream fetch and a forward-only update of the captured selected branches. Divergent or rewritten history is never reset; tips occupied by another worktree are not moved. A selected checked-out branch uses native fast-forward integration with explicit review. Multiple selections require final batch approval. Saved upstream settings stay unchanged. |
| `` n t `` | 建立標籤 |  |
| `` t s o `` | 排序規則 |  |
| `` <f2> `` | 重新命名分支 |  |
| `` g t `` | 檢視遠端設定 | 檢視有關遠端分支的設定（例如重設至遠端） |
| `` t d e `` | 開啟外部差異工具 (git difftool) |  |
| `` v, <space> `` | 檢視提交 |  |
| `` t f t `` | 搜尋 |  |

## 標籤

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 複製標籤到剪貼簿 |  |
| `` y h `` | Copy commit hash |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` s c `` | 檢出 | 檢出選擇的標籤作為分離的HEAD。 |
| `` n t `` | 建立標籤 | 基於目前提交建立一個新標籤。您將在彈窗中輸入標籤名稱和描述(可選)。 |
| `` n w `` | 新建工作樹 |  |
| `` <delete> `` | 刪除 | 檢視本機／遠端標籤的刪除選項。 |
| `` p `` | 推送標籤 | 推送選擇的標籤到遠端。您將在彈窗中選擇一個遠端。 |
| `` t d e `` | 開啟外部差異工具 (git difftool) |  |
| `` v, <space> `` | 檢視提交 |  |
| `` t f t `` | 搜尋 |  |

## 檔案

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 複製檔案名稱到剪貼簿 |  |
| `` <space> `` | Toggle staging of selected files | 切換所選檔案的暫存狀態。 |
| `` g h c `` | 檢視合併衝突選項 | 檢視用於解決合併衝突的選項。 |
| `` t f s `` | 篩選檔案 (預存/未預存) |  |
| `` <enter> `` | 提交變更 | 提交暫存區變更 |
| `` g e n `` | 沒有預提交 hook 就提交更改 |  |
| `` a `` | 修改上次提交 |  |
| `` g e c `` | 使用 git 編輯器提交變更 |  |
| `` e `` | 編輯 | 使用外部編輯器開啟 |
| `` o `` | 開啟檔案 | 使用預設軟體開啟 |
| `` i `` | 忽略或排除檔案 |  |
| `` <f5> `` | 重新整理 | 重新整理Git狀態（即在背景執行`git status`、`git branch`等命令以更新面板內容）。此操作不會執行`git fetch`。 |
| `` g s s `` | 收藏 | 貯藏所有變更.若要使用其他貯藏變體,請使用檢視貯藏選項快捷鍵。 |
| `` g s o `` | 檢視收藏選項 | 檢視貯藏選項（例如：貯藏所有、貯藏已暫存變更、貯藏未暫存變更）。 |
| `` x `` | 捨棄 | 檢視選中變動進行捨棄復原 |
| `` g z u `` | 檢視遠端重設選項 |  |
| `` g z w `` | 重設 | 檢視工作樹的重置選項（例如：清除工作樹）。 |
| `` t t t `` | 顯示檔案樹狀視圖 | 在平面佈局和樹佈局之間切換檔案檢視。平面佈局在單個列表中顯示所有檔案路徑，樹佈局按目錄分組檔案。<br><br>可以在設定檔中使用 'gui.showFileTree' 鍵更改預設設定。 |
| `` t d e `` | 開啟外部差異工具 (git difftool) |  |
| `` f `` | 擷取 | 同步遠端異動 |
| `` t t c `` | 摺疊全部檔案 | 摺疊檔案樹中的全部目錄 |
| `` t t e `` | 展開全部檔案 | 展開檔案樹中的全部目錄 |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` v `` | 聚焦主檢視 |  |
| `` t f t `` | 搜尋 |  |

## 次要

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | 切換程式碼塊選擇 | 切換逐行選擇與程式碼塊選擇模式。 |
| `` t h r `` | 切換拖曳選擇 |  |
| `` e `` | 編輯檔案 | 使用外部編輯器開啟 |
| `` g h s `` | 切換預存 | 切換現有行的狀態 (已預存/未預存) |
| `` g h u `` | Unstage selected changes |  |
| `` g h t `` | 向 (或從) 補丁中添加/刪除行 |  |
| `` g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 刪除變更 (git reset) | 選取未暫存的變更時，使用 `git reset` 捨棄變更。選取已暫存的變更時，取消暫存變更。 |
| `` x `` | 從提交中移除行 | 從本次提交中移除所選行。此操作會在背景執行互動式變基，因此如果後續提交也修改了這些行，您可能會遇到合併衝突。 |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | 編輯程式碼塊 | 在外部編輯器中編輯選中的程式碼塊。 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` t p h `` | 選擇上一段 |  |
| `` t n h `` | 選擇下一段 |  |
| `` t p f `` | Go to previous file |  |
| `` t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | 退出回到側邊面板 |  |
| `` g e n `` | 沒有預提交 hook 就提交更改 |  |
| `` g e c `` | 使用 git 編輯器提交變更 |  |
| `` / `` | 搜尋 |  |

## 狀態

| Key | Action | Info |
|-----|--------|-------------|
| `` t a c `` | 編輯設定檔案 | 使用外部編輯器開啟 |
| `` t a u `` | 檢查更新 |  |
| `` t r r `` | 切換到最近使用的版本庫 |  |
| `` t l n `` | 顯示/迴圈所有分支日誌 |  |
| `` t l p `` | 顯示/迴圈所有分支日誌（反向） |  |
| `` v, <space> `` | 聚焦主檢視 |  |

## 確認面板

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 關閉/取消 |  |
| `` <ctrl+o> `` | 複製到剪貼簿 |  |

## 輸入提示

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 確認 |  |
| `` <esc> `` | 關閉/取消 |  |

## 遠端

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | 檢視分支 |  |
| `` n r `` | 新增遠端 |  |
| `` <delete> `` | 刪除 | 刪除選中的遠端。從遠端跟蹤遠端分支的任何本地分支都不會受到影響。 |
| `` e `` | 編輯 | 編輯遠端 |
| `` f `` | 擷取 | 擷取遠端 |
| `` n f `` | 新增復刻遠端倉庫 | 透過替換 origin URL 中的所有者來快速新增復刻遠端倉庫，並可選擇從新遠端倉庫檢出分支。 |
| `` t f t `` | 搜尋 |  |

## 遠端分支

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 複製分支名稱到剪貼簿 |  |
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
| `` s c `` | 檢出 | 基於目前選中的遠端分支檢出一個新的本地分支，或者將遠端分支作分離的HEAD。 |
| `` n w `` | 新建工作樹 |  |
| `` m `` | 合併到當前檢出的分支 | 檢視將選中項合併到目前分支的選項(正常合併，壓縮合並) |
| `` r `` | Rebase current onto selected branch… | 將檢出的分支變基到所選的分支上。 |
| `` <delete> `` | 刪除 | 從遠端刪除遠端分支。 |
| `` g t `` | 設置為遠端 | 將此分支設為當前分支之遠端 |
| `` t s o `` | 排序規則 |  |
| `` t d e `` | 開啟外部差異工具 (git difftool) |  |
| `` v, <space> `` | 檢視提交 |  |
| `` t f t `` | 搜尋 |  |
