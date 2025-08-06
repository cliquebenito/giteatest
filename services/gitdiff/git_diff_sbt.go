package gitdiff

import (
	"bufio"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/sbt/response"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// DiffFileStatus статусы файлов
// Added (A), Copied (C), Deleted (D), Modified (M), Renamed (R), все остальные - Unknown
type DiffFileStatus uint8

const (
	DiffFileStatusAdd DiffFileStatus = iota + 1
	DiffFileStatusCopy
	DiffFileStatusDel
	DiffFileStatusModified
	DiffFileStatusRename
	DiffFileStatusUnknown
)

// GetDiffFilesWithStat по аналогии с методом GetDiff
// Запрос git diff --name-status, который возвращает список файлов со статусом изменений произошедших в этом файле
func GetDiffFilesWithStat(gitRepo *git.Repository, opts *DiffOptions) ([]*response.CommitFiles, error) {
	repoPath := gitRepo.Path

	commit, err := gitRepo.GetCommit(opts.AfterCommitID)
	if err != nil {
		return nil, err
	}

	cmdDiff := git.NewCommand(gitRepo.Ctx)
	if (len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == git.EmptySHA) && commit.ParentCount() == 0 {
		cmdDiff.AddArguments("diff", "--name-status").
			AddArguments(git.EmptyTreeSHA). // append empty tree ref
			AddDynamicArguments(opts.AfterCommitID)
	} else {
		actualBeforeCommitID := opts.BeforeCommitID
		if len(actualBeforeCommitID) == 0 {
			parentCommit, _ := commit.Parent(0)
			actualBeforeCommitID = parentCommit.ID.String()
		}

		cmdDiff.AddArguments("diff", "--name-status").
			AddDynamicArguments(actualBeforeCommitID, opts.AfterCommitID)
		opts.BeforeCommitID = actualBeforeCommitID
	}

	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	go func() {
		cmdDiff.SetDescription(fmt.Sprintf("GetDiffRangeFiles [repo_path: %s]", repoPath))
		if err := cmdDiff.Run(&git.RunOpts{
			Timeout: time.Duration(setting.Git.Timeout.Default) * time.Second,
			Dir:     repoPath,
			Stderr:  os.Stderr,
			Stdout:  writer,
		}); err != nil {
			log.Error("error during RunWithContext: %w", err)
		}

		_ = writer.Close()
	}()

	diff, err := parseDiffFileWithStatus(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to ParsePatch: %w", err)
	}

	return diff, nil
}

// ParseDiffFileWithStatus - парсер ответа на запрос git diff --name-status,
// который возвращает список файлов со статусом изменений
// Added (A), Copied (C), Deleted (D), Modified (M), Renamed (R), все остальные - Unknown
func parseDiffFileWithStatus(reader io.Reader) ([]*response.CommitFiles, error) {
	diff := make([]*response.CommitFiles, 0)

	readerSize := setting.Git.MaxGitDiffLineCharacters

	input := bufio.NewReaderSize(reader, readerSize)
	line, err := input.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return diff, nil
		}
		return diff, err
	}

	for {
		var currDiff response.CommitFiles
		switch {
		case strings.HasPrefix(line, "A"):
			currDiff.Status = int8(DiffFileStatusAdd)
		case strings.HasPrefix(line, "C"):
			currDiff.Status = int8(DiffFileStatusCopy)
		case strings.HasPrefix(line, "D"):
			currDiff.Status = int8(DiffFileStatusDel)
		case strings.HasPrefix(line, "M"):
			currDiff.Status = int8(DiffFileStatusModified)
		case strings.HasPrefix(line, "R"):
			currDiff.Status = int8(DiffFileStatusRename)
		default:
			currDiff.Status = int8(DiffFileStatusUnknown)
		}

		currDiff.File = strings.TrimSpace(line[1:len(line)])
		line, err = input.ReadString('\n')

		diff = append(diff, &currDiff)

		if err != nil {
			if err != io.EOF {
				return diff, err
			}
			break
		}
	}

	return diff, nil
}

// GetDiffStat по аналогии с методом GetDiff только с параметром --shortstat, который выводит статистику изменений в коммите
// число добавленных, удаленных строк, общее число измененных строк и количество измененных файлов в коммите
func GetDiffStat(gitRepo *git.Repository, opts *DiffOptions) (*response.CommitStats, error) {
	diff := response.CommitStats{}
	commit, err := gitRepo.GetCommit(opts.AfterCommitID)
	if err != nil {
		return nil, err
	}

	separator := "..."

	if (len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == git.EmptySHA) && commit.ParentCount() != 0 {
		parentCommit, _ := commit.Parent(0)
		opts.BeforeCommitID = parentCommit.ID.String()
	}

	diffPaths := []string{opts.BeforeCommitID + separator + opts.AfterCommitID}
	if len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == git.EmptySHA {
		diffPaths = []string{git.EmptyTreeSHA, opts.AfterCommitID}
	}

	diff.FilesChanged, diff.Additions, diff.Deletions, err = git.GetDiffShortStat(gitRepo.Ctx, gitRepo.Path, nil, diffPaths...)
	if err != nil && strings.Contains(err.Error(), "no merge base") {
		// git >= 2.28 now returns an error if base and head have become unrelated.
		// previously it would return the results of git diff --shortstat base head so let's try that...
		diffPaths = []string{opts.BeforeCommitID, opts.AfterCommitID}
		diff.FilesChanged, diff.Additions, diff.Deletions, err = git.GetDiffShortStat(gitRepo.Ctx, gitRepo.Path, nil, diffPaths...)
	}

	diff.Total = diff.Additions + diff.Deletions
	return &diff, nil
}

// GetDiffFile builds a Diff between two commits of a repository
// Passing the empty string as beforeCommitID returns a diff from the parent commit.
// по аналогии с методом GetDiff, только без check path и short stat
func GetDiffFile(gitRepo *git.Repository, opts *DiffOptions, files ...string) (*DiffFile, error) {
	repoPath := gitRepo.Path

	commit, err := gitRepo.GetCommit(opts.AfterCommitID)
	if err != nil {
		return nil, err
	}

	cmdDiff := git.NewCommand(gitRepo.Ctx)
	if (len(opts.BeforeCommitID) == 0 || opts.BeforeCommitID == git.EmptySHA) && commit.ParentCount() == 0 {
		cmdDiff.AddArguments("diff", "--src-prefix=\\a/", "--dst-prefix=\\b/", "-M").
			AddArguments(git.EmptyTreeSHA). // append empty tree ref
			AddDynamicArguments(opts.AfterCommitID)
	} else {
		actualBeforeCommitID := opts.BeforeCommitID
		if len(actualBeforeCommitID) == 0 {
			parentCommit, _ := commit.Parent(0)
			actualBeforeCommitID = parentCommit.ID.String()
		}

		cmdDiff.AddArguments("diff", "--src-prefix=\\a/", "--dst-prefix=\\b/", "-M").
			AddDynamicArguments(actualBeforeCommitID, opts.AfterCommitID)
		opts.BeforeCommitID = actualBeforeCommitID
	}

	cmdDiff.AddDashesAndList(files...)

	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	go func() {
		cmdDiff.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
		if err := cmdDiff.Run(&git.RunOpts{
			Timeout: time.Duration(setting.Git.Timeout.Default) * time.Second,
			Dir:     repoPath,
			Stderr:  os.Stderr,
			Stdout:  writer,
		}); err != nil {
			log.Error("error during RunWithContext: %w", err)
		}

		_ = writer.Close()
	}()

	diff, err := ParsePatch(opts.MaxLines, opts.MaxLineCharacters, opts.MaxFiles, reader, "")
	if err != nil {
		return nil, fmt.Errorf("unable to ParsePatch: %w", err)
	}

	if len(diff.Files) != 0 {
		return diff.Files[0], nil
	}

	return nil, nil
}
