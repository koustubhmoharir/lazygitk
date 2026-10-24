package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type selectedRefCheckoutState struct {
	types.IStateAccessor
	generation int
}

func (s *selectedRefCheckoutState) GetRepoGeneration() int { return s.generation }

type selectedRefCheckoutUI struct {
	*namedCheckoutFakeIGuiCommon
	state selectedRefCheckoutState
	menus []types.CreateMenuOptions
}

func (ui *selectedRefCheckoutUI) State() types.IStateAccessor { return &ui.state }
func (ui *selectedRefCheckoutUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

func selectedRefCheckoutMenu(t *testing.T, f *namedCheckoutFixture, source models.Ref) (*selectedRefCheckoutUI, types.CreateMenuOptions) {
	t.Helper()
	ui := &selectedRefCheckoutUI{namedCheckoutFakeIGuiCommon: f.ui}
	f.c.IGuiCommon = ui
	if err := (&RefsHelper{c: f.c}).CreateSelectedRefCheckoutMenu(source); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 1 {
		t.Fatal("missing checkout options")
	}
	return ui, ui.menus[0]
}

func TestSelectedRefCheckoutCapturesBranchAndOffersNaming(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	branch := &models.Branch{Name: "other"}
	ui, menu := selectedRefCheckoutMenu(t, f, branch)
	if len(menu.Items) != 3 || menu.Items[0].CommandID != "branch.checkout" || menu.Items[1].CommandID != "branch.checkout-at-selected-commit" {
		t.Fatal("missing selected branch and independent named checkout options")
	}
	branch.Name = "main"
	namedCheckoutGit(t, f.root, "branch", "-f", "other", f.oid)
	if err := menu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirms) != 1 || namedCheckoutGit(t, f.root, "rev-parse", "HEAD") != f.oid {
		t.Fatal("a moved selected branch must require approval before any checkout")
	}
}

func TestSelectedRefCheckoutRemoteHasNamedAndDetachedOptions(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.other)
	ui, menu := selectedRefCheckoutMenu(t, f, &models.RemoteBranch{RemoteName: "origin", Name: "topic"})
	if len(menu.Items) != 2 || menu.Items[0].CommandID != "branch.checkout-at-selected-commit" {
		t.Fatal("remote checkout must offer naming rather than silently choose a local branch")
	}
	if err := menu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	ui.state.generation++
	if err := ui.prompts[0].HandleConfirm("new-topic"); err == nil {
		t.Fatal("pending name prompt followed a repository change")
	}
}

func TestSelectedRefCheckoutDetachedCarriesHarmlessDirtyFiles(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	ui, menu := selectedRefCheckoutMenu(t, f, &models.Branch{Name: "other"})
	namedCheckoutWrite(t, f.root, "carry", "local edits\n")
	namedCheckoutWrite(t, f.root, "untracked", "local untracked\n")
	if err := menu.Items[2].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if namedCheckoutGit(t, f.root, "rev-parse", "HEAD") != f.other || namedCheckoutGit(t, f.root, "rev-parse", "--abbrev-ref", "HEAD") != "HEAD" {
		t.Fatal("checkout did not detach at the captured commit")
	}
	for name, want := range map[string]string{"carry": "local edits\n", "untracked": "local untracked\n"} {
		content, err := os.ReadFile(filepath.Join(f.root, name))
		if err != nil || string(content) != want {
			t.Fatalf("lost local %s: %q, %v", name, content, err)
		}
	}
	if namedCheckoutGit(t, f.root, "stash", "list") != "" {
		t.Fatal("checkout used a stash")
	}
}

func TestSelectedRefCheckoutDetachedRefusesOverwrites(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	ui, menu := selectedRefCheckoutMenu(t, f, &models.Branch{Name: "other"})
	namedCheckoutWrite(t, f.root, "tracked", "local edits\n")
	if err := menu.Items[2].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err == nil {
		t.Fatal("checkout overwrote a dirty path")
	}
	if namedCheckoutGit(t, f.root, "rev-parse", "HEAD") != f.oid || namedCheckoutGit(t, f.root, "stash", "list") != "" {
		t.Fatal("failed checkout moved HEAD or stashed changes")
	}
}
