package models

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"sc-gitaly-server-hooks/pkg/readers/env"
)

type HookRequestOptions struct {
	OwnerName string
	RepoName  string
	Opts      *HookOptions
}

func NewHookRequestOptions(ownerName, repoName string, opts *HookOptions) *HookRequestOptions {
	return &HookRequestOptions{
		OwnerName: ownerName,
		RepoName:  repoName,
		Opts:      opts,
	}
}

// gitPushOptions is a wrapper around a map[string]string
type gitPushOptions map[string]string

// Bool checks for a key in the map and parses as a boolean
func (g gitPushOptions) Bool(key string, def bool) bool {
	if val, ok := g[key]; ok {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return def
}

func pushOptions() map[string]string {
	opts := make(map[string]string)
	if pushCount, err := strconv.Atoi(os.Getenv(GitPushOptionCount)); err == nil {
		for idx := 0; idx < pushCount; idx++ {
			opt := os.Getenv(fmt.Sprintf("GIT_PUSH_OPTION_%d", idx))
			kv := strings.SplitN(opt, "=", 2)
			if len(kv) == 2 {
				opts[kv[0]] = kv[1]
			}
		}
	}
	return opts
}

// HookOptions represents the options for the Hook calls
type HookOptions struct {
	OldCommitIDs                    []string
	NewCommitIDs                    []string
	RefFullNames                    []string
	UserID                          int64
	UserName                        string
	GitObjectDirectory              string
	GitAlternativeObjectDirectories string
	GitQuarantinePath               string
	GitPushOptions                  gitPushOptions
	PullRequestID                   int64
	DeployKeyID                     int64 // if the pusher is a DeployKey, then UserID is the repo's org user.
	IsWiki                          bool
	ActionPerm                      int
}

func NewHookOptions() *HookOptions {
	return &HookOptions{}
}

func NewHookOptionsWithUserInfo(userId int64, userName string) *HookOptions {
	return &HookOptions{
		UserName:     userName,
		UserID:       userId,
		OldCommitIDs: make([]string, 0, HookBatchSize),
		NewCommitIDs: make([]string, 0, HookBatchSize),
		RefFullNames: make([]string, 0, HookBatchSize),
	}
}

func NewHookOptionsWithCommitInfo(userId int64, commitDesc []CommitDescriptor) *HookOptions {
	options := &HookOptions{
		UserID:       userId,
		OldCommitIDs: make([]string, 0, HookBatchSize),
		NewCommitIDs: make([]string, 0, HookBatchSize),
		RefFullNames: make([]string, 0, HookBatchSize),
	}
	options.setCommitDescription(commitDesc)

	return options
}

func (o *HookOptions) SetGitOptionsFromEnv(reader env.Reader) error {
	o.GitAlternativeObjectDirectories = reader.GetByKeyWithoutError(GitAlternativeObjectDirectories)
	o.GitObjectDirectory = reader.GetByKeyWithoutError(GitObjectDirectory)
	o.GitQuarantinePath = reader.GetByKeyWithoutError(GitQuarantinePath)
	o.GitPushOptions = pushOptions()

	return nil
}

func (o *HookOptions) setCommitDescription(commitDescs []CommitDescriptor) {
	for i, commitDesc := range commitDescs {
		if i > HookBatchSize {
			break
		}

		o.OldCommitIDs = append(o.OldCommitIDs, commitDesc.ParentCommitSha)
		o.NewCommitIDs = append(o.NewCommitIDs, commitDesc.ChildCommitSha)
		o.RefFullNames = append(o.RefFullNames, commitDesc.RefName)
	}
}
