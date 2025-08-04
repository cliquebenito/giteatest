package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// FileOptionInterface интерфейс для операций с файлом
type FileOptionInterface interface {
	GetBranch() string
}

// CreateFileOptions Данные для создания файла
// `author` и `committer` опциональны (если указан только один, то эти же данные будут использованы и для второго,
// а если не указаны оба то будут использованы данные авторизованного пользователя)
type CreateFileOptions struct {
	FileOptions
	// content must be base64 encoded
	Content *string `json:"content" binding:"Required"`
}

// GetBranch Возвращает имя ветки, реализация интерфейса FileOptionInterface
func (o *CreateFileOptions) GetBranch() string {
	return o.FileOptions.BranchName
}

type UpdateFileOptions struct {
	FileOptions
	// content must be base64 encoded
	Content *string `json:"content" binding:"Required"`
	// sha is the SHA for the file that already exists
	SHA string `json:"sha" binding:"Required"`
	// from_path (optional) is the path of the original file which will be moved/renamed to the path in the URL
	FromPath string `json:"from_path" binding:"SbtMaxSize(500)"`
}

// GetBranch Возвращает имя ветки, реализация интерфейса FileOptionInterface
func (o UpdateFileOptions) GetBranch() string {
	return o.FileOptions.BranchName
}

// DeleteFileOptions Данные для удаления файла
// `author` и `committer` опциональны (если указан только один, то эти же данные будут использованы и для второго,
// а если не указаны оба то будут использованы данные авторизованного пользователя)
type DeleteFileOptions struct {
	FileOptions
	// sha is the SHA for the file that already exists
	SHA string `json:"sha" binding:"Required"`
}

// GetBranch Возвращает имя ветки, реализация интерфейса FileOptionInterface
func (o *DeleteFileOptions) GetBranch() string {
	return o.FileOptions.BranchName
}

// FileOptions Данные для создания файла
type FileOptions struct {
	// message (optional) for the commit of this file. if not supplied, a default message will be used
	Message string `json:"message"`
	// branch (optional) to base this file from. if not given, the default branch is used
	BranchName string `json:"branch" binding:"SbtGitRefName;SbtMaxSize(100)"`
	// new_branch (optional) will make a new branch from `branch` before creating the file
	NewBranchName string `json:"new_branch" binding:"SbtGitRefName;SbtMaxSize(100)"`
	// `author` and `committer` are optional (if only one is given, it will be used for the other, otherwise the authenticated user will be used)
	Author    Identity          `json:"author"`
	Committer Identity          `json:"committer"`
	Dates     CommitDateOptions `json:"dates"`
	// Add a Signed-off-by trailer by the committer at the end of the commit log message.
	Signoff bool `json:"signoff"`
}

func (f *CreateFileOptions) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
