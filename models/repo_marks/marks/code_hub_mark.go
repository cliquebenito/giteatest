package marks

import "code.gitea.io/gitea/models/repo_marks"

// Не менять, так как это лишь внутренний ключ, связи потеряются
const codeHubMarkKey = "code_hub"

// Assert interface satisfaction
var _ repo_marks.RepoMark = codeHubMark{}

type codeHubMark struct {
	label string
	key   string
}

func (c codeHubMark) Label() string {
	return c.label
}

func (c codeHubMark) Key() string {
	return c.key
}

// GetCodeHubMark returns code hub mark
func GetCodeHubMark(labelName string) repo_marks.RepoMark {
	return codeHubMark{label: labelName, key: codeHubMarkKey}
}
