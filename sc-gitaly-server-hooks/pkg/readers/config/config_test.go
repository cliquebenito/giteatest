package config

import (
	"testing"

	"sc-gitaly-server-hooks/pkg/models"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestConfig_Read(t *testing.T) {
	configReader := NewConfigReader()

	tests := []struct {
		name       string
		fs         afero.Fs
		configBody []byte
		path       string
		want       *models.Config
	}{
		{
			name: "Correct config",
			fs:   afero.NewMemMapFs(),
			configBody: []byte(`
[sourcecontrol]
ADDRESS= 'http://localhost:3000/'
TOKEN = 'token'`,
			),
			path: "gitaly.config.toml",
			want: &models.Config{SourceControl: &models.SourceControlCofig{Address: "http://localhost:3000/", Token: "token"}},
		},
		{
			name: "Missing address",
			fs:   afero.NewMemMapFs(),
			configBody: []byte(`
[sourcecontrol]
TOKEN = 'token'`,
			),
			path: "gitaly.config.toml",
			want: &models.Config{SourceControl: &models.SourceControlCofig{Token: "token"}},
		},
		{
			name: "Missing token",
			fs:   afero.NewMemMapFs(),
			configBody: []byte(`
[sourcecontrol]
ADDRESS= 'http://localhost:3000/'`,
			),
			path: "gitaly.config.toml",
			want: &models.Config{SourceControl: &models.SourceControlCofig{Address: "http://localhost:3000/"}},
		},
		{
			name: "Empty SourceControl",
			fs:   afero.NewMemMapFs(),
			configBody: []byte(`
[sourcecontrol]
`,
			),
			path: "gitaly.config.toml",
			want: &models.Config{SourceControl: &models.SourceControlCofig{}},
		},
		{
			name:       "Missing SourceControl",
			fs:         afero.NewMemMapFs(),
			configBody: []byte(``),
			path:       "gitaly.config.toml",
			want:       &models.Config{},
		}, {
			name: "Correct hooks config",
			fs:   afero.NewMemMapFs(),
			configBody: []byte(`
[hooks]
CONFIG_PATH="/Users/21401747/GolandProjects/gitaly/gitaly.config.toml"
LOG_PATH="/Users/21401747/GolandProjects/sc-gitaly-server-hooks/logs.log"
LOG_ERROR_PATH="/Users/21401747/GolandProjects/sc-gitaly-server-hooks/logs_err"
LOG_LEVEL="debug""`,
			),
			path: "gitaly.config.toml",
			want: &models.Config{SourceControl: &models.SourceControlCofig{Address: "http://localhost:3000/", Token: "token"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			configFile, err := tt.fs.Create(tt.path)
			require.NoError(t, err)
			_, err = configFile.Write(tt.configBody)
			require.NoError(t, err)
			err = configFile.Close()
			require.NoError(t, err)

			got, err := configReader.Read(tt.fs, tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
