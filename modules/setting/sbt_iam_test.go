//go:build !correct

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseWhiteListRolesUser(t *testing.T) {
	iniValid := `
[iam]
WHITE_LIST_ROLES_USER = role1
`

	iniEmpty := `
[iam]
`

	tests := []struct {
		name     string
		ini      string
		expected []string
	}{
		{
			name: "valid input",
			ini:  iniValid,
			expected: []string{
				"role1",
			},
		},
		{
			name:     "empty config",
			ini:      iniEmpty,
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData(tt.ini)
			require.NoError(t, err)

			sec := cfg.Section("iam")

			got := parseWhiteListRolesUser(sec)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_parseWhiteListRolesAdmin(t *testing.T) {
	iniValid := `
[iam]
WHITE_LIST_ROLES_ADMIN = admin1
`

	iniEmpty := `
[iam]
`

	tests := []struct {
		name     string
		ini      string
		expected []string
	}{
		{
			name: "valid input",
			ini:  iniValid,
			expected: []string{
				"admin1",
			},
		},
		{
			name:     "empty config",
			ini:      iniEmpty,
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData(tt.ini)
			require.NoError(t, err)

			sec := cfg.Section("iam")

			got := parseWhiteListRolesAdmin(sec)

			assert.Equal(t, tt.expected, got)
		})
	}
}
