//go:build !correct

package task_tracker_client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/gitnames"
)

var testToken = "token"

func testHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")

	if token != testToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp := `{"units":[{"code":"GITRU-1","isExists":true}]}`
	w.Write([]byte(resp))
}

func TestTaskTrackerClient_CheckCodes(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(testHandler))
	defer testServer.Close()

	ctx := context.Background()

	codes := []gitnames.UnitCode{{Code: "GITRU-1"}}

	client := TaskTrackerClient{
		httpClient: testServer.Client(),
		baseURL:    testServer.URL,
		token:      testToken,
	}

	resp := CheckCodesResponse{Units: []Unit{{Code: "GITRU-1", IsExists: true}}}

	got, err := client.CheckCodes(ctx, codes)
	require.NoError(t, err)
	require.Equal(t, resp, got)
}

func TestTaskTrackerClient_CheckCodes_negative(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(testHandler))
	defer testServer.Close()

	ctx := context.Background()

	codes := []gitnames.UnitCode{{Code: "GITRU-1"}}

	client := TaskTrackerClient{
		httpClient: testServer.Client(),
		baseURL:    testServer.URL,
		token:      "wrongtoken",
	}

	_, err := client.CheckCodes(ctx, codes)
	require.Error(t, err)
}
