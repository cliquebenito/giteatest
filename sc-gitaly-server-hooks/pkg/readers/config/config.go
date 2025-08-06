package config

import (
	"bytes"
	"fmt"
	"io"

	"sc-gitaly-server-hooks/pkg/models"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
)

type Reader struct{}

func NewConfigReader() Reader {
	return Reader{}
}

func (c Reader) Read(fs afero.Fs, path string) (*models.Config, error) {
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return &models.Config{}, err
	}

	return c.fromReader(bytes.NewReader(b))
}

func (c Reader) fromReader(reader io.Reader) (*models.Config, error) {
	config := &models.Config{}
	if err := toml.NewDecoder(reader).Decode(config); err != nil {
		return &models.Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	return config, nil
}
