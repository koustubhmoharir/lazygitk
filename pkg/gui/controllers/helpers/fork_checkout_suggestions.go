package helpers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: name suggestions for the named checkout of one commit. Local branches
// already at that commit come first, then the other branch names as in
// GetBranchNameSuggestionsFunc. Each name appears once.
func (self *SuggestionsHelper) GetBranchNameSuggestionsAtCommitFunc(oid string) func(string) []*types.Suggestion {
	names := self.branchNamesAtCommitFirst(oid)

	return func(input string) []*types.Suggestion {
		matching := names
		if input != "" {
			matching = utils.FilterStrings(input, names, self.c.UserConfig().Gui.UseFuzzySearch())
		}
		return lo.Map(matching, func(branchName string, _ int) *types.Suggestion {
			return &types.Suggestion{
				Value: branchName,
				Label: presentation.GetBranchTextStyle(branchName).Sprint(branchName),
			}
		})
	}
}

func (self *SuggestionsHelper) branchNamesAtCommitFirst(oid string) []string {
	seen := map[string]bool{}
	var names []string
	for _, branch := range self.c.Model().Branches {
		// The detached HEAD entry is not a branch, so it is never listed first.
		if !branch.DetachedHead && branch.CommitHash == oid && !seen[branch.Name] {
			seen[branch.Name] = true
			names = append(names, branch.Name)
		}
	}
	for _, name := range self.getBranchNames() {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}
