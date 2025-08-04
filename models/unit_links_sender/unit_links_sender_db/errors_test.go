package unit_links_sender_db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPullRequestNotFoundError(t *testing.T) {
	t.Run("TaskAlreadyLockedError check", func(t *testing.T) {
		gotErr := NewTaskAlreadyLockedError(0)
		targetErr := &TaskAlreadyLockedError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
