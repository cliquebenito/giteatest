package utils

import "github.com/gobwas/glob/syntax"

// IsRuleNameSpecial return true if it contains special character
// Deprecated: use ProtectedBranchManager.isRuleNameSpecial
func IsRuleNameSpecial(ruleName string) bool {
	for i := 0; i < len(ruleName); i++ {
		if syntax.Special(ruleName[i]) {
			return true
		}
	}
	return false
}
