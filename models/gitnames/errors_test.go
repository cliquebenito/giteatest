package gitnames

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEmptyUnitCodesListError(t *testing.T) {
	t.Run("UnitCodeNotFoundError check", func(t *testing.T) {
		gotErr := NewEmptyUnitCodesListError()
		targetErr := &EmptyUnitCodesListError{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
