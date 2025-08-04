package enver

import (
	"encoding/base64"
	"fmt"
	"os"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
)

type Enver struct {
	PostgresDSN       string
	PostgresSchema    string
	GitalyStorageName string
	RepoRootPath      string
}

func New(withSc bool) (Enver, error) {
	pgDSN, exists := os.LookupEnv("POSTGRES_DSN")
	if !exists {
		return Enver{}, fmt.Errorf("env POSTGRES_DSN is empty")
	}

	pgSchema, exists := os.LookupEnv("POSTGRES_SCHEMA")
	if !exists {
		return Enver{}, fmt.Errorf("env POSTGRES_SCHEMA is empty")
	}

	rawGitalyServers, exists := os.LookupEnv("GITALY_SERVERS")
	if !exists {
		return Enver{}, fmt.Errorf("env GITALY_SERVERS is empty")
	}

	if withSc {
		gitalyServers := make(map[string]setting.ServerInfo)

		if err := unmarshalGitalyServers(rawGitalyServers, &gitalyServers); err != nil {
			return Enver{}, fmt.Errorf("injecting GITALY_SERVERS: %w", err)
		}

		if len(gitalyServers) > 1 {
			return Enver{}, fmt.Errorf("env GITALY_SERVERS contains more than one gitaly server")
		}

		gitalyStorageName := ""
		for storageName := range gitalyServers {
			gitalyStorageName = storageName
		}

		if gitalyStorageName == "" {
			return Enver{}, fmt.Errorf("env GITALY_STORAGE_NAME is empty")
		}

		repoRootPath, exists := os.LookupEnv("REPO_ROOT_PATH")
		if !exists {
			return Enver{}, fmt.Errorf("env REPO_ROOT_PATH is empty")
		}

		return Enver{PostgresDSN: pgDSN, PostgresSchema: pgSchema, GitalyStorageName: gitalyStorageName, RepoRootPath: repoRootPath}, nil
	}

	return Enver{PostgresDSN: pgDSN, PostgresSchema: pgSchema}, nil
}

func unmarshalGitalyServers(encoded string, servers *map[string]setting.ServerInfo) error {
	gitalyServersJSON, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("failed decoding base64: %w", err)
	}

	if err := json.Unmarshal(gitalyServersJSON, servers); err != nil {
		return fmt.Errorf("failed unmarshalling json: %w", err)
	}

	return nil
}
