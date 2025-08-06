package unit_links_db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPullRequestNotFoundError(t *testing.T) {
	t.Run("PullRequestNotFoundError check", func(t *testing.T) {
		gotErr := NewPullRequestNotFoundError(0)
		targetErr := &PullRequestNotFoundError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
