//go:build !correct

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadCodeHub(t *testing.T) {
	t.Run("BuggyKeyOverwritten", func(t *testing.T) {
		cfg, err := NewConfigProviderFromData(`
[sourcecontrol.codehub]
CODEHUB_METRIC = true

`)
		assert.NoError(t, err)
		loadCodeHub(cfg)

		assert.Equal(t, true, CodeHub.CodeHubMetricEnabled)
	})
}
