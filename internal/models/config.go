package models

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

type BackupConfig struct {
	RepoConfigs []*gitalypb.Repository
}

func (b BackupConfig) JSON() ([]byte, error) {
	var buffer bytes.Buffer

	for _, repoConfig := range b.RepoConfigs {
		repoConfigBody, err := json.Marshal(repoConfig)
		if err != nil {
			return nil, fmt.Errorf("marshal repo config: %w", err)
		}

		_, err = buffer.Write(repoConfigBody)
		if err != nil {
			return nil, fmt.Errorf("write repo config: %w", err)
		}
	}

	return buffer.Bytes(), nil
}
