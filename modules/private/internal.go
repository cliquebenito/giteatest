// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"code.gitea.io/gitea/models/git_hooks"

	"code.gitea.io/gitea/modules/httplib"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/proxyprotocol"
	"code.gitea.io/gitea/modules/setting"
)

const (
	envSSHConnection = "SSH_CONNECTION"
	envSSHClient     = "SSH_CLIENT"
	defaultClientIP  = "127.0.0.1"
)

// Response is used for internal request response (for user message and error message)
type Response struct {
	Err     string `json:"err,omitempty"`      // server-side error log message, it won't be exposed to end users
	UserMsg string `json:"user_msg,omitempty"` // meaningful error message for end users, it will be shown in git client's output.
}

// ResponseWithHook is used for internal request response (for git hook and error message)
type ResponseWithHook struct {
	Err  string              `json:"err,omitempty"`  // server-side error log message, it won't be exposed to end users
	Hook git_hooks.ScGitHook `json:"hook,omitempty"` // git hook
}

func getClientIP() string {
	sshConnEnv := strings.TrimSpace(os.Getenv(envSSHConnection))
	if len(sshConnEnv) == 0 {
		sshClientEnv := strings.TrimSpace(os.Getenv(envSSHClient))
		if len(sshClientEnv) > 0 {
			fields := strings.Fields(sshClientEnv)
			if len(fields) >= 1 && fields[0] != "" {
				return fields[0]
			}
		}
		return defaultClientIP
	}
	fields := strings.Fields(sshConnEnv)
	if len(fields) >= 1 && fields[0] != "" {
		return fields[0]
	}
	return defaultClientIP
}

func newInternalRequest(ctx context.Context, url, method string, body ...any) *httplib.Request {
	if setting.InternalToken == "" {
		log.Fatal(`The INTERNAL_TOKEN setting is missing from the configuration file: %q.
Ensure you are running in the correct environment or set the correct configuration file with -c.`, setting.CustomConf)
	}
	// если включена интеграция с sec man берем internal token из env, в ином случае из app.ini
	internalToken := os.Getenv("INTERNAL_TOKEN")
	if len(internalToken) == 0 {
		internalToken = setting.InternalToken
	}
	req := httplib.NewRequest(url, method).
		SetContext(ctx).
		Header("X-Real-IP", getClientIP()).
		Header("Authorization", fmt.Sprintf("Bearer %s", internalToken)).
		SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
			ServerName:         setting.Domain,
		})

	if setting.Protocol == setting.HTTPUnix {
		req.SetTransport(&http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				conn, err := d.DialContext(ctx, "unix", setting.HTTPAddr)
				if err != nil {
					return conn, err
				}
				if setting.LocalUseProxyProtocol {
					if err = proxyprotocol.WriteLocalHeader(conn); err != nil {
						_ = conn.Close()
						return nil, err
					}
				}
				return conn, err
			},
		})
	} else if setting.LocalUseProxyProtocol {
		req.SetTransport(&http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				conn, err := d.DialContext(ctx, network, address)
				if err != nil {
					return conn, err
				}
				if err = proxyprotocol.WriteLocalHeader(conn); err != nil {
					_ = conn.Close()
					return nil, err
				}
				return conn, err
			},
		})
	}

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
