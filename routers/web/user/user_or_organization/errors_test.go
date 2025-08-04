package user_or_organization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInternalServerError(t *testing.T) {
	t.Run("Internal server error check", func(t *testing.T) {
		gotErr := NewInternalServerError(fmt.Errorf("some error"))
		targetErr := &InternalServerError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
