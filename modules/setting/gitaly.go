package setting

import (
	"strings"

	"code.gitea.io/gitea/modules/log"
)

// Gitaly настройки специфичные только для Gitaly
var Gitaly struct {
	//Enabled Активирован ли Gitaly
	Enabled bool

	//MainServerName имя основного сервера Gitaly в который отправляются запросы
	MainServerName string

	// todo подумать когда использовать другой сервер, а не основной
	// GitalyServers список серверов Gitaly ключом является название хранилища
	GitalyServers
}

// loadSourceControl подтягивает настройки из конфигурационного файла
func loadGitaly(rootCfg ConfigProvider) {
	sec := rootCfg.Section("gitaly")

	Gitaly.Enabled = sec.Key("ENABLED").MustBool(false)
	if Gitaly.Enabled {
		serversSec := rootCfg.Section("gitaly.servers")
		Gitaly.GitalyServers = make(map[string]ServerInfo, 0)
		firstServer := strings.TrimPrefix(serversSec.ChildSections()[0].Name(), "gitaly.servers.")

		for _, v := range serversSec.ChildSections() {
			server := ServerInfo{
				Address: v.Key("ADDRESS").MustString(""),
				Token:   v.Key("TOKEN").MustString(""),
			}
			Gitaly.GitalyServers[strings.TrimPrefix(v.Name(), "gitaly.servers.")] = server
		}

		Gitaly.MainServerName = sec.Key("MAIN_SERVER_NAME").MustString(firstServer)

		if _, ok := Gitaly.GitalyServers[Gitaly.MainServerName]; !ok {
			log.Fatal("Incorrect name of Gitaly Main Server '%s' or this server not configured", Gitaly.MainServerName)
		}
	}
}

// ServerInfo содержит информацию о том, как связаться с сервером Gitaly или Praefect.
// Это необходимо для RPC-систем Gitaly, в которых задействовано более одного Gitaly.
// Без этой информации Gitaly не знала бы, как связаться с удаленным узлом.
type ServerInfo struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}

// GitalyServers хранит информация о серверах Gitaly, такую как {"default":{"token":"x","address":"y"}},
// которая должна быть передана в метаданных `gitaly-servers`.
type GitalyServers map[string]ServerInfo
