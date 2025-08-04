package iamtokenparser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewErrorIAMTenantNotFound(t *testing.T) {
	t.Run("ErrorIAMTenantNotFound check", func(t *testing.T) {
		gotErr := NewErrorIAMTenantNotFound(fmt.Errorf("test error"))
		targetErr := &ErrorIAMTenantNotFound{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}

func TestNewErrorIAMClaimNotExists(t *testing.T) {
	t.Run("ErrorIAMClaimNotExists check", func(t *testing.T) {
		gotErr := NewErrorIAMClaimNotExists(fmt.Errorf("test error"))
		targetErr := &ErrorIAMClaimNotExists{}
		require.ErrorAs(t, gotErr, &targetErr)
	})
}
