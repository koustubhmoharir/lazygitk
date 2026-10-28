_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit 按键绑定

## 全局键绑定

| Key | Action | Info |
|-----|--------|-------------|
| `` <alt+up>, <pgup> (fn+up/shift+k) `` | 向上滚动主面板 |  |
| `` <alt+down>, <pgdown> (fn+down/shift+j) `` | 向下滚动主面板 |  |
| `` t l c `` | 打开命令日志菜单 | 查看命令日志的选项，例如显示/隐藏命令日志以及聚焦命令日志 |
| `` p `` | 推送 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` g l `` | 拉取 | Uses the branch selected when the action starts; changing selection cannot redirect an open prompt or running operation. |
| `` t d > `` | 提高重命名相似度阈值 | 提高将删除和添加对视为重命名所需的相似度阈值。<br><br>默认值可在配置文件中通过键 'git.renameSimilarityThreshold' 更改。 |
| `` t d < `` | 降低重命名相似度阈值 | 降低将删除和添加对视为重命名所需的相似度阈值。<br><br>默认值可在配置文件中通过键 'git.renameSimilarityThreshold' 更改。 |
| `` t d + `` | 扩大差异视图中显示的上下文范围 | 增加差异视图中变更周围显示的上下文量。<br><br>默认值可在配置文件中通过键 'git.diffContextSize' 更改。 |
| `` t d - `` | 缩小差异视图中显示的上下文范围 | 减少差异视图中变更周围显示的上下文量。<br><br>默认值可在配置文件中通过键 'git.diffContextSize' 更改。 |
| `` c p `` | 拣选(Cherry-Pick) |  |
| `` f `` | 抓取 | 从远程获取变更 |
| `` : `` | 执行 Shell 命令 | 调出可输入shell命令执行的提示符。 |
| `` g h o `` | 查看自定义补丁选项 |  |
| `` g o `` | 查看合并/变基选项 | 查看当前合并或变基的中止、继续、跳过选项 |
| `` <f5> `` | 刷新 | 刷新Git状态（即在后台运行`git status`、`git branch`等命令以更新面板内容）。此操作不会执行`git fetch`。 |
| `` =, + `` | 下一屏模式(正常/半屏/全屏) |  |
| `` - `` | 上一屏模式 |  |
| `` t d n `` | Cycle diff renderers | Choose the next renderer in the list of configured diff renderers. |
| `` t d p `` | Cycle diff renderers (reverse) | Choose the previous renderer in the list of configured diff renderers. |
| `` t n j `` | Jump to file in diff | Pick one of the files of the diff shown in the main view, and scroll the main view to it. The focus stays in this panel. |
| `` <esc> `` | 取消 |  |
| `` ? `` | 打开菜单 |  |
| `` w o `` | 查看按路径过滤选项 | 查看用于过滤提交日志的选项，以便仅显示与过滤器匹配的提交。 |
| `` t d c `` | 打开 diff 菜单 | 查看与比较两个引用相关的选项，例如与选定的 ref 进行比较，输入要比较的 ref，然后反转比较方向。 |
| `` q `` | 退出 |  |
| `` t a s `` | 挂起应用程序 |  |
| `` t d w `` | 切换是否在差异视图中显示空白字符差异 | 切换是否在差异视图中显示空白字符更改。<br><br>默认值可在配置文件中通过键 'git.ignoreWhitespaceInDiffView' 更改。 |
| `` t a c `` | 编辑配置文件 | 使用外部编辑器打开文件 |
| `` u u `` | 撤销 | Reflog将用于确定运行哪个git命令来撤消最后一个git命令。这并不包括对工作树的更改，只考虑提交。 |
| `` u r `` | 重做 | Reflog将用于确定运行哪个git命令来重做上一个git命令。这并不包括对工作树的更改，只考虑提交。 |

## 列表面板导航

| Key | Action | Info |
|-----|--------|-------------|
| `` 1 `` | Focus 状态 panel |  |
| `` 2 `` | Focus 文件 panel |  |
| `` 3 `` | Focus 分支 panel |  |
| `` 4 `` | Focus 提交 panel |  |
| `` 5 `` | Focus 贮藏 panel |  |
| `` <pgup> `` | 上一页 |  |
| `` <pgdown> `` | 下一页 |  |
| `` <home> `` | 滚动到顶部 |  |
| `` <end> `` | 滚动到底部 |  |
| `` t h r `` | 切换拖动选择 |  |
| `` <shift+down>, J `` | 向下扩展选择范围 |  |
| `` <shift+up>, K `` | 向上扩展选择范围 |  |
| `` / `` | 开始搜索 |  |
| `` , `` | 上一页 |  |
| `` . `` | 下一页 |  |
| `` <, <home> `` | 滚动到顶部 |  |
| `` >, <end> `` | 滚动到底部 |  |
| `` <alt+left> `` | 向左滚动 |  |
| `` <alt+right> `` | 向右滚动 |  |
| `` <tab> `` | 下一个标签 |  |
| `` <backtab> `` | 上一个标签 |  |

## 子提交

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | 硬重置 |  |
| `` s m `` | 混合重置 |  |
| `` s s `` | 软重置 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 检出 | 检出所选择的提交作为分离HEAD。 |
| `` o `` | 在浏览器中打开提交 |  |
| `` g m n `` | 移动提交至新分支 | 创建一个新分支，并将当前分支未推送的提交移动到该分支。如果您打算开始新工作但忘记先创建新分支，这会很有用。<br><br>请注意，此操作忽略选择，新分支总是从主分支创建或堆叠在当前分支之上（您可以选择哪种方式）。 |
| `` n w `` | 新建工作树 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 选择当前分支的提交 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交的文件 |  |
| `` / `` | 开始搜索 |  |

## 子模块

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 复制子模块名称到剪贴板 |  |
| `` v, <space> `` | 进入 | 输入子模块 |
| `` <delete> `` | 删除 | 删除选定的子模块及其相应的目录 |
| `` g u s `` | 更新 | 更新子模块 |
| `` n s `` | 添加新的子模块 |  |
| `` e `` | 更新子模块 URL |  |
| `` g u i `` | 初始化 | 初始化子模块 |
| `` g u b `` | 查看批量子模块选项 |  |
| `` t a g `` | 彩蛋 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 工作区

| Key | Action | Info |
|-----|--------|-------------|
| `` n w `` | 新建工作树 |  |
| `` s w `` | 切换 | 切换到选中的工作树 |
| `` o `` | 在编辑器中编写 |  |
| `` <delete> `` | 删除 | 删除选定的工作树。这将删除工作树的目录以及 .git 目录中有关工作树的元数据。 |
| `` v, <space> `` | 查看提交 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 引用日志

| Key | Action | Info |
|-----|--------|-------------|
| `` y h `` | Copy commit hash |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | 硬重置 |  |
| `` s m `` | 混合重置 |  |
| `` s s `` | 软重置 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 检出 | 检出所选择的提交作为分离HEAD。 |
| `` o `` | 在浏览器中打开提交 |  |
| `` g m n `` | 移动提交至新分支 | 创建一个新分支，并将当前分支未推送的提交移动到该分支。如果您打算开始新工作但忘记先创建新分支，这会很有用。<br><br>请注意，此操作忽略选择，新分支总是从主分支创建或堆叠在当前分支之上（您可以选择哪种方式）。 |
| `` n w `` | 新建工作树 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 选择当前分支的提交 |  |
| `` y i `` | Copy abbreviated commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | 聚焦主视图 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 提交

| Key | Action | Info |
|-----|--------|-------------|
| `` y i `` | Copy abbreviated commit hash |  |
| `` g b `` | 查看二分查找选项 |  |
| `` <f2> `` | 改写提交 | 重写所选提交的消息。 |
| `` g e r `` | 使用编辑器重命名提交 |  |
| `` <delete> `` | 删除提交 | 删除选中的提交。这将通过变基从分支中删除该提交，如果该提交修改的内容依赖于后续的提交，则需要解决合并冲突。 |
| `` e `` | 编辑(开始交互式变基) | 编辑提交 |
| `` g i `` | 开始交互式变基 | 为分支上的提交启动交互式变基。这将包括从 HEAD 提交到第一个合并提交或主分支提交的所有提交。<br>如果您想从所选提交启动交互式变基，请按 `e`。 |
| `` g r p `` | 拣选(Pick) | 标记选中的提交为 picked（变基过程中）。这意味该提交将在后续的变基中保留。 |
| `` > `` | 下移提交 |  |
| `` < `` | 上移提交 |  |
| `` c p `` | 拣选(Cherry-Pick) |  |
| `` g r b `` | 标记一个主提交用于变基 | 选择下一次变基的主提交。当您变基到一个分支时，只有高于主提交的提交才会被引入。这使用“git rebase --onto”命令。 |
| `` a `` | 修补(Amend) | 用已暂存的变更来修补提交 |
| `` g e a `` | 修补提交属性 | 设置或重置提交的作者，或添加其他作者。 |
| `` g v c `` | 撤销(Revert) | 为所选提交创建还原提交，这会反向应用所选提交的更改。 |
| `` n t `` | 标签提交 | 创建一个新标签指向所选提交。您可以在弹窗中输入标签名称和描述(可选)。 |
| `` t l o `` | 打开日志菜单 | 查看提交日志的选项，例如更改排序顺序、隐藏 git graph、显示整个 git graph。 |
| `` g p v `` | 在浏览器中打开拉取请求 |  |
| `` w a `` | Show selected commits and their ancestors |  |
| `` w d `` | Show selected commits and their descendants |  |
| `` c y `` | Mark selected commits for cherry-pick |  |
| `` c x `` | Unmark selected commits for cherry-pick |  |
| `` s d `` | Checkout detached HEAD |  |
| `` s h `` | 硬重置 |  |
| `` s m `` | 混合重置 |  |
| `` s s `` | 软重置 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` d c `` | Diff with commit… |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` r `` | Rebase current branch onto this commit… | Rebase the checked-out branch onto the selected commit. Requires no tracked changes; never stashes or updates other branch refs. Independent of the marked base. |
| `` g c `` | Combine selected commits |  |
| `` s c `` | 检出 | 检出所选择的提交作为分离HEAD。 |
| `` o `` | 在浏览器中打开提交 |  |
| `` g m n `` | 移动提交至新分支 | 创建一个新分支，并将当前分支未推送的提交移动到该分支。如果您打算开始新工作但忘记先创建新分支，这会很有用。<br><br>请注意，此操作忽略选择，新分支总是从主分支创建或堆叠在当前分支之上（您可以选择哪种方式）。 |
| `` n w `` | 新建工作树 |  |
| `` t d e `` | Diff using current comparison mode |  |
| `` t l s `` | 选择当前分支的提交 |  |
| `` y h `` | Copy commit hash |  |
| `` y s `` | Copy commit subject |  |
| `` y m `` | Copy commit message |  |
| `` y b `` | Copy commit message body |  |
| `` y u `` | Copy commit URL |  |
| `` y d `` | Copy commit diff |  |
| `` y a `` | Copy commit author |  |
| `` y t `` | Copy commit tags |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交的文件 |  |
| `` / `` | 开始搜索 |  |

## 提交信息

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 确认 |  |
| `` <esc> `` | 关闭 |  |

## 提交文件

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 复制路径到剪贴板 |  |
| `` d w `` | Diff with working tree |  |
| `` d p `` | Diff with previous commit |  |
| `` t d f `` | Diff with working file… |  |
| `` d c `` | Diff with commit… |  |
| `` g v f `` | 检出 | 检出文件 |
| `` x `` | 查看'放弃变更'选项 | 放弃对此文件的提交变更 |
| `` o `` | 打开文件 | 使用默认程序打开该文件 |
| `` e `` | 编辑(Edit) | 使用外部编辑器打开文件 |
| `` t d e `` | Diff using current comparison mode |  |
| `` g h t `` | 补丁中包含的切换文件 | 切换文件是否包含在自定义补丁中。请参阅 https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches。 |
| `` g h a `` | 操作所有文件 | 添加或删除所有提交中的文件到自定义的补丁中。请参阅 https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches。 |
| `` v, <space> `` | Focus file diff / Toggle directory | If a file is selected, focus its diff so you can act on individual lines. If it is a directory, collapse or expand it. |
| `` t t t `` | 切换文件树视图 | 在平面布局和树布局之间切换文件视图。平面布局在单个列表中显示所有文件路径，树布局按目录分组文件。<br><br>可以在配置文件中使用 'gui.showFileTree' 键更改默认设置。 |
| `` t t c `` | 折叠全部文件 | 折叠文件树中的全部目录 |
| `` t t e `` | 展开全部文件 | 展开文件树中的全部目录 |
| `` <left> `` | Collapse directory or select parent |  |
| `` <right> `` | Expand directory or select first child |  |
| `` y n `` | Copy file name |  |
| `` y f `` | Copy absolute path |  |
| `` y d `` | Copy file diff |  |
| `` y c `` | Copy file content |  |
| `` 0 `` | 聚焦主视图 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 文件

| Key | Action | Info |
|-----|--------|-------------|
| `` y p `` | 复制路径到剪贴板 |  |
| `` <space> `` | Toggle staging of selected files | 为选定的文件切换暂存状态 |
| `` g h c `` | 查看合并冲突选项 | 查看用于解决合并冲突的选项。 |
| `` w s `` | 通过状态过滤文件 |  |
| `` <enter> `` | 提交变更 | 提交暂存文件 |
| `` g e n `` | 提交变更而无需预先提交钩子 |  |
| `` a `` | 修补最后一次提交 |  |
| `` g e c `` | 使用 Git 编辑器提交变更 |  |
| `` e `` | 编辑(Edit) | 使用外部编辑器打开文件 |
| `` o `` | 打开文件 | 使用默认程序打开该文件 |
| `` i `` | 忽略文件 |  |
| `` <f5> `` | 刷新 | 刷新Git状态（即在后台运行`git status`、`git branch`等命令以更新面板内容）。此操作不会执行`git fetch`。 |
| `` g s s `` | 贮藏 | 贮藏所有变更.若要使用其他贮藏变体,请使用查看贮藏选项快捷键 |
| `` g s o `` | 查看贮藏选项 | 查看贮藏选项（例如：贮藏所有、贮藏已暂存变更、贮藏未暂存变更） |
| `` x `` | 查看'放弃变更'选项 | 查看选中文件的放弃变更选项 |
| `` g z u `` | 查看上游重置选项 |  |
| `` g z w `` | 重置 | 查看工作树的重置选项（例如：清除工作树）。 |
| `` t t t `` | 切换文件树视图 | 在平面布局和树布局之间切换文件视图。平面布局在单个列表中显示所有文件路径，树布局按目录分组文件。<br><br>可以在配置文件中使用 'gui.showFileTree' 键更改默认设置。 |
| `` t d e `` | 使用外部差异比较工具(git difftool) |  |
| `` t t c `` | 折叠全部文件 | 折叠文件树中的全部目录 |
| `` t t e `` | 展开全部文件 | 展开文件树中的全部目录 |
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
| `` 0 `` | 聚焦主视图 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 本地分支

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 复制分支名称到剪贴板 |  |
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
| `` g y `` | 显示 git-flow 选项 |  |
| `` s h `` | 硬重置 |  |
| `` s m `` | 混合重置 |  |
| `` s s `` | 软重置 |  |
| `` n b `` | Create/move branch here… | Create or move a local branch to the selected commit without changing HEAD, the index or working files. |
| `` s c `` | 检出 | 检出选中的项目 |
| `` g m n `` | 移动提交至新分支 | 创建一个新分支，并将当前分支未推送的提交移动到该分支。如果您打算开始新工作但忘记先创建新分支，这会很有用。<br><br>请注意，此操作忽略选择，新分支总是从主分支创建或堆叠在当前分支之上（您可以选择哪种方式）。 |
| `` n w `` | 新建工作树 |  |
| `` g p c `` | 创建拉取请求 |  |
| `` g p o `` | 创建拉取请求选项 |  |
| `` g p v `` | 在浏览器中打开拉取请求 |  |
| `` y r `` | 复制拉取请求 URL 到剪贴板 |  |
| `` <delete> `` | 删除 | 查看本地/远程分支的删除选项 |
| `` r `` | Rebase current onto selected branch… | 将检出的分支变基到所选的分支上。 |
| `` m `` | Merge selected branch into current… | 查看将选中项合并到当前分支的选项(正常合并，压缩合并) |
| `` g f `` | Fast-forward selected branches | Move each selected branch to its upstream as it exists locally, without fetching. A branch behind its upstream fast-forwards with no prompt; a checked-out branch uses a fast-forward-only merge that keeps your uncommitted changes. You are asked only before overwriting ignored files. Diverged branches, branches without an upstream or with a missing tracking ref, and branches another worktree occupies are listed in one alert at the end. |
| `` n t `` | 创建标签 |  |
| `` t s o `` | 排序 |  |
| `` <f2> `` | 重命名分支 |  |
| `` g t `` | 查看上游选项 | 查看与分支上游相关的选项，例如设置/取消设置上游和重置为上游。 |
| `` t d e `` | 使用外部差异比较工具(git difftool) |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 标签

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 复制标签到剪贴板 |  |
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
| `` s c `` | 检出 | 检出选择的标签作为分离的HEAD |
| `` n t `` | 创建标签 | 基于当前提交创建一个新标签。您将在弹窗中输入标签名称和描述(可选)。 |
| `` n w `` | 新建工作树 |  |
| `` <delete> `` | 删除 | 查看本地/远程标签的删除选项 |
| `` p `` | 推送标签 | 推送选择的标签到远端。您将在弹窗中选择一个远端。 |
| `` t d e `` | 使用外部差异比较工具(git difftool) |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 次要

| Key | Action | Info |
|-----|--------|-------------|
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | 切换代码块选择 | 切换逐行选择与代码块选择模式。 |
| `` t h r `` | 切换拖动选择 |  |
| `` e `` | 编辑文件 | 使用外部编辑器打开文件 |
| `` <space>, g h s `` | 切换暂存状态 | 切换行暂存状态 |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | 添加/移除 行到补丁 |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 取消变更(git reset) | 当选择未暂存的变更时，使用git reset丢弃该变更。当选择已暂存的变更时，取消暂存该变更 |
| `` x `` | 从提交中移除行 | 从本次提交中移除所选行。此操作会在后台运行交互式变基，因此如果后续提交也修改了这些行，您可能会遇到合并冲突。 |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | 编辑代码块 | 在外部编辑器中编辑选中的代码块 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | 选择上一个区块 |  |
| `` ], t n h `` | 选择下一个区块 |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | 退出回到侧边面板 |  |
| `` g e n `` | 提交变更而无需预先提交钩子 |  |
| `` g e c `` | 使用 Git 编辑器提交变更 |  |
| `` / `` | 开始搜索 |  |

## 正在合并

| Key | Action | Info |
|-----|--------|-------------|
| `` <space>, g h p `` | 选中区块 |  |
| `` g h b `` | Pick both hunks |  |
| `` k, <up> `` | 选择顶部块 |  |
| `` j, <down> `` | 选择底部块 |  |
| `` [, t p c `` | 选择上一个冲突 |  |
| `` ], t n c `` | 选择下一个冲突 |  |
| `` u u `` | 撤销 | 撤消上次合并冲突解决 |
| `` e `` | 编辑文件 | 使用外部编辑器打开文件 |
| `` o `` | 打开文件 | 使用默认程序打开该文件 |
| `` g h c `` | 查看合并冲突选项 | 查看用于解决合并冲突的选项。 |
| `` <esc> `` | 返回文件面板 |  |

## 正常

| Key | Action | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | 向下滚动 |  |
| `` <mouse wheel up> (fn+down) `` | 向上滚动 |  |
| `` t n d `` | Switch diff pane | Switch to the other focused diff pane. |
| `` t h s `` | 切换代码块选择 | 切换逐行选择与代码块选择模式。 |
| `` t h r `` | 切换拖动选择 |  |
| `` e `` | 编辑文件 | 使用外部编辑器打开文件 |
| `` <space>, g h s `` | 切换暂存状态 | 切换行暂存状态 |
| `` <space>, g h u `` | Unstage selected changes |  |
| `` <space>, g h t `` | 添加/移除 行到补丁 |  |
| `` <space>, g h t `` | Remove selected lines from custom patch |  |
| `` x `` | 取消变更(git reset) | 当选择未暂存的变更时，使用git reset丢弃该变更。当选择已暂存的变更时，取消暂存该变更 |
| `` x `` | 从提交中移除行 | 从本次提交中移除所选行。此操作会在后台运行交互式变基，因此如果后续提交也修改了这些行，您可能会遇到合并冲突。 |
| `` x `` | Unstage selected changes |  |
| `` g e h `` | 编辑代码块 | 在外部编辑器中编辑选中的代码块 |
| `` y d `` | Copy selected diff lines to clipboard |  |
| `` [, t p h `` | 选择上一个区块 |  |
| `` ], t n h `` | 选择下一个区块 |  |
| `` {, t p f `` | Go to previous file |  |
| `` }, t n f `` | Go to next file |  |
| `` t n j `` | Jump to file |  |
| `` g p v `` | Open pull request at selected line | Open the branch's pull request in your browser, at the line the selection is on, so that you can comment on it there. Only pull requests on GitHub are found. |
| `` <esc> `` | 退出回到侧边面板 |  |
| `` g e n `` | 提交变更而无需预先提交钩子 |  |
| `` g e c `` | 使用 Git 编辑器提交变更 |  |
| `` / `` | 开始搜索 |  |

## 状态

| Key | Action | Info |
|-----|--------|-------------|
| `` s n `` | Open repository by path |  |
| `` t a c `` | 编辑配置文件 | 使用外部编辑器打开文件 |
| `` t a u `` | 检查更新 |  |
| `` s r `` | 切换到最近的仓库 |  |
| `` t l n `` | 显示/循环所有分支日志 |  |
| `` t l p `` | 显示/循环所有分支日志（反向） |  |
| `` v, <space>, 0 `` | 聚焦主视图 |  |

## 确认面板

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 确认 |  |
| `` <esc> `` | 关闭 |  |
| `` <ctrl+o> `` | 复制到剪贴板 |  |

## 菜单

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 执行 |  |
| `` <esc> `` | 关闭 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 贮藏

| Key | Action | Info |
|-----|--------|-------------|
| `` g s a `` | 应用 | 将贮藏项应用到您的工作目录。 |
| `` g s p `` | 应用并删除 | 将存储项应用到工作目录并删除存储项。 |
| `` <delete> `` | 删除 | 从贮藏列表中删除该贮藏项 |
| `` n w `` | 新建工作树 |  |
| `` <f2> `` | 重命名贮藏 |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交的文件 |  |
| `` / `` | 通过文本过滤当前视图 |  |

## 输入提示

| Key | Action | Info |
|-----|--------|-------------|
| `` <enter> `` | 确认 |  |
| `` <esc> `` | 关闭 |  |

## 远程

| Key | Action | Info |
|-----|--------|-------------|
| `` v, <space> `` | 查看分支 |  |
| `` n r `` | 添加新的远程仓库 |  |
| `` <delete> `` | 删除 | 删除选中的远程。从远程跟踪远程分支的任何本地分支都不会受到影响。 |
| `` e `` | 编辑(Edit) | 编辑远程仓库 |
| `` f `` | 抓取 | 抓取远程仓库 |
| `` n f `` | 添加复刻远程仓库 | 通过替换 origin URL 中的所有者来快速添加复刻远程仓库，并可选择从新远程仓库检出分支。 |
| `` / `` | 通过文本过滤当前视图 |  |

## 远程分支

| Key | Action | Info |
|-----|--------|-------------|
| `` y n `` | 复制分支名称到剪贴板 |  |
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
| `` s c `` | 检出 | 基于当前选中的远程分支检出一个新的本地分支，或者将远程分支作分离的HEAD。 |
| `` n w `` | 新建工作树 |  |
| `` m `` | 合并到当前检出的分支 | 查看将选中项合并到当前分支的选项(正常合并，压缩合并) |
| `` r `` | Rebase current onto selected branch… | 将检出的分支变基到所选的分支上。 |
| `` <delete> `` | 删除 | 从远程删除远程分支。 |
| `` g t `` | 设置为上游 | 设置为检出分支的上游 |
| `` t s o `` | 排序 |  |
| `` t d e `` | 使用外部差异比较工具(git difftool) |  |
| `` 0 `` | 聚焦主视图 |  |
| `` v, <space> `` | 查看提交 |  |
| `` / `` | 通过文本过滤当前视图 |  |
