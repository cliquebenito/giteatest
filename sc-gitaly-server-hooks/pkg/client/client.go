package client

import (
	"context"

	"sc-gitaly-server-hooks/pkg/models"
)

type HookClient interface {
	PreReceive(ctx context.Context, requestOpts *models.HookRequestOptions) ResponseExtra
	PostReceive(ctx context.Context, requestOpts *models.HookRequestOptions) (*models.HookPostReceiveResult, ResponseExtra)
	ProcReceive(ctx context.Context, requestOpts *models.HookRequestOptions) (*models.HookProcReceiveResult, ResponseExtra)
}
