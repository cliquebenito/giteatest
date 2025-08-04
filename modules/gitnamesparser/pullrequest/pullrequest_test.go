package pullrequest

import (
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/gitnames"
)

func Test_pullRequestNameParser_Parse(t *testing.T) {
	tests := []struct {
		name            string
		pullRequestName string
		branchName      string
		commitNames     []string
		want            gitnames.PullRequestLinks
	}{
		{pullRequestName: "(GIT_RU_1-112) Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "(GIT_RU_1-112) Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU_1-112"}}}}},
		{pullRequestName: "GIT_RU-111 Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "GIT_RU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU-111"}}}}},
		{pullRequestName: "(GIT_RU-111) Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "(GIT_RU-111) Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU-111"}}}}},
		{pullRequestName: "[GIT_RU-111] Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "[GIT_RU-111] Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU-111"}}}}},
		{pullRequestName: "GITRU-111 GIT_RU-112 Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "GITRU-111 GIT_RU-112 Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GITRU-111"}, {"GIT_RU-112"}}}}},
		{pullRequestName: "[GITRU-111] (GIT_RU-112) Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "[GITRU-111] (GIT_RU-112) Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GITRU-111"}, {"GIT_RU-112"}}}}},
		{pullRequestName: "[GITRU-111](GIT_RU-112) Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "[GITRU-111](GIT_RU-112) Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GITRU-111"}, {"GIT_RU-112"}}}}},
		{pullRequestName: "GITRU-111;GIT_RU-112;Pull Request", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "GITRU-111;GIT_RU-112;Pull Request", LinkedUnits: gitnames.LinkedUnits{{"GITRU-111"}, {"GIT_RU-112"}}}}},
		{pullRequestName: "GITRU-111;GIT_RU-112;Запрос на слияние", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "GITRU-111;GIT_RU-112;Запрос на слияние", LinkedUnits: gitnames.LinkedUnits{{"GITRU-111"}, {"GIT_RU-112"}}}}},
		{pullRequestName: "Запрос на слияние GIT_RU-112", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "Запрос на слияние GIT_RU-112", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU-112"}}}}},
		{pullRequestName: "[GIT_RU-111] Запрос на слияние GIT_RU-112", want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "[GIT_RU-111] Запрос на слияние GIT_RU-112", LinkedUnits: gitnames.LinkedUnits{{"GIT_RU-111"}, {"GIT_RU-112"}}}}},

		{pullRequestName: "", commitNames: []string{"GITRU-1"}, want: gitnames.PullRequestLinks{CommitsLinks: []gitnames.CommitLinks{{Base: gitnames.Base{Description: "GITRU-1", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-1"}}}}}}},
		{pullRequestName: "GITRU-1", commitNames: nil, want: gitnames.PullRequestLinks{Base: gitnames.Base{Description: "GITRU-1", LinkedUnits: gitnames.LinkedUnits{{"GITRU-1"}}}}},

		{
			pullRequestName: "GITRU-111 Pull Request",
			commitNames:     []string{"GITRU-112 Commit"},
			want: gitnames.PullRequestLinks{
				Base: gitnames.Base{Description: "GITRU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-111"}}},
				CommitsLinks: []gitnames.CommitLinks{
					{Base: gitnames.Base{Description: "GITRU-112 Commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-112"}}}},
				},
			},
		},

		{
			pullRequestName: "GIT_RU-111 Pull Request",
			commitNames: []string{
				"(GIT_RU-112)Commit", "[GIT_RU-113] Some other commit", "GIT_RU-114: one more commit", "[GIT-11]GIT-22(GIT-3) desc"},
			want: gitnames.PullRequestLinks{
				Base: gitnames.Base{Description: "GIT_RU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-111"}}},
				CommitsLinks: []gitnames.CommitLinks{
					{Base: gitnames.Base{Description: "(GIT_RU-112)Commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-112"}}}},
					{Base: gitnames.Base{Description: "[GIT_RU-113] Some other commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-113"}}}},
					{Base: gitnames.Base{Description: "GIT_RU-114: one more commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-114"}}}},
					{Base: gitnames.Base{Description: "[GIT-11]GIT-22(GIT-3) desc", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT-11"}, {Code: "GIT-22"}, {Code: "GIT-3"}}}},
				},
			},
		},

		{
			pullRequestName: "GIT_RU-111 Pull Request",
			commitNames: []string{
				"(GIT_RU-112)Commit", "[GIT_RU-113] Some other commit", "GIT_RU-114: one more commit", "[GIT-11]GIT-22(GIT-3) desc"},
			want: gitnames.PullRequestLinks{
				Base: gitnames.Base{Description: "GIT_RU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-111"}}},
				CommitsLinks: []gitnames.CommitLinks{
					{Base: gitnames.Base{Description: "(GIT_RU-112)Commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-112"}}}},
					{Base: gitnames.Base{Description: "[GIT_RU-113] Some other commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-113"}}}},
					{Base: gitnames.Base{Description: "GIT_RU-114: one more commit", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-114"}}}},
					{Base: gitnames.Base{Description: "[GIT-11]GIT-22(GIT-3) desc", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT-11"}, {Code: "GIT-22"}, {Code: "GIT-3"}}}},
				},
			},
		},
		{
			pullRequestName: "GIT_RU-111 Pull Request",
			branchName:      "feature/GITRU-1126",
			want: gitnames.PullRequestLinks{
				Base:        gitnames.Base{Description: "GIT_RU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-111"}}},
				BranchLinks: gitnames.BranchLinks{Base: gitnames.Base{Description: "feature/GITRU-1126", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-1126"}}}},
			},
		},
		{
			pullRequestName: "GIT_RU-111 Pull Request",
			branchName:      "feature/GITRU-1126",
			commitNames:     []string{"GITRU-12"},
			want: gitnames.PullRequestLinks{
				Base: gitnames.Base{Description: "GIT_RU-111 Pull Request", LinkedUnits: gitnames.LinkedUnits{{Code: "GIT_RU-111"}}},
				BranchLinks: gitnames.BranchLinks{
					Base: gitnames.Base{Description: "feature/GITRU-1126", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-1126"}}},
				},
				CommitsLinks: []gitnames.CommitLinks{
					{Base: gitnames.Base{Description: "GITRU-12", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-12"}}}},
				},
			},
		},
		{
			pullRequestName: "Pull Request",
			branchName:      "feature/GITRU-13108-description",
			want: gitnames.PullRequestLinks{
				BranchLinks: gitnames.BranchLinks{
					Base: gitnames.Base{Description: "feature/GITRU-13108-description", LinkedUnits: gitnames.LinkedUnits{{Code: "GITRU-13108"}}},
				},
			},
		},
	}

	p := pullRequestLinksParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := pull_request_reader.PullRequestHeader{
				PullRequestName: tt.pullRequestName,
				BranchName:      tt.branchName,
				CommitNames:     tt.commitNames,
			}
			got, err := p.Parse(header)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
