// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import "reflect"

var Cron struct {
	// CheckDelayUnlocked Задержка при проверке на то, что cron работа была разблокирована вручную, по умолчанию 60 секунд
	CheckDelayUnlocked int64
}

// GetCronSettings maps the cron subsection to the provided config
func GetCronSettings(name string, config interface{}) (interface{}, error) {
	return getCronSettings(CfgProvider, name, config)
}

func getCronSettings(rootCfg ConfigProvider, name string, config interface{}) (interface{}, error) {
	if err := rootCfg.Section("cron." + name).MapTo(config); err != nil {
		return config, err
	}

	typ := reflect.TypeOf(config).Elem()
	val := reflect.ValueOf(config).Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := val.Field(i)
		tpField := typ.Field(i)
		if tpField.Type.Kind() == reflect.Struct && tpField.Anonymous {
			if err := rootCfg.Section("cron." + name).MapTo(field.Addr().Interface()); err != nil {
				return config, err
			}
		}
	}

	return config, nil
}

// loadCron подтягивает настройки из конфигурационного файла
func loadCron(rootCfg ConfigProvider) {
	sec := rootCfg.Section("cron")

	Cron.CheckDelayUnlocked = sec.Key("CHECK_DELAY_UNLOCKED").MustInt64(60)
}
