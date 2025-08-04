package gitnamesparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUnitCodeNotFoundError(t *testing.T) {
	t.Run("UnitCodeNotFoundError check", func(t *testing.T) {
		gotErr := NewUnitCodeNotFoundError("raw name")
		targetErr := &UnitCodeNotFoundError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}

func TestNewEmptyCommitLinksError(t *testing.T) {
	t.Run("EmptyCommitLinksError check", func(t *testing.T) {
		gotErr := NewEmptyCommitLinksError()
		targetErr := &EmptyCommitLinksError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
