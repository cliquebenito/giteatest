package console

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"sc-gitaly-server-hooks/pkg/models"
)

type Reader struct{}

func NewConsoleReader() Reader {
	return Reader{}
}

func (r Reader) Read(ctx context.Context, source io.Reader) ([]models.CommitDescriptor, error) {
	commitDescriptors := make([]models.CommitDescriptor, 0)
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fields := bytes.Fields(scanner.Bytes())
			if len(fields) != 3 {
				continue
			}

			commitDescriptor := models.CommitDescriptor{
				ParentCommitSha: string(fields[0]),
				ChildCommitSha:  string(fields[1]),
				RefName:         string(fields[2]),
			}

			commitDescriptors = append(commitDescriptors, commitDescriptor)
		}
	}

	return commitDescriptors, nil
}

func (r Reader) ReadBranchesAndTags(ctx context.Context, source io.Reader) ([]models.CommitDescriptor, error) {
	commitDescriptors := make([]models.CommitDescriptor, 0)
	scanner := bufio.NewScanner(source)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fields := bytes.Fields(scanner.Bytes())
			if len(fields) != 3 {
				continue
			}

			commitDescriptor := models.CommitDescriptor{
				ParentCommitSha: string(fields[0]),
				ChildCommitSha:  string(fields[1]),
				RefName:         string(fields[2]),
			}

			if !strings.HasPrefix(commitDescriptor.RefName, models.BranchPrefix) && !strings.HasPrefix(commitDescriptor.RefName, models.TagPrefix) {
				continue
			}

			commitDescriptors = append(commitDescriptors, commitDescriptor)
		}
	}

	return commitDescriptors, nil
}

func (r Reader) ReadPktLine(ctx context.Context, source io.Reader, requestType models.PktLineType) (*models.GitPktLine, error) {
	in := bufio.NewReader(source)

	lengthBytes := make([]byte, 4)

	var err error
	for i := 0; i < 4; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			lengthBytes[i], err = in.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("Pkt-Line: read stdin failed : %v", err)
			}
		}
	}

	gitPktLine := new(models.GitPktLine)

	gitPktLine.Length, err = strconv.ParseUint(string(lengthBytes), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("protocol: format parse error.\nPkt-Line format is wrong :%v", err)
	}

	if gitPktLine.Length == 0 {
		if requestType == models.PktLineTypeData {
			return nil, fmt.Errorf("protocol: format data error.\nPkt-Line format is wrong")
		}
		gitPktLine.Type = models.PktLineTypeFlush
		return gitPktLine, nil
	}

	if gitPktLine.Length <= 4 || gitPktLine.Length > 65520 || requestType == models.PktLineTypeFlush {
		return nil, fmt.Errorf("protocol: format length error.\nPkt-Line format is wrong")
	}

	gitPktLine.Data = make([]byte, gitPktLine.Length-4)
	for i := range gitPktLine.Data {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gitPktLine.Data[i], err = in.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("protocol: data error.\nPkt-Line: read stdin failed : %v", err)
			}
		}
	}

	gitPktLine.Type = models.PktLineTypeData

	return gitPktLine, nil
}
