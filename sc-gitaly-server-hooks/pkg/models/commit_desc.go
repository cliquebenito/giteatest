package models

import "strings"

type CommitDescriptor struct {
	ParentCommitSha string `json:"parent_commit_sha"`
	ChildCommitSha  string `json:"child_commit_sha"`
	RefName         string `json:"ref_name"`
}

func (c CommitDescriptor) IsHUINYA() bool {
	if !strings.HasPrefix(c.RefName, BranchPrefix) && strings.HasPrefix(c.RefName, TagPrefix) {
		return true
	}
	return false
}
