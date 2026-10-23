package git_commands

import "slices"

// branchActionMergePolicy is the caller-independent interpretation of the
// merge options that affect merge planning and message construction.
type branchActionMergePolicy struct {
	ff                      string
	squash                  bool
	commit                  bool
	invalidCombination      bool
	strategies              []string
	strategyOptions         []string
	allowUnrelatedHistories bool
	customMessage           bool
}

func resolveBranchActionMergePolicy(defaultFF string, defaultStrategies []string, records []branchActionMergeArgument) branchActionMergePolicy {
	policy := branchActionMergePolicy{
		ff:         defaultFF,
		commit:     true,
		strategies: slices.Clone(defaultStrategies),
	}
	commitOption := -1 // Git's default is resolved after squash parsing.
	explicitStrategy := false
	for _, record := range records {
		switch record.Name {
		case "--ff":
			policy.ff = "true"
		case "--no-ff":
			policy.ff = "false"
		case "--ff-only":
			policy.ff = "only"
		case "--squash":
			policy.squash = true
		case "--no-squash":
			policy.squash = false
		case "--commit":
			commitOption = 1
		case "--no-commit":
			commitOption = 0
		case "--strategy":
			if !explicitStrategy {
				policy.strategies = nil
				explicitStrategy = true
			}
			policy.strategies = append(policy.strategies, record.Value)
		case "--no-strategy":
			// Git's option callback is a no-op for this negated spelling.
		case "--strategy-option":
			policy.strategyOptions = append(policy.strategyOptions, record.Value)
		case "--no-strategy-option":
			policy.strategyOptions = nil
		case "--allow-unrelated-histories":
			policy.allowUnrelatedHistories = true
		case "--no-allow-unrelated-histories":
			policy.allowUnrelatedHistories = false
		case "--message", "--file":
			policy.customMessage = true
		}
	}
	policy.invalidCombination = policy.squash && (commitOption > 0 || policy.ff == "false")
	if policy.squash {
		policy.commit = false
	} else if commitOption >= 0 {
		policy.commit = commitOption == 1
	}
	return policy
}
