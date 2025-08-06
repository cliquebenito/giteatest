// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/lfs"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	asymkey_service "code.gitea.io/gitea/services/asymkey"

	stdcharset "golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// IdentityOptions for a person's identity like an author or committer
type IdentityOptions struct {
	Name  string
	Email string
}

// CommitDateOptions store dates for GIT_AUTHOR_DATE and GIT_COMMITTER_DATE
type CommitDateOptions struct {
	Author    time.Time
	Committer time.Time
}

// UpdateRepoFileOptions holds the repository file update options
type UpdateRepoFileOptions struct {
	LastCommitID string
	OldBranch    string
	NewBranch    string
	TreePath     string
	FromTreePath string
	Message      string
	Content      string
	SHA          string
	IsNewFile    bool
	Author       *IdentityOptions
	Committer    *IdentityOptions
	Dates        *CommitDateOptions
	Signoff      bool
}

func detectEncodingAndBOM(entry *git.TreeEntry, repo *repo_model.Repository) (string, bool) {
	reader, err := entry.Blob().DataAsync()
	if err != nil {
		// return default
		return "UTF-8", false
	}
	defer reader.Close()
	buf := make([]byte, 1024)
	n, err := util.ReadAtMost(reader, buf)
	if err != nil {
		// return default
		return "UTF-8", false
	}
	buf = buf[:n]

	if setting.LFS.StartServer {
		pointer, _ := lfs.ReadPointerFromBuffer(buf)
		if pointer.IsValid() {
			meta, err := git_model.GetLFSMetaObjectByOid(db.DefaultContext, repo.ID, pointer.Oid)
			if err != nil && err != git_model.ErrLFSObjectNotExist {
				// return default
				return "UTF-8", false
			}
			if meta != nil {
				dataRc, err := lfs.ReadMetaObject(pointer)
				if err != nil {
					// return default
					return "UTF-8", false
				}
				defer dataRc.Close()
				buf = make([]byte, 1024)
				n, err = util.ReadAtMost(dataRc, buf)
				if err != nil {
					// return default
					return "UTF-8", false
				}
				buf = buf[:n]
			}
		}
	}

	encoding, err := charset.DetectEncoding(buf)
	if err != nil {
		// just default to utf-8 and no bom
		return "UTF-8", false
	}
	if encoding == "UTF-8" {
		return encoding, bytes.Equal(buf[0:3], charset.UTF8BOM)
	}
	charsetEncoding, _ := stdcharset.Lookup(encoding)
	if charsetEncoding == nil {
		return "UTF-8", false
	}

	result, n, err := transform.String(charsetEncoding.NewDecoder(), string(buf))
	if err != nil {
		// return default
		return "UTF-8", false
	}

	if n > 2 {
		return encoding, bytes.Equal([]byte(result)[0:3], charset.UTF8BOM)
	}

	return encoding, false
}

// CreateOrUpdateRepoFile adds or updates a file in the given repository
func CreateOrUpdateRepoFile(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, opts *UpdateRepoFileOptions) (*structs.FileResponse, error) {
	// If no branch name is set, assume default branch
	if opts.OldBranch == "" {
		opts.OldBranch = repo.DefaultBranch
	}
	if opts.NewBranch == "" {
		opts.NewBranch = opts.OldBranch
	}

	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// oldBranch must exist for this operation
	if _, err := gitRepo.GetBranch(opts.OldBranch); err != nil && !repo.IsEmpty {
		return nil, err
	}

	// A NewBranch can be specified for the file to be created/updated in a new branch.
	// Check to make sure the branch does not already exist, otherwise we can't proceed.
	// If we aren't branching to a new branch, make sure user can commit to the given branch
	if opts.NewBranch != opts.OldBranch {
		existingBranch, err := gitRepo.GetBranch(opts.NewBranch)
		if existingBranch != nil {
			return nil, models.ErrBranchAlreadyExists{
				BranchName: opts.NewBranch,
			}
		}
		if err != nil && !git.IsErrBranchNotExist(err) {
			return nil, err
		}
	} else if err := VerifyBranchProtection(ctx, repo, doer, opts.OldBranch, opts.TreePath); err != nil {
		return nil, err
	}

	// If FromTreePath is not set, set it to the opts.TreePath
	if opts.TreePath != "" && opts.FromTreePath == "" {
		opts.FromTreePath = opts.TreePath
	}

	// Check that the path given in opts.treePath is valid (not a git path)
	treePath := CleanUploadFileName(opts.TreePath)
	if treePath == "" {
		return nil, models.ErrFilenameInvalid{
			Path: opts.TreePath,
		}
	}
	// If there is a fromTreePath (we are copying it), also clean it up
	fromTreePath := CleanUploadFileName(opts.FromTreePath)
	if fromTreePath == "" && opts.FromTreePath != "" {
		return nil, models.ErrFilenameInvalid{
			Path: opts.FromTreePath,
		}
	}

	message := strings.TrimSpace(opts.Message)

	author, committer := GetAuthorAndCommitterUsers(opts.Author, opts.Committer, doer)

	encoding := "UTF-8"
	bom := false

	if !repo.IsEmpty {
		// Get the commit of the original branch
		commit, err := gitRepo.GetBranchCommit(opts.OldBranch)
		if err != nil {
			return nil, err // Couldn't get a commit for the branch
		}

		// Assigned LastCommitID in opts if it hasn't been set
		if opts.LastCommitID == "" {
			opts.LastCommitID = commit.ID.String()
		} else {
			lastCommitID, err := gitRepo.ConvertToSHA1(opts.LastCommitID)
			if err != nil {
				return nil, fmt.Errorf("ConvertToSHA1: Invalid last commit ID: %w", err)
			}
			opts.LastCommitID = lastCommitID.String()

		}

		if !opts.IsNewFile {
			fromEntry, err := commit.GetTreeEntryByPath(fromTreePath)
			if err != nil {
				return nil, err
			}
			if opts.SHA != "" {
				// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
				if opts.SHA != fromEntry.ID.String() {
					return nil, models.ErrSHADoesNotMatch{
						Path:       treePath,
						GivenSHA:   opts.SHA,
						CurrentSHA: fromEntry.ID.String(),
					}
				}
			} else if opts.LastCommitID != "" {
				// If a lastCommitID was given and it doesn't match the commitID of the head of the branch throw
				// an error, but only if we aren't creating a new branch.
				if commit.ID.String() != opts.LastCommitID && opts.OldBranch == opts.NewBranch {
					if changed, err := commit.FileChangedSinceCommit(treePath, opts.LastCommitID); err != nil {
						return nil, err
					} else if changed {
						return nil, models.ErrCommitIDDoesNotMatch{
							GivenCommitID:   opts.LastCommitID,
							CurrentCommitID: opts.LastCommitID,
						}
					}
					// The file wasn't modified, so we are good to delete it
				}
			} else {
				// When updating a file, a lastCommitID or SHA needs to be given to make sure other commits
				// haven't been made. We throw an error if one wasn't provided.
				return nil, models.ErrSHAOrCommitIDNotProvided{}
			}
			encoding, bom = detectEncodingAndBOM(fromEntry, repo)
		}
	}

	content := opts.Content
	if bom {
		content = string(charset.UTF8BOM) + content
	}
	if encoding != "UTF-8" {
		charsetEncoding, _ := stdcharset.Lookup(encoding)
		if charsetEncoding != nil {
			result, _, err := transform.String(charsetEncoding.NewEncoder(), content)
			if err != nil {
				// Look if we can't encode back in to the original we should just stick with utf-8
				log.Error("Error re-encoding %s (%s) as %s - will stay as UTF-8: %v", opts.TreePath, opts.FromTreePath, encoding, err)
				result = content
			}
			content = result
		} else {
			log.Error("Unknown encoding: %s", encoding)
		}
	}
	// Reset the opts.Content to our adjusted content to ensure that LFS gets the correct content
	opts.Content = content

	requestMessages := make([]*gitalypb.UserCommitFilesRequest, 0)

	header := &gitalypb.UserCommitFilesRequest{
		UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Header{
			Header: &gitalypb.UserCommitFilesRequestHeader{
				Repository: gitRepo.GitalyRepo,
				User: &gitalypb.User{
					GlId:       strconv.Itoa(int(committer.ID)),
					Name:       []byte(committer.Name),
					Email:      []byte(committer.GetDefaultEmail()),
					GlUsername: committer.Name,
				},
				BranchName:        []byte(opts.NewBranch),
				CommitMessage:     []byte(message),
				CommitAuthorName:  []byte(author.Name),
				CommitAuthorEmail: []byte(author.GetDefaultEmail()),
				StartBranchName:   []byte(opts.OldBranch),
				Force:             false,
				StartSha:          opts.LastCommitID,
			},
		},
	}

	requestMessages = append(requestMessages, header)
	var actionType gitalypb.UserCommitFilesActionHeader_ActionType
	if treePath != fromTreePath {
		actionType = gitalypb.UserCommitFilesActionHeader_MOVE
		requestMessages = append(requestMessages, newActionRequest(actionType, treePath, fromTreePath))
	}
	if opts.IsNewFile {
		actionType = gitalypb.UserCommitFilesActionHeader_CREATE
	} else {
		actionType = gitalypb.UserCommitFilesActionHeader_UPDATE
	}

	requestMessages = append(requestMessages, newActionRequest(actionType, treePath, fromTreePath))
	requestMessages = append(requestMessages, &gitalypb.UserCommitFilesRequest{
		UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Action{
			Action: &gitalypb.UserCommitFilesAction{
				UserCommitFilesActionPayload: &gitalypb.UserCommitFilesAction_Content{
					Content: []byte(opts.Content),
				},
			},
		},
	},
	)

	ctxWithCancel, cancel := context.WithCancel(gitRepo.Ctx)
	defer cancel()
	userCommitFilesClient, err := gitRepo.OperationClient.UserCommitFiles(ctxWithCancel)
	if err != nil {
		return nil, err
	}

	for _, reqMes := range requestMessages {
		err = userCommitFilesClient.Send(reqMes)
		if err != nil {
			return nil, err
		}
	}

	recv, err := userCommitFilesClient.CloseAndRecv()
	if err != nil || recv.IndexError != "" || recv.PreReceiveError != "" {
		return nil, err
	}

	if repo.IsEmpty {
		_ = repo_model.UpdateRepositoryCols(ctx, &repo_model.Repository{ID: repo.ID, IsEmpty: false}, "is_empty")
	}

	return nil, nil
}

func newActionRequest(actionType gitalypb.UserCommitFilesActionHeader_ActionType, filepath, previousPath string) *gitalypb.UserCommitFilesRequest {
	return &gitalypb.UserCommitFilesRequest{
		UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Action{
			Action: &gitalypb.UserCommitFilesAction{
				UserCommitFilesActionPayload: &gitalypb.UserCommitFilesAction_Header{
					Header: &gitalypb.UserCommitFilesActionHeader{
						Action:       actionType,
						FilePath:     []byte(filepath),
						PreviousPath: []byte(previousPath),
					},
				},
			},
		},
	}
}

// VerifyBranchProtection verify the branch protection for modifying the given treePath on the given branch
func VerifyBranchProtection(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, branchName, treePath string) error {
	protectedBranch, err := git_model.GetMergeMatchProtectedBranchRule(ctx, repo.ID, branchName)
	if err != nil {
		log.Error("Error has occured while get merge match protected branch with repoID - %d, branchName - %s", repo.ID, branchName)
		return fmt.Errorf("Err: get merge protected branch: %w", err)
	}
	if protectedBranch != nil {
		protectedBranch.Repo = repo
		isUnprotectedFile := false
		glob := git_model.GetUnprotectedFilePatterns(*protectedBranch)
		if len(glob) != 0 {
			isUnprotectedFile = git_model.IsUnprotectedFile(*protectedBranch, glob, treePath)
		}
		if !git_model.CanUserPush(ctx, *protectedBranch, doer) && !isUnprotectedFile {
			return models.ErrUserCannotCommit{
				UserName: doer.LowerName,
			}
		}
		if protectedBranch.RequireSignedCommits {
			_, _, _, err := asymkey_service.SignCRUDAction(ctx, repo.OwnerName, repo.Name, repo.RepoPath(), doer, repo.RepoPath(), branchName)
			if err != nil {
				if !asymkey_service.IsErrWontSign(err) {
					return err
				}
				return models.ErrUserCannotCommit{
					UserName: doer.LowerName,
				}
			}
		}
		patterns := git_model.GetProtectedFilePatterns(*protectedBranch)
		for _, pat := range patterns {
			if pat.Match(strings.ToLower(treePath)) {
				return models.ErrFilePathProtected{
					Path: treePath,
				}
			}
		}
	}
	return nil
}
