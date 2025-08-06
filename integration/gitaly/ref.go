package gitaly

import "gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

// RefClient encapsulates RefService calls
type RefClient struct {
	gitalypb.RefServiceClient
}
