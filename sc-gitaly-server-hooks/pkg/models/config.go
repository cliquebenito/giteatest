package models

type Config struct {
	SourceControl *SourceControlCofig `toml:"sourcecontrol,omitempty"`
	GitalyHooks   *HooksConfig        `toml:"gitaly_hooks,omitempty"`
}

func (c Config) SourceControlConfig() *SourceControlCofig {
	return c.SourceControl
}

func (c Config) GitalyHooksConfig() *HooksConfig {
	return c.GitalyHooks
}

type SourceControlCofig struct {
	Address string `toml:"ADDRESS,omitempty"`
	Token   string `toml:"TOKEN,omitempty"`
}

func (c SourceControlCofig) GetAddress() string {
	if c.Address[len(c.Address)-1:] != "/" {
		c.Address += "/"
	}
	return c.Address
}

func (c SourceControlCofig) GetToken() string {
	return c.Token
}

type HooksConfig struct {
	ConfigPath   string `toml:"CONFIG_PATH,omitempty"`
	LogPath      string `toml:"LOG_PATH,omitempty"`
	LogErrorPath string `toml:"LOG_ERROR_PATH,omitempty"`
	LogLevel     string `toml:"LOG_LEVEL,omitempty"`
}

func (c HooksConfig) GetConfigPath() string {
	if c.ConfigPath == "" {
		c.ConfigPath = HookGitalyConfigPath
		return c.ConfigPath
	}
	return c.ConfigPath
}

func (c HooksConfig) GetLogPath() string {
	if c.LogPath == "" {
		c.LogPath = HookLogPath
		return c.LogPath
	}
	return c.LogPath
}

func (c HooksConfig) GetLogErrorPath() string {
	if c.LogPath == "" {
		c.LogPath = HookLogErrorPath
		return c.LogPath
	}
	return c.LogPath
}

func (c HooksConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		c.LogLevel = HookLogLevel
		return c.LogLevel
	}
	return c.LogLevel
}
