package gitaly

import "gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

type OperationClient struct {
	gitalypb.OperationServiceClient
}
