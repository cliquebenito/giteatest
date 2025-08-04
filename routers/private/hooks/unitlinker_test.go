//go:build !correct

package hooks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getProjectAndRepoNames(t *testing.T) {
	type testCase struct {
		name     string
		uri      string
		wantRepo string

		wantProject string
	}

	tests := []testCase{
		{uri: "/api/internal/hook/post-receive/sdf/repo1", wantRepo: "repo1", wantProject: "sdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProject, gotRepo, err := getProjectAndRepoNames(tt.uri)
			require.NoError(t, err)

		})
	}
}
