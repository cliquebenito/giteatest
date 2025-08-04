package gitnames

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommit_GetUniqCodes(t *testing.T) {
	tests := []struct {
		name string
		base Base
		want []UnitCode
	}{
		{base: Base{LinkedUnits: LinkedUnits{{Code: "GITRU-1"}, {Code: "GITRU-2"}, {Code: "GITRU-2"}}}, want: []UnitCode{{Code: "GITRU-1"}, {Code: "GITRU-2"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := CommitLinks{Base: tt.base}
			got, err := b.GetUniqCodes()
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestCommit_GetUniqCodes_negative(t *testing.T) {
	tests := []struct {
		name string
		base Base
		want error
	}{
		{base: Base{}, want: NewEmptyUnitCodesListError()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := CommitLinks{Base: tt.base}
			_, err := b.GetUniqCodes()
			require.ErrorAs(t, err, &tt.want)
		})
	}
}
