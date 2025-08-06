package protected_branch

import (
	"testing"

	"github.com/gobwas/glob"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		protectBranch *ProtectedBranch
		branchName    string
		expected      bool
	}{
		{&ProtectedBranch{RuleName: "master"}, "master", true},
		{&ProtectedBranch{RuleName: "master"}, "master_sc", false},
		{&ProtectedBranch{RuleName: "develop"}, "develop", true},
		{&ProtectedBranch{RuleName: "develop"}, "develop_sc", false},
		{&ProtectedBranch{RuleName: "bugfix/*"}, "bugfix/bugfix1", true},
		{&ProtectedBranch{RuleName: "bugfix/*"}, "bugfix1/bugfix2", false},
		{&ProtectedBranch{RuleName: "hotfix/**"}, "hotfix/hotfix1", true},
		{&ProtectedBranch{RuleName: "release/.*"}, "release/release1", false},
		{&ProtectedBranch{RuleName: "test/.*"}, "test/test1", false},
		{&ProtectedBranch{RuleName: ".*"}, "branch1", false},
		{&ProtectedBranch{RuleName: "*"}, "branch2", true},
		{&ProtectedBranch{RuleName: "**"}, "branch3", true},
	}

	for _, test := range tests {
		if result := test.protectBranch.Match(test.branchName); result != test.expected {
			t.Errorf("Match(%q, %q) = %v, want %v", test.protectBranch.RuleName, test.branchName, result, test.expected)
		}
	}
}

func TestLoadGlob(t *testing.T) {
	tests := []struct {
		name     string
		branch   ProtectedBranch
		expected glob.Glob
		isPlain  bool
	}{
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "master",
			},
			expected: glob.MustCompile("master", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "feature/*",
			},
			expected: glob.MustCompile("feature/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "bugfix/*",
			},
			expected: glob.MustCompile("bugfix/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "release/*",
			},
			expected: glob.MustCompile("release/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "hotfix/*",
			},
			expected: glob.MustCompile("hotfix/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "develop",
			},
			expected: glob.MustCompile("develop", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "feature/123",
			},
			expected: glob.MustCompile("feature/123", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "bugfix/456",
			},
			expected: glob.MustCompile("bugfix/456", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "release/789",
			},
			expected: glob.MustCompile("release/789", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is plain",
			branch: ProtectedBranch{
				GlobRule: nil,
				RuleName: "hotfix/012",
			},
			expected: glob.MustCompile("hotfix/012", '/'),
			isPlain:  true,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: glob.MustCompile("feature/*", '/'),
				RuleName: "feature/123",
			},
			expected: glob.MustCompile("feature/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: glob.MustCompile("bugfix/*", '/'),
				RuleName: "bugfix/456",
			},
			expected: glob.MustCompile("bugfix/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: glob.MustCompile("release/*", '/'),
				RuleName: "release/789",
			},
			expected: glob.MustCompile("release/*", '/'),
			isPlain:  false,
		},
		{
			name: "RuleName is glob pattern",
			branch: ProtectedBranch{
				GlobRule: glob.MustCompile("hotfix/*", '/'),
				RuleName: "hotfix/012",
			},
			expected: glob.MustCompile("hotfix/*", '/'),
			isPlain:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			globRule, isPlain := test.branch.LoadGlob()
			if !globRule.Match(test.branch.RuleName) {
				t.Errorf("Expected glob rule to match %s, but it did not", test.branch.RuleName)
			}
			if isPlain != test.isPlain {
				t.Errorf("Expected isPlain to be %v, but got %v", test.isPlain, isPlain)
			}
		})
	}
}
