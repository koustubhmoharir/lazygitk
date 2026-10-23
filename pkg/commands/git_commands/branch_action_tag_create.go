package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionTagCreatePlan is an opaque snapshot for creating one native tag.
type BranchActionTagCreatePlan struct {
	owner                                                  *GitCommon
	target                                                 BranchActionTarget
	name, description, configSignature                     string
	oldOID                                                 string
	exists, annotated, signed, overwriteApproved, prepared bool
}

func (p BranchActionTagCreatePlan) Target() BranchActionTarget { return p.target }
func (p BranchActionTagCreatePlan) Name() string               { return p.name }
func (p BranchActionTagCreatePlan) Description() string        { return p.description }
func (p BranchActionTagCreatePlan) OldOID() string             { return p.oldOID }
func (p BranchActionTagCreatePlan) Annotated() bool            { return p.annotated }
func (p BranchActionTagCreatePlan) Signed() bool               { return p.signed }
func (p BranchActionTagCreatePlan) DestinationExists() bool    { return p.exists }
func (p BranchActionTagCreatePlan) WithOverwriteApproved() BranchActionTagCreatePlan {
	p.overwriteApproved = true
	return p
}

// PrepareBranchActionTagCreate captures all inputs needed to create a tag.
func (self *TagCommands) PrepareBranchActionTagCreate(target BranchActionTarget, name, description string) (BranchActionTagCreatePlan, error) {
	b := NewBranchCommands(self.GitCommon)
	if err := b.CheckBranchActionTarget(target); err != nil {
		return BranchActionTagCreatePlan{}, err
	}
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return BranchActionTagCreatePlan{}, fmt.Errorf("invalid literal tag name")
	}
	if strings.ContainsRune(description, '\x00') {
		return BranchActionTagCreatePlan{}, fmt.Errorf("tag description contains NUL")
	}
	ref := "refs/tags/" + name
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return BranchActionTagCreatePlan{}, fmt.Errorf("invalid tag name: %w", err)
	}
	state, err := b.readPlacementRef(ref)
	if err != nil {
		return BranchActionTagCreatePlan{}, err
	}
	config, err := b.deleteConfigSignature()
	if err != nil {
		return BranchActionTagCreatePlan{}, err
	}
	value, err := self.cmd.New(NewGitCmd("config").Arg("--type=bool", "--get", "tag.gpgSign").ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	sign := false
	if err == nil {
		switch strings.TrimSpace(value) {
		case "true":
			sign = true
		case "false":
		default:
			return BranchActionTagCreatePlan{}, fmt.Errorf("Git returned an invalid tag.gpgSign boolean")
		}
	} else if !commandExitedWithOne(err) {
		return BranchActionTagCreatePlan{}, err
	}
	if state.symbolic != "" {
		return BranchActionTagCreatePlan{}, fmt.Errorf("tag destination is a symbolic ref")
	}
	return BranchActionTagCreatePlan{owner: self.GitCommon, target: target, name: name, description: description, configSignature: config, oldOID: state.oid, exists: state.exists, annotated: description != "" || sign, signed: sign, prepared: true}, nil
}

// BranchActionTagCreateCommand revalidates the reviewed snapshot before making one native Git command.
func (self *TagCommands) BranchActionTagCreateCommand(plan BranchActionTagCreatePlan) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("tag creation plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchActionTagCreate(plan.target, plan.name, plan.description)
	if err != nil {
		return nil, err
	}
	if fresh.oldOID != plan.oldOID || fresh.exists != plan.exists || fresh.annotated != plan.annotated || fresh.signed != plan.signed || fresh.configSignature != plan.configSignature {
		return nil, ErrBranchActionChanged
	}
	if plan.exists && !plan.overwriteApproved {
		return nil, fmt.Errorf("overwriting a tag requires explicit approval")
	}
	args := NewGitCmd("tag").ArgIf(plan.annotated, "-m", plan.description).ArgIf(plan.exists && plan.overwriteApproved, "--force").Arg("--", plan.name, plan.target.sourceOID+"^{commit}").ToArgv()
	return self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}
