package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	files_service "code.gitea.io/gitea/services/repository/files"
	"net/http"
	"time"
)

// UpdateFile изменение файла в репозитории, если указана ветка то в этой ветке, если нет то в ветке по умолчанию
func UpdateFile(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	updateFileOptions := web.GetForm(ctx).(*request.UpdateFileOptions)

	if updateFileOptions.BranchName == "" {
		updateFileOptions.BranchName = ctx.Repo.Repository.DefaultBranch
	}

	opts := &files_service.UpdateRepoFileOptions{
		Content:      *updateFileOptions.Content,
		SHA:          updateFileOptions.SHA,
		IsNewFile:    false,
		Message:      updateFileOptions.Message,
		FromTreePath: updateFileOptions.FromPath,
		TreePath:     ctx.Params("*"),
		OldBranch:    updateFileOptions.BranchName,
		NewBranch:    updateFileOptions.NewBranchName,
		Committer: &files_service.IdentityOptions{
			Name:  updateFileOptions.Committer.Name,
			Email: updateFileOptions.Committer.Email,
		},
		Author: &files_service.IdentityOptions{
			Name:  updateFileOptions.Author.Name,
			Email: updateFileOptions.Author.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    updateFileOptions.Dates.Author,
			Committer: updateFileOptions.Dates.Committer,
		},
		Signoff: updateFileOptions.Signoff,
	}
	if opts.Dates.Author.IsZero() {
		opts.Dates.Author = time.Now()
	}
	if opts.Dates.Committer.IsZero() {
		opts.Dates.Committer = time.Now()
	}

	if opts.Message == "" {
		opts.Message = ctx.Tr("repo.editor.update", opts.TreePath)
	}

	if fileResponse, err := createOrUpdateFile(ctx, opts); err != nil {
		handleCreateOrUpdateFileError(ctx, err, log)
	} else {
		ctx.JSON(http.StatusOK, fileResponse)
	}
}
