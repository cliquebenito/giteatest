package unit_links

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateDiff(t *testing.T) {
	tests := []struct {
		name string
		old  AllUnitLinks
		new  AllUnitLinks
		want Diff
	}{
		{
			old: AllUnitLinks{{ID: 1, FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			new: AllUnitLinks{{ID: 1, FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			want: Diff{
				LinksToAdd:    nil,
				LinksToDelete: nil,
			},
		},
		{
			old: AllUnitLinks{{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			new: AllUnitLinks{{FromUnitID: 2, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			want: Diff{
				LinksToAdd:    AllUnitLinks{{FromUnitID: 2, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
				LinksToDelete: AllUnitLinks{{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			},
		},
		{
			old: AllUnitLinks{
				{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"},
				{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"},
			},
			new: AllUnitLinks{
				{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"},
			},
			want: Diff{
				LinksToAdd:    nil,
				LinksToDelete: AllUnitLinks{{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-1"}},
			},
		},
		{
			old: nil,
			new: AllUnitLinks{
				{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"},
			},
			want: Diff{
				LinksToAdd: AllUnitLinks{
					{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"},
				},
				LinksToDelete: nil,
			},
		},
		{
			old: nil, new: nil, want: Diff{LinksToAdd: nil, LinksToDelete: nil},
		},
		{
			old:  AllUnitLinks{{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"}},
			new:  nil,
			want: Diff{LinksToAdd: nil, LinksToDelete: AllUnitLinks{{FromUnitID: 1, FromUnitType: PullRequestFromUnitType, ToUnitID: "GITRU-2"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateDiff(tt.old, tt.new)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
