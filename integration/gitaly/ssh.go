package gitaly

import (
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

type SSHServiceClient struct {
	gitalypb.SSHServiceClient
}
