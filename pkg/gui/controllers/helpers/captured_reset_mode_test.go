package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type capturedResetTestUI struct {
	*namedCheckoutFakeIGuiCommon
	menus []types.CreateMenuOptions
	state *capturedResetTestState
}

type capturedResetTestState struct {
	types.IStateAccessor
	generation int
}

func (state *capturedResetTestState) GetRepoGeneration() int { return state.generation }

func (ui *capturedResetTestUI) State() types.IStateAccessor { return ui.state }

func (ui *capturedResetTestUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

func resetModeFixture(t *testing.T) (*namedCheckoutFixture, *capturedResetTestUI) {
	t.Helper()
	f := newNamedCheckoutFixture(t)
	ui := &capturedResetTestUI{namedCheckoutFakeIGuiCommon: f.ui, state: &capturedResetTestState{}}
	f.c = &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}
	return f, ui
}

func resetModeBegin(t *testing.T, f *namedCheckoutFixture, ui *capturedResetTestUI, mode string) types.ConfirmOpts {
	t.Helper()
	if err := ResetCurrentToCapturedSourceMode(f.c, "captured source", f.oid, mode); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirms) != 1 {
		t.Fatalf("confirm count = %d, want 1", len(ui.confirms))
	}
	if len(ui.menus) != 0 {
		t.Fatal("direct mode unexpectedly opened a mode menu")
	}
	return ui.confirms[0]
}

type resetModeSnapshot struct{ head, index, worktree string }

func resetModeState(t *testing.T, root string) resetModeSnapshot {
	t.Helper()
	head := namedCheckoutGit(t, root, "rev-parse", "HEAD")
	index := namedCheckoutGit(t, root, "write-tree")
	worktree, err := os.ReadFile(filepath.Join(root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	return resetModeSnapshot{head: head, index: index, worktree: string(worktree)}
}

func TestCapturedResetMode(t *testing.T) {
	t.Run("confirmation and mode effects", func(t *testing.T) {
		for _, mode := range []string{"hard", "mixed", "soft"} {
			t.Run(mode, func(t *testing.T) {
				f, ui := resetModeFixture(t)
				namedCheckoutWrite(t, f.root, "tracked", "working change\n")
				namedCheckoutGit(t, f.root, "add", "tracked")
				before := resetModeState(t, f.root)
				confirm := resetModeBegin(t, f, ui, mode)
				if !strings.Contains(confirm.Prompt, f.oid) || !strings.Contains(confirm.Prompt, before.head) || !strings.Contains(confirm.Prompt, "captured source") {
					t.Fatalf("confirmation omitted captured endpoints: %q", confirm.Prompt)
				}
				afterPrompt := resetModeState(t, f.root)
				if afterPrompt != before {
					t.Fatal("reset mutated state before confirmation")
				}
				if err := confirm.HandleConfirm(); err != nil {
					t.Fatal(err)
				}
				if err := ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				got := resetModeState(t, f.root)
				if got.head != f.oid {
					t.Fatalf("HEAD = %s, want captured source %s", got.head, f.oid)
				}
				switch mode {
				case "soft":
					if got.index != before.index || got.worktree != before.worktree {
						t.Fatal("soft reset changed index or worktree")
					}
				case "mixed":
					if got.index == before.index || got.worktree != before.worktree {
						t.Fatal("mixed reset did not preserve only worktree changes")
					}
				case "hard":
					if got.worktree == before.worktree {
						t.Fatal("hard reset preserved worktree changes")
					}
				}
				if err := confirm.HandleConfirm(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
					t.Fatalf("replayed confirmation error = %v", err)
				}
			})
		}
	})

	t.Run("cancel leaves repository unchanged", func(t *testing.T) {
		for _, mode := range []string{"hard", "mixed", "soft"} {
			t.Run(mode, func(t *testing.T) {
				f, ui := resetModeFixture(t)
				before := resetModeState(t, f.root)
				confirm := resetModeBegin(t, f, ui, mode)
				if err := confirm.HandleClose(); err != nil {
					t.Fatal(err)
				}
				after := resetModeState(t, f.root)
				if before != after {
					t.Fatal("canceled reset changed repository")
				}
			})
		}
	})

	t.Run("captured source change refused", func(t *testing.T) {
		f, ui := resetModeFixture(t)
		confirm := resetModeBegin(t, f, ui, "soft")
		namedCheckoutGit(t, f.root, "commit", "--allow-empty", "-qm", "concurrent")
		if err := confirm.HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
			t.Fatalf("source change error = %v", err)
		}
	})

	t.Run("captured source ref change refused", func(t *testing.T) {
		f, ui := resetModeFixture(t)
		namedCheckoutGit(t, f.root, "branch", "captured-source", f.oid)
		if err := ResetCurrentToCapturedSourceMode(f.c, "captured source", "refs/heads/captured-source", "soft"); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		confirm := ui.confirms[0]
		namedCheckoutGit(t, f.root, "update-ref", "refs/heads/captured-source", f.other)
		if err := confirm.HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); !errors.Is(err, git_commands.ErrBranchActionChanged) {
			t.Fatalf("source ref change error = %v", err)
		}
	})

	t.Run("repository replacement refused", func(t *testing.T) {
		f, ui := resetModeFixture(t)
		confirm := resetModeBegin(t, f, ui, "soft")
		ui.git = nil
		if err := confirm.HandleConfirm(); err == nil {
			t.Fatal("repository replacement accepted")
		}
		if len(ui.workers) != 0 {
			t.Fatal("stale confirm queued mutation")
		}
	})

	t.Run("invalid mode refuses before Git", func(t *testing.T) {
		c := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: &capturedResetTestUI{namedCheckoutFakeIGuiCommon: &namedCheckoutFakeIGuiCommon{}}}
		if err := ResetCurrentToCapturedSourceMode(c, "source", "HEAD", "merge"); err == nil {
			t.Fatal("invalid mode accepted")
		}
	})
}
