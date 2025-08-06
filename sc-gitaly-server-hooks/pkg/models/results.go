package models

import (
	"fmt"
	"os"
)

// HookPostReceiveResult represents an individual result from PostReceive
type HookPostReceiveResult struct {
	Results      []HookPostReceiveBranchResult
	RepoWasEmpty bool
	Err          string
}

// HookPostReceiveBranchResult represents an individual branch result from PostReceive
type HookPostReceiveBranchResult struct {
	Message bool
	Create  bool
	Branch  string
	URL     string
}

// HookProcReceiveResult represents an individual result from ProcReceive
type HookProcReceiveResult struct {
	Results []hookProcReceiveRefResult
	Err     string
}

// hookProcReceiveRefResult represents an individual result from ProcReceive
type hookProcReceiveRefResult struct {
	OldOID       string
	NewOID       string
	Ref          string
	OriginalRef  string
	IsForcePush  bool
	IsNotMatched bool
	Err          string
}

func (r HookPostReceiveBranchResult) Print() error {
	if !r.Message {
		return nil
	}

	_, err := fmt.Fprintln(os.Stderr, "")
	if err != nil {
		return fmt.Errorf("cannot print results: %w", err)
	}
	if r.Create {
		_, err = fmt.Fprintf(os.Stderr, "\nCreate a new pull request for '%s':\n  %s\n\n", r.Branch, r.URL)
		if err != nil {
			return fmt.Errorf("cannot print results: %w", err)
		}
	} else {
		_, err = fmt.Fprintf(os.Stderr, "\nVisit the existing pull request:\n  %s\n\n", r.URL)
		if err != nil {
			return fmt.Errorf("cannot print results: %w", err)
		}
	}
	err = os.Stderr.Sync()
	if err != nil {
		return fmt.Errorf("cannot print results: %w", err)
	}
	return nil
}
