package git_commands

import (
	"fmt"
	"slices"
	"strings"
)

// BranchActionDeleteContext retains inspection inputs while the user chooses
// local/remote refs. It authorizes no deletion and permits occupied sources to
// be inspected for a remote-only action.
type BranchActionDeleteContext struct {
	owner           *GitCommon
	targets         []BranchActionTarget
	remotes         []string
	configSignature string
	prepared        bool
}

func (context BranchActionDeleteContext) Targets() []BranchActionTarget {
	return slices.Clone(context.targets)
}

func (context BranchActionDeleteContext) Remotes() []string { return slices.Clone(context.remotes) }

func (self *BranchCommands) PrepareBranchActionDeleteContext(targets []BranchActionTarget) (BranchActionDeleteContext, error) {
	if len(targets) == 0 {
		return BranchActionDeleteContext{}, fmt.Errorf("select at least one branch")
	}
	context := BranchActionDeleteContext{owner: self.GitCommon, targets: slices.Clone(targets)}
	for _, target := range targets {
		if err := self.CheckBranchActionTarget(target); err != nil {
			return BranchActionDeleteContext{}, err
		}
		if target.HeadRef() != targets[0].HeadRef() || target.HeadOID() != targets[0].HeadOID() {
			return BranchActionDeleteContext{}, ErrBranchActionChanged
		}
	}
	entries, err := (&SyncCommands{GitCommon: self.GitCommon}).branchSyncConfigEntries()
	if err != nil {
		return BranchActionDeleteContext{}, err
	}
	for _, entry := range entries {
		rest, remote := strings.CutPrefix(entry.key, "remote.")
		if !remote || !entry.hasValue || entry.value == "" {
			continue
		}
		name, variable := "", ""
		if dot := strings.LastIndexByte(rest, '.'); dot >= 0 {
			name, variable = rest[:dot], rest[dot+1:]
		}
		if name != "" && name != "." && !strings.ContainsAny(name, "\x00\r\n:") && (variable == "url" || variable == "pushurl") {
			context.remotes = append(context.remotes, name)
		}
	}
	slices.Sort(context.remotes)
	context.remotes = slices.Compact(context.remotes)
	context.configSignature, err = self.deleteConfigSignature()
	if err != nil {
		return BranchActionDeleteContext{}, err
	}
	for _, target := range targets {
		if err := self.CheckBranchActionTarget(target); err != nil {
			return BranchActionDeleteContext{}, err
		}
	}
	context.prepared = true
	return context, nil
}

func (self *BranchCommands) CheckBranchActionDeleteContext(context BranchActionDeleteContext) error {
	if !context.prepared || context.owner != self.GitCommon {
		return fmt.Errorf("delete context was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchActionDeleteContext(context.targets)
	if err != nil {
		return err
	}
	if fresh.configSignature != context.configSignature || !slices.Equal(fresh.remotes, context.remotes) {
		return ErrBranchActionChanged
	}
	return nil
}
