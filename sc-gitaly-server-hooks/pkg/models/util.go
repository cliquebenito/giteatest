package models

const (
	HookConfigPath       = "/etc/gitaly/hooks/config.toml"
	HookGitalyConfigPath = "/etc/gitaly/config.toml"
	HookLogPath          = "/var/log/gitaly/hooks/hooks.log"
	HookLogErrorPath     = "/var/log/gitaly/hooks/hooks_error.log"
	HookLogLevel         = "info"

	EnvRepoUsername = "GL_PROJECT_PATH"
	EnvRepoName     = "GL_REPOSITORY"
	EnvPusherID     = "GL_ID"
	EnvPusherName   = "GL_USERNAME"

	GitAlternativeObjectDirectories = "GIT_ALTERNATE_OBJECT_DIRECTORIES"
	GitObjectDirectory              = "GIT_OBJECT_DIRECTORY"
	GitQuarantinePath               = "GIT_QUARANTINE_PATH"
	GitPushOptionCount              = "GIT_PUSH_OPTION_COUNT"

	HookBatchSize = 30
	BranchPrefix  = "refs/heads/"
	TagPrefix     = "refs/tags/"
	EmptySHA      = "0000000000000000000000000000000000000000"
)
