package git_hooks

// ScGitHookType тип для описания типов git хука
type ScGitHookType string

// Перечисление типов git хука
const (
	PreReceive  ScGitHookType = "pre-receive"
	Update      ScGitHookType = "update"
	PostReceive ScGitHookType = "post-receive"
)
