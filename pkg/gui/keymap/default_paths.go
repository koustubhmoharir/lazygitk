package keymap

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// defaultPaths contains fork-owned normal-command routes. Paths are compiled
// once from config labels and normalized for command-key matching.
var defaultPaths = buildDefaultPaths()

func buildDefaultPaths() map[string][][]gocui.Key {
	paths := map[string][][]gocui.Key{}
	add := func(ids, labels string) {
		path := make([]gocui.Key, 0, strings.Count(labels, " ")+1)
		for _, label := range strings.Fields(labels) {
			key, ok := config.KeyFromLabel(label)
			if !ok || !key.IsSet() {
				panic("invalid internal default key label: " + label)
			}
			path = append(path, gocui.NormalizeCommandKey(key))
		}
		for _, id := range strings.Fields(ids) {
			paths[id] = append(paths[id], append([]gocui.Key(nil), path...))
		}
	}

	// Core actions and context-specific aliases.
	add("diff.commit.worktree diff.history-file.worktree", "d w")
	add("diff.commit.previous diff.history-file.previous", "d p")
	add("diff.commit.selected-commit diff.history-file.selected-commit", "d c")
	add("tag.push branch.push-current branch.push-selected", "p")
	add("remote.fetch repo.fetch", "f")
	add("branch.rename stash.rename commit.reword", "f2")
	add("file.edit file.edit-line commit.edit remote.edit-url submodule.edit-url", "e")
	add("file.open worktree.open-editor commit.open-browser", "o")
	add("app.refresh", "f5")
	add("branch.delete-menu remote-branch.delete remote.delete tag.delete-menu stash.drop commit.drop submodule.delete worktree.remove-menu", "delete")
	add("file.discard-menu commit.discard-file-changes diff.discard-worktree-selection commit.discard-selection", "x")
	add("commit.amend-head commit.amend-selected", "a")
	add("branch.rebase-current-onto-selected branch.rebase-current-onto-selected-commit", "r")
	add("branch.merge-selected-into-current", "m")
	add("branch.reset.hard", "s h")
	add("branch.reset.mixed", "s m")
	add("branch.reset.soft", "s s")
	add("branch.checkout remote-branch.checkout tag.checkout commit.checkout-menu", "s c")
	add("branch.place-at-selected-commit", "n b")
	add("commit.create", "enter")
	add("file.ignore-menu", "i")
	add("branch.reflog-undo conflict.undo", "u u")
	add("branch.reflog-redo", "u r")
	add("app.quit", "q")
	add("app.command-palette", "?")
	add("search.open commit.search", "/")
	add("app.shell-command", ":")
	add("ui.screen.next-mode", "=")
	add("ui.screen.next-mode", "+")
	add("ui.screen.previous-mode", "-")
	add("file.toggle-stage", "space")
	add("commit.mark-cherry-pick", "c y")
	add("commit.unmark-cherry-pick", "c x")
	add("commit.cherry-pick", "c p")
	add("commit.combine-selected", "g c")
	// Fork: one flat copy menu. Each second key has the same meaning in every
	// panel that offers it. Nested copy menus keep no default route.
	add("clipboard.copy-branch-name clipboard.copy-tag-name clipboard.copy-submodule-name clipboard.copy-file-name", "y n")
	add("clipboard.copy-file-path", "y p")
	add("clipboard.copy-absolute-file-path", "y f")
	add("clipboard.copy-history-file-content", "y c")
	add("clipboard.copy-commit-hash", "y h")
	add("clipboard.copy-abbreviated-commit-hash", "y i")
	add("clipboard.copy-commit-subject", "y s")
	add("clipboard.copy-commit-message", "y m")
	add("clipboard.copy-commit-body", "y b")
	add("clipboard.copy-commit-url", "y u")
	add("clipboard.copy-commit-author", "y a")
	add("clipboard.copy-commit-tags", "y t")
	add("clipboard.copy-commit-diff clipboard.copy-working-file-diff clipboard.copy-history-file-diff clipboard.copy-diff-selection", "y d")
	add("clipboard.copy-working-tree-diff", "y w")
	add("branch.copy-pull-request-url", "y r")
	add("clipboard.copy-confirmation", "y y")
	add("tag.create-at-selection tag.create-at-current", "n t")
	add("worktree.create worktree.create-menu", "n w")
	add("remote.create", "n r")
	add("remote.add-fork", "n f")
	add("submodule.create", "n s")
	add("history-file.checkout", "g v f")
	add("branch.pull-current branch.pull-selected", "g l")
	add("branch.fast-forward-selected", "g f")
	add("branch.upstream-menu branch.set-upstream-to-selection", "g t")
	add("commit.rebase-interactive", "g i")
	add("bisect.options-menu", "g b")
	add("commit.revert", "g v c")
	add("stash.apply", "g a")
	add("branch.merge-rebase-menu", "g o")
	add("branch.git-flow-menu", "g y")
	add("commit.mark-rebase-base", "g r b")
	add("commit.mark-pick", "g r p")
	add("branch.move-commits-to-new", "g m n")
	add("commit.move-up", "g m u")
	add("commit.move-down", "g m d")
	add("commit.create-with-editor", "g e c")
	add("commit.create-without-hooks", "g e n")
	add("commit.reword-with-editor", "g e r")
	add("commit.edit-attributes-menu", "g e a")
	add("diff.edit-hunk", "g e h")
	add("branch.create-pull-request", "g p c")
	add("branch.pull-request-menu", "g p o")
	add("branch.open-pull-request commit.open-pull-request diff.open-line-pull-request", "g p v")
	add("stash.create", "g s s")
	add("stash.create-menu", "g s o")
	add("stash.pop", "g s p")
	add("diff.stage-selection", "g h s")
	add("diff.unstage-selection", "g h u")
	add("patch.toggle-file-selection patch.toggle-selection patch.remove-selection", "g h t")
	add("patch.toggle-all-file-selection", "g h a")
	add("patch.options-menu", "g h o")
	add("conflict.options-menu", "g h c")
	add("conflict.pick-hunk", "g h p")
	add("conflict.pick-both-hunks", "g h b")
	add("submodule.update", "g u s")
	add("submodule.initialize", "g u i")
	add("submodule.bulk-menu", "g u b")
	add("branch.reset-upstream-menu", "g z u")
	add("worktree.discard-menu", "g z w")
	add("remote.view-branches navigation.open-commits navigation.open-diff-files navigation.open-history-file-diff navigation.focus-diff file.enter", "v")
	// Fork: repository switching lives under s (roadmap item 14).
	add("repo.switch-menu", "s r")
	add("repo.open-path", "s n")
	add("worktree.switch", "t r w")
	add("submodule.enter", "t r s")
	add("diff.commit.external diff.history-file.external diff.ref.external diff.working-file.external", "t d e")
	add("diff.history-file.working-file", "t d f")
	add("diff.options-menu", "t d c")
	add("diff.toggle-whitespace", "t d w")
	add("diff.renderer.next", "t d n")
	add("diff.renderer.previous", "t d p")
	add("diff.context.increase", "t d +")
	add("diff.context.decrease", "t d -")
	add("diff.rename-threshold.increase", "t d >")
	add("diff.rename-threshold.decrease", "t d <")
	add("filter.open", "t f t")
	add("file.filter-menu", "t f s")
	add("filter.options-menu", "t f o")
	add("ui.file-tree.toggle", "t t t")
	add("ui.file-tree.collapse-all", "t t c")
	add("ui.file-tree.expand-all", "t t e")
	add("history.all-branches.next", "t l n")
	add("history.all-branches.previous", "t l p")
	add("commit.log-menu", "t l o")
	add("command-log.options-menu", "t l c")
	add("navigation.focus-command-log", "t l f")
	add("commit.select-current-branch-history", "t l s")
	add("branch.sort-menu remote-branch.sort-menu", "t s o")
	add("config.edit-menu", "t a c")
	add("app.check-update", "t a u")
	add("app.suspend", "t a s")
	add("app.quit-without-cd", "t a q")
	add("game.snake", "t a g")
	add("navigation.diff.next-hunk", "t n h")
	add("navigation.diff.previous-hunk", "t p h")
	add("navigation.diff.next-file", "t n f")
	add("navigation.diff.previous-file", "t p f")
	add("navigation.conflict.next", "t n c")
	add("navigation.conflict.previous", "t p c")
	add("navigation.diff.switch-pane", "t n d")
	add("diff.jump-to-file-menu", "t n j")
	add("selection.diff.toggle-hunk", "t h s")
	add("selection.diff.toggle-range selection.list.toggle-range", "t h r")

	// Space outside Files shares the inspection route; the main controller
	// projects the contextual route to the appropriate command.
	for _, id := range []string{"file.enter", "remote.view-branches", "navigation.open-commits", "navigation.open-diff-files", "navigation.open-history-file-diff", "navigation.focus-diff"} {
		add(id, "space")
	}

	// Replaced normal routes, and nested copy menus that moved into the flat y
	// menu, intentionally remain recognized with no path.
	for _, id := range strings.Fields("file.refresh file.copy-menu history-file.copy-menu commit.copy-attribute-menu commit.copy-for-cherry-pick commit.squash-down commit.mark-fixup commit.set-fixup-message commit.create-fixup-menu commit.autosquash commit.find-fixup-target commit.clear-cherry-pick-selection branch.create-and-checkout branch.reset-menu branch.checkout-by-name branch.checkout-previous branch.force-checkout file.toggle-stage-all diff.primary-selection diff.discard-selection") {
		if _, exists := paths[id]; !exists {
			paths[id] = nil
		}
	}

	return paths
}

// DefaultPaths returns cloned default normal-command paths for a known ID.
func DefaultPaths(commandID string) ([][]gocui.Key, bool) {
	paths, ok := defaultPaths[commandID]
	if !ok {
		return nil, false
	}
	cloned := make([][]gocui.Key, len(paths))
	for i := range paths {
		cloned[i] = append([]gocui.Key(nil), paths[i]...)
	}
	return cloned, true
}
