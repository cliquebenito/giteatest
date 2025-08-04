package branch

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/modules/gitnamesparser"
)

func Test_branchParser_Parse(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		want       gitnames.BranchLinks
	}{
		{
			branchName: "feature/GITRU-1_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/GITRU-1_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-1"}},
				},
			},
		},
		{
			branchName: "feature/GIT_RU-1_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/GIT_RU-1_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}},
				},
			},
		},
		{
			branchName: "feature/GIT_RU_12-1_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/GIT_RU_12-1_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU_12-1"}},
				},
			},
		},
		{
			branchName: "GIT_RU-1_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GIT_RU-1_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}},
				},
			},
		},
		{
			branchName: "GIT_RU-1-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GIT_RU-1-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}},
				},
			},
		},
		{
			branchName: "GIT_RU-1-GIT_RU-2-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GIT_RU-1-GIT_RU-2-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}, {Code: "GIT_RU-2"}},
				},
			},
		},
		{
			branchName: "GIT_RU-1-GIT_RU-2_GITRU-3-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GIT_RU-1-GIT_RU-2_GITRU-3-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}, {Code: "GIT_RU-2"}, {Code: "GITRU-3"}},
				},
			},
		},
		{
			branchName: "feature/GIT_RU-1-GIT_RU-2_GITRU-3-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/GIT_RU-1-GIT_RU-2_GITRU-3-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-1"}, {Code: "GIT_RU-2"}, {Code: "GITRU-3"}},
				},
			},
		},
		{
			branchName: "bug/GITRU-111_some_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "bug/GITRU-111_some_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-111"}},
				},
			},
		},
		{
			branchName: "GITRU-111_GITRU-112_GITRU-113_some-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GITRU-111_GITRU-112_GITRU-113_some-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-111"}, {Code: "GITRU-112"}, {Code: "GITRU-113"}},
				},
			},
		},
		{
			branchName: "GIT_RU-111_some-description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "GIT_RU-111_some-description",
					LinkedUnits: []gitnames.UnitCode{{Code: "GIT_RU-111"}},
				},
			},
		},
		{
			branchName: "feature/TSKM_NW_RP-2",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/TSKM_NW_RP-2",
					LinkedUnits: []gitnames.UnitCode{{Code: "TSKM_NW_RP-2"}},
				},
			},
		},
		{
			branchName: "feature/TSKM_NW_RP-2_some_description",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "feature/TSKM_NW_RP-2_some_description",
					LinkedUnits: []gitnames.UnitCode{{Code: "TSKM_NW_RP-2"}},
				},
			},
		},
		{
			branchName: "TSKM_NW_RP-2",
			want: gitnames.BranchLinks{
				Base: gitnames.Base{
					Description: "TSKM_NW_RP-2",
					LinkedUnits: []gitnames.UnitCode{{Code: "TSKM_NW_RP-2"}},
				},
			},
		},
	}

	b := branchParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Parse(tt.branchName)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_branchParser_Parse_negative(t *testing.T) {
	targetErr := gitnamesparser.NewUnitCodeNotFoundError("")

	tests := []struct {
		name  string
		value string
		want  error
	}{
		{value: "", want: targetErr},
		{value: "GITRU", want: targetErr},
		{value: "some text", want: targetErr},
	}

	b := branchParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := b.Parse(tt.value)
			require.ErrorAs(t, err, &tt.want)
		})
	}
}
