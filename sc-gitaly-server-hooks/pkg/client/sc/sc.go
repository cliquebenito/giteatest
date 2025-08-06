package sc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"sc-gitaly-server-hooks/pkg/client"
	"sc-gitaly-server-hooks/pkg/models"
)

type SCHookClient struct {
	config *models.SourceControlCofig
}

func NewScClient(config *models.SourceControlCofig) SCHookClient {
	return SCHookClient{
		config: config,
	}
}

func (s SCHookClient) newInternalRequest(ctx context.Context, url, method string, body ...any) *client.Request {
	req := client.NewRequest(url, method).
		SetContext(ctx).
		Header("X-Real-IP", getClientIP()).
		Header("Authorization", fmt.Sprintf("Bearer %s", s.config.GetToken())).
		SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "localhost",
		})

	if len(body) == 1 {
		req.Header("Content-Type", "application/json")
		jsonBytes, _ := json.Marshal(body[0])
		req.Body(jsonBytes)
	} else if len(body) > 1 {
		log.Fatal("Too many arguments for newInternalRequest")
	}

	req.SetTimeout(10*time.Second, 60*time.Second)
	return req
}

// PreReceive check whether the provided commits are allowed
func (s SCHookClient) PreReceive(ctx context.Context, requestOpts *models.HookRequestOptions) client.ResponseExtra {
	reqURL := fmt.Sprintf("%sapi/internal/hook/pre-receive/%s/%s", s.config.GetAddress(), url.PathEscape(requestOpts.OwnerName), url.PathEscape(requestOpts.RepoName))
	req := s.newInternalRequest(ctx, reqURL, "POST", requestOpts.Opts)
	req.SetReadWriteTimeout(time.Duration(60+len(requestOpts.Opts.OldCommitIDs)) * time.Second)
	_, extra := client.RequestJSONResp(req, &client.ResponseText{})
	return extra
}

// PostReceive updates services and users
func (s SCHookClient) PostReceive(ctx context.Context, requestOpts *models.HookRequestOptions) (*models.HookPostReceiveResult, client.ResponseExtra) {
	reqURL := fmt.Sprintf("%sapi/internal/hook/post-receive/%s/%s", s.config.GetAddress(), url.PathEscape(requestOpts.OwnerName), url.PathEscape(requestOpts.RepoName))
	req := s.newInternalRequest(ctx, reqURL, "POST", requestOpts.Opts)
	req.SetReadWriteTimeout(time.Duration(60+len(requestOpts.Opts.OldCommitIDs)) * time.Second)
	return client.RequestJSONResp(req, &models.HookPostReceiveResult{})
}

// ProcReceive proc-receive hook
func (s SCHookClient) ProcReceive(ctx context.Context, requestOpts *models.HookRequestOptions) (*models.HookProcReceiveResult, client.ResponseExtra) {
	reqURL := fmt.Sprintf("%sapi/internal/hook/proc-receive/%s/%s", s.config.GetAddress(), url.PathEscape(requestOpts.OwnerName), url.PathEscape(requestOpts.RepoName))

	req := s.newInternalRequest(ctx, reqURL, "POST", requestOpts.Opts)
	req.SetReadWriteTimeout(time.Duration(60+len(requestOpts.Opts.OldCommitIDs)) * time.Second)
	return client.RequestJSONResp(req, &models.HookProcReceiveResult{})
}

func getClientIP() string {
	sshConnEnv := strings.TrimSpace(os.Getenv("SSH_CONNECTION"))
	if len(sshConnEnv) == 0 {
		return "127.0.0.1"
	}
	return strings.Fields(sshConnEnv)[0]
}
