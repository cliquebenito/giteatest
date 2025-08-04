package utils

import "testing"

func TestIsRuleNameSpecial(t *testing.T) {
	tests := []struct {
		ruleName string
		expected bool
	}{
		{"master", false},
		{"release/8.3.0", false},
		{"master", false},
		{"develop", false},
		{"feature/*", true},
		{"bugfix/.*", true},
		{"support/.*", true},
		{"test/.*", true},
		{".*", true},
		{".*", true},
	}

	for _, test := range tests {
		if result := IsRuleNameSpecial(test.ruleName); result != test.expected {
			t.Errorf("IsRuleNameSpecial(%q) = %v, want %v", test.ruleName, result, test.expected)
		}
	}
}
