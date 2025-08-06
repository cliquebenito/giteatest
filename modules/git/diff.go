// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// RawDiffType type of a raw diff.
type RawDiffType string

// RawDiffType possible values.
const (
	RawDiffNormal RawDiffType = "diff"
	RawDiffPatch  RawDiffType = "patch"
)

// GetRawDiff dumps diff results of repository in given commit ID to io.Writer.
func GetRawDiff(repo *Repository, commitID string, diffType RawDiffType, writer io.Writer) error {
	return GetRepoRawDiffForFile(repo, "4b825dc642cb6eb9a060e54bf8d69288fbee4904", commitID, diffType, "", writer) // "4b825dc642cb6eb9a060e54bf8d69288fbee4904" empty
}

//todo use by flag

// GetReverseRawDiff dumps the reverse diff results of repository in given commit ID to io.Writer.
func GetReverseRawDiff(ctx context.Context, repoPath, commitID string, writer io.Writer) error {
	stderr := new(bytes.Buffer)
	cmd := NewCommand(ctx, "show", "--pretty=format:revert %H%n", "-R").AddDynamicArguments(commitID)
	if err := cmd.Run(&RunOpts{
		Dir:    repoPath,
		Stdout: writer,
		Stderr: stderr,
	}); err != nil {
		return fmt.Errorf("Run: %w - %s", err, stderr)
	}
	return nil
}

// GetRepoRawDiffForFile dumps diff results of file in given commit ID to io.Writer according given repository
func GetRepoRawDiffForFile(repo *Repository, startCommit, endCommit string, diffType RawDiffType, file string, writer io.Writer) error {
	byteFiles := make([][]byte, 0)
	byteFiles = append(byteFiles, []byte(file))
	const maxBytes = int32(^uint32(0) >> 1)
	switch diffType {
	case RawDiffNormal:
		commitDiff, err := repo.DiffClient.CommitDiff(repo.Ctx, &gitalypb.CommitDiffRequest{
			Repository:    repo.GitalyRepo,
			LeftCommitId:  startCommit,
			RightCommitId: endCommit,
			Paths:         byteFiles,
			CollapseDiffs: false,
			EnforceLimits: true,
			MaxFiles:      maxBytes,
			MaxLines:      maxBytes,
			MaxBytes:      maxBytes,
			MaxPatchBytes: maxBytes,
		})
		if err != nil {
			return err
		}
		canRead := true
		for canRead {
			recv, err := commitDiff.Recv()
			if err != nil && err != io.EOF {
				return err
			}
			if recv == nil {
				canRead = false
				continue
			}
			rawPatchData, err := ReadRawPatchData(commitDiff, string(recv.GetRawPatchData()), recv.GetEndOfPatch())
			if err != nil {
				log.Error("Error has occurred while reading raw patch data: %v", err)
				return fmt.Errorf("failed to read raw patch data: %w", err)
			}
			_, err = writer.Write([]byte(rawPatchData))
			if err != nil {
				return err
			}
		}

		//diff, err := repo.DiffClient.RawDiff(repo.Ctx, &gitalypb.RawDiffRequest{
		//	Repository:    repo.GitalyRepo,
		//	LeftCommitId:  startCommit,
		//	RightCommitId: endCommit,
		//}, grpc.MaxCallRecvMsgSize(int(^uint32(0)>>1)), grpc.MaxCallSendMsgSize(int(^uint32(0)>>1)), grpc.MaxRetryRPCBufferSize(int(^uint32(0)>>1)))
		//if err != nil {
		//	return err
		//}
		//// todo не все вычитывается
		//msg := &gitalypb.RawDiffResponse{}
		////recv, err := diff.Recv()
		//err = diff.RecvMsg(msg)
		//if err != nil {
		//	return err
		//}
		//_, err = writer.Write(msg.GetData())
		//if err != nil {
		//	return err
		//}
	case RawDiffPatch:
		diff, err := repo.DiffClient.RawPatch(repo.Ctx, &gitalypb.RawPatchRequest{
			Repository:    repo.GitalyRepo,
			LeftCommitId:  startCommit,
			RightCommitId: endCommit,
		})
		if err != nil {
			return err
		}
		recv, err := diff.Recv()
		if err != nil {
			return err
		}
		_, err = writer.Write(recv.GetData())
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid diffType: %s", diffType)
	}
	return nil
}

// ParseDiffHunkString parse the diffhunk content and return
func ParseDiffHunkString(diffhunk string) (leftLine, leftHunk, rightLine, righHunk int) {
	ss := strings.Split(diffhunk, "@@")
	ranges := strings.Split(ss[1][1:], " ")
	leftRange := strings.Split(ranges[0], ",")
	leftLine, _ = strconv.Atoi(leftRange[0][1:])
	if len(leftRange) > 1 {
		leftHunk, _ = strconv.Atoi(leftRange[1])
	}
	if len(ranges) > 1 {
		rightRange := strings.Split(ranges[1], ",")
		rightLine, _ = strconv.Atoi(rightRange[0])
		if len(rightRange) > 1 {
			righHunk, _ = strconv.Atoi(rightRange[1])
		}
	} else {
		log.Debug("Parse line number failed: %v", diffhunk)
		rightLine = leftLine
		righHunk = leftHunk
	}
	return leftLine, leftHunk, rightLine, righHunk
}

// Example: @@ -1,8 +1,9 @@ => [..., 1, 8, 1, 9]
var hunkRegex = regexp.MustCompile(`^@@ -(?P<beginOld>[0-9]+)(,(?P<endOld>[0-9]+))? \+(?P<beginNew>[0-9]+)(,(?P<endNew>[0-9]+))? @@`)

const cmdDiffHead = "diff --git "

func isHeader(lof string, inHunk bool) bool {
	return strings.HasPrefix(lof, cmdDiffHead) || (!inHunk && (strings.HasPrefix(lof, "---") || strings.HasPrefix(lof, "+++")))
}

// CutDiffAroundLine cuts a diff of a file in way that only the given line + numberOfLine above it will be shown
// it also recalculates hunks and adds the appropriate headers to the new diff.
// Warning: Only one-file diffs are allowed.
func CutDiffAroundLine(originalDiff io.Reader, line int64, old bool, numbersOfLine int) (string, error) {
	if line == 0 || numbersOfLine == 0 {
		// no line or num of lines => no diff
		return "", nil
	}

	scanner := bufio.NewScanner(originalDiff)
	hunk := make([]string, 0)

	// begin is the start of the hunk containing searched line
	// end is the end of the hunk ...
	// currentLine is the line number on the side of the searched line (differentiated by old)
	// otherLine is the line number on the opposite side of the searched line (differentiated by old)
	var begin, end, currentLine, otherLine int64
	var headerLines int

	inHunk := false

	for scanner.Scan() {
		lof := scanner.Text()
		// Add header to enable parsing

		if isHeader(lof, inHunk) {
			if strings.HasPrefix(lof, cmdDiffHead) {
				inHunk = false
			}
			hunk = append(hunk, lof)
			headerLines++
		}
		if currentLine > line {
			break
		}
		// Detect "hunk" with contains commented lof
		if strings.HasPrefix(lof, "@@") {
			inHunk = true
			// Already got our hunk. End of hunk detected!
			if len(hunk) > headerLines {
				break
			}
			// A map with named groups of our regex to recognize them later more easily
			submatches := hunkRegex.FindStringSubmatch(lof)
			groups := make(map[string]string)
			for i, name := range hunkRegex.SubexpNames() {
				if i != 0 && name != "" {
					groups[name] = submatches[i]
				}
			}
			if old {
				begin, _ = strconv.ParseInt(groups["beginOld"], 10, 64)
				end, _ = strconv.ParseInt(groups["endOld"], 10, 64)
				// init otherLine with begin of opposite side
				otherLine, _ = strconv.ParseInt(groups["beginNew"], 10, 64)
			} else {
				begin, _ = strconv.ParseInt(groups["beginNew"], 10, 64)
				if groups["endNew"] != "" {
					end, _ = strconv.ParseInt(groups["endNew"], 10, 64)
				} else {
					end = 0
				}
				// init otherLine with begin of opposite side
				otherLine, _ = strconv.ParseInt(groups["beginOld"], 10, 64)
			}
			end += begin // end is for real only the number of lines in hunk
			// lof is between begin and end
			if begin <= line && end >= line {
				hunk = append(hunk, lof)
				currentLine = begin
				continue
			}
		} else if len(hunk) > headerLines {
			hunk = append(hunk, lof)
			// Count lines in context
			switch lof[0] {
			case '+':
				if !old {
					currentLine++
				} else {
					otherLine++
				}
			case '-':
				if old {
					currentLine++
				} else {
					otherLine++
				}
			case '\\':
				// FIXME: handle `\ No newline at end of file`
			default:
				currentLine++
				otherLine++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// No hunk found
	if currentLine == 0 {
		return "", nil
	}
	// headerLines + hunkLine (1) = totalNonCodeLines
	if len(hunk)-headerLines-1 <= numbersOfLine {
		// No need to cut the hunk => return existing hunk
		return strings.Join(hunk, "\n"), nil
	}
	var oldBegin, oldNumOfLines, newBegin, newNumOfLines int64
	if old {
		oldBegin = currentLine
		newBegin = otherLine
	} else {
		oldBegin = otherLine
		newBegin = currentLine
	}
	// headers + hunk header
	newHunk := make([]string, headerLines)
	// transfer existing headers
	copy(newHunk, hunk[:headerLines])
	// transfer last n lines
	newHunk = append(newHunk, hunk[len(hunk)-numbersOfLine-1:]...)
	// calculate newBegin, ... by counting lines
	for i := len(hunk) - 1; i >= len(hunk)-numbersOfLine; i-- {
		switch hunk[i][0] {
		case '+':
			newBegin--
			newNumOfLines++
		case '-':
			oldBegin--
			oldNumOfLines++
		default:
			oldBegin--
			newBegin--
			newNumOfLines++
			oldNumOfLines++
		}
	}
	// construct the new hunk header
	newHunk[headerLines] = fmt.Sprintf("@@ -%d,%d +%d,%d @@",
		oldBegin, oldNumOfLines, newBegin, newNumOfLines)
	diff := strings.Join(newHunk, "\n")
	diff = fmt.Sprintf("diff --git \n--- \n+++ \n%s", diff)
	return diff, nil
}

// GetAffectedFiles returns the affected files between two commits
func GetAffectedFiles(repo *Repository, oldCommitID, newCommitID string, env []string) ([]string, error) {
	affectedFiles := make([]string, 0, 32)

	req := make([]*gitalypb.FindChangedPathsRequest_Request, 0, 2)
	req = append(req,
		&gitalypb.FindChangedPathsRequest_Request{
			Type: &gitalypb.FindChangedPathsRequest_Request_CommitRequest_{
				CommitRequest: &gitalypb.FindChangedPathsRequest_Request_CommitRequest{
					CommitRevision: oldCommitID,
				},
			},
		},
	)
	req = append(req,
		&gitalypb.FindChangedPathsRequest_Request{
			Type: &gitalypb.FindChangedPathsRequest_Request_CommitRequest_{
				CommitRequest: &gitalypb.FindChangedPathsRequest_Request_CommitRequest{
					CommitRevision: newCommitID,
				},
			},
		},
	)

	paths, err := repo.DiffClient.FindChangedPaths(repo.Ctx, &gitalypb.FindChangedPathsRequest{
		Repository: repo.GitalyRepo,
		Commits:    nil,
		Requests:   req,
	})
	if err != nil {
		return nil, err
	}

	recv, err := paths.Recv()
	if err != nil {
		return nil, err
	}

	for _, path := range recv.GetPaths() {
		affectedFiles = append(affectedFiles, string(path.Path))
	}

	return affectedFiles, err
}

func ReadRawPatchData(commitDiffStream gitalypb.DiffService_CommitDiffClient, currentRawPatchData string, endOfPatch bool) (string, error) {
	rawPatchData := currentRawPatchData
	if endOfPatch {
		return rawPatchData, nil
	}

	recv, err := commitDiffStream.Recv()
	if err != nil && err != io.EOF {
		log.Error("Error has occurred while reading commit diff stream: %v", err)
		return rawPatchData, fmt.Errorf("received error while reading commit diff stream: %w", err)
	}
	if recv == nil {
		return rawPatchData, nil
	}

	rawPatchData += string(recv.GetRawPatchData())
	return ReadRawPatchData(commitDiffStream, rawPatchData, recv.EndOfPatch)
}
