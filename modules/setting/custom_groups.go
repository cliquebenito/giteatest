package setting

import (
	"strings"
	"unicode"

	"code.gitea.io/gitea/modules/log"
)

// SourceControlCustomGroups настройки кастомных групп привилегий
var SourceControlCustomGroups struct {
	//Enabled Активирована ли функционал кастомных групп привилегий
	Enabled bool
	//CustomGroups Кастомные группы привилегий
	CustomGroups
}

// CustomGroup Кастомная группа привилегий
type CustomGroup struct {
	Name       string
	Privileges string
}

const (
	maxCustomGroupNameLength = 100  // максимальная длина имени группы
	maxPrivilegesAmount      = 10   // максимальное количество привилегий
	maxGroup                 = 1003 // максимальное количество групп
)

// CustomGroups Кастомные группы привилегий
type CustomGroups map[string]CustomGroup

// loadSourceControlCustomGroups подтягивает настройки из конфигурационного файла
func loadSourceControlCustomGroups(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sourcecontrol.custom_groups")
	maxGroups := 0
	SourceControlCustomGroups.Enabled = sec.Key("ENABLED").MustBool(false)
	if SourceControlCustomGroups.Enabled && SourceControl.Enabled && SourceControl.TenantWithRoleModeEnabled {
		SourceControlCustomGroups.CustomGroups = make(map[string]CustomGroup)

		for _, value := range sec.ChildSections() {
			secName := strings.TrimPrefix(value.Name(), "sourcecontrol.custom_groups.")

			if hasNonLatinLetters(secName) {
				log.Fatal("Custom group code '%s' is incorrect", secName)
			}
			if len(value.Key("NAME").Value()) > maxCustomGroupNameLength {
				log.Fatal("Name of custom group '%s' is incorrect", value.Key("NAME").Value())
			}

			// проверка на дубликаты в PRIVILEGES
			rawPrivs := value.Key("PRIVILEGES").MustString("")
			privSlice := strings.Split(rawPrivs, ",")
			seen := make(map[string]struct{}, len(privSlice))
			for _, p := range privSlice {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if _, exists := seen[p]; exists {
					log.Fatal("Duplicate privilege '%s' found in custom group '%s'", p, secName)
				}
				seen[p] = struct{}{}
			}

			if len(seen) > maxPrivilegesAmount {
				log.Fatal("Too many privileges defined in custom group '%s'. Max allowed is 10, got %d", secName, len(seen))
			}
			group := CustomGroup{
				Name:       value.Key("NAME").MustString(secName),
				Privileges: rawPrivs,
			}
			SourceControlCustomGroups.CustomGroups[secName] = group
			maxGroups++
		}
		if maxGroups > maxGroup {
			log.Fatal("Too many custom group. Max allowed is %d", maxGroup)
		}
	}
}

// hasNonLatinLetters проверяет, что группа содержит только латинские буквы(без пробелов)
func hasNonLatinLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) || (r < 'A' || (r > 'Z' && r < 'a') || r > 'z') {
			return true
		}
	}
	return false
}
