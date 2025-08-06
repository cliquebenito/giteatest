package protected_brancher

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/require"
)

var getter *ProtectedBranchGetter

func init() {
	getter = NewProtectedBranchGetter()
}

func TestGetGlob(t *testing.T) {
	cases := []struct {
		name                string
		protectBranch       protected_branch.ProtectedBranch
		expectedGlob        glob.Glob
		expectedIsPlainName bool
	}{
		{
			name: "GlobRule is not nil",
			protectBranch: protected_branch.ProtectedBranch{
				GlobRule:    glob.MustCompile("pattern1", '/'),
				IsPlainName: true,
			},
			expectedGlob:        glob.MustCompile("pattern1", '/'),
			expectedIsPlainName: true,
		},
		{
			name: "GlobRule is nil, RuleName is valid glob pattern",
			protectBranch: protected_branch.ProtectedBranch{
				RuleName: "pattern1",
			},
			expectedGlob:        glob.MustCompile("pattern1", '/'),
			expectedIsPlainName: true,
		},
		{
			name: "GlobRule is nil, RuleName is global glob pattern",
			protectBranch: protected_branch.ProtectedBranch{
				RuleName: "*",
			},
			expectedGlob:        glob.MustCompile("*", '/'),
			expectedIsPlainName: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			globRule, isPlainName := getter.GetGlob(nil, tc.protectBranch)
			require.Equal(t, tc.expectedGlob, globRule)
			require.Equal(t, tc.expectedIsPlainName, isPlainName)
		})
	}
}

func TestIsRuleNameSpecial(t *testing.T) {
	cases := []struct {
		name           string
		ruleName       string
		expectedResult bool
	}{
		{
			name:           "ruleName is empty",
			ruleName:       "",
			expectedResult: false,
		},
		{
			name:           "ruleName contains only non-special characters",
			ruleName:       "pattern1",
			expectedResult: false,
		},
		{
			name:           "ruleName contains special characters",
			ruleName:       "*",
			expectedResult: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := getter.IsRuleNameSpecial(tc.ruleName)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestGetProtectedFilePatterns(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		expectedResult []glob.Glob
	}{
		{
			name: "ProtectedFilePatterns is empty",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "",
			},
			expectedResult: []glob.Glob{},
		},
		{
			name: "ProtectedFilePatterns contains only valid glob expressions",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "pattern1;pattern2",
			},
			expectedResult: []glob.Glob{glob.MustCompile("pattern1", '.', '/'), glob.MustCompile("pattern2", '.', '/')},
		},
		{
			name: "ProtectedFilePatterns contains global glob expressions",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "*;pattern1",
			},
			expectedResult: []glob.Glob{glob.MustCompile("*", '.', '/'), glob.MustCompile("pattern1", '.', '/')},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := getter.GetProtectedFilePatterns(nil, tc.protectBranch)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestGetUnprotectedFilePatterns(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		expectedResult []glob.Glob
	}{
		{
			name: "UnprotectedFilePatterns is empty",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "",
			},
			expectedResult: []glob.Glob{},
		},
		{
			name: "UnprotectedFilePatterns contains only valid glob expressions",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "pattern1;pattern2",
			},
			expectedResult: []glob.Glob{glob.MustCompile("pattern1", '.', '/'), glob.MustCompile("pattern2", '.', '/')},
		},
		{
			name: "UnprotectedFilePatterns contains global glob expressions",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "*;pattern1",
			},
			expectedResult: []glob.Glob{glob.MustCompile("*", '.', '/'), glob.MustCompile("pattern1", '.', '/')},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := getter.GetUnprotectedFilePatterns(nil, tc.protectBranch)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}
