package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/routers/common"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	archiverService "code.gitea.io/gitea/services/repository/archiver"
	"errors"
	"net/http"
	"path"
	"time"
)

// SingleDownload метод скачивания файла по имени репозитория и treePath пути до файла.
// Так же можно скачать файл с определенного гит референса (имени ветки или хэш-коммита)
// источник routers/web/repo/download.go#SingleDownload
func SingleDownload(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	blob, lastModified := getBlobForEntry(ctx, log)
	if blob == nil {
		return
	}

	if err := common.ServeBlob(ctx.Base, ctx.Repo.TreePath, blob, lastModified); err != nil {
		log.Error("An error has occurred while to try get file: %s from repoId: %s, error: %v", ctx.Repo.TreePath, ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}
}

// getBlobForEntry метод получения файла по пути до файла
func getBlobForEntry(ctx *context.Context, log logger.Logger) (blob *git.Blob, lastModified time.Time) {
	entry, err := ctx.Repo.Commit.GetTreeEntryByPath(ctx.Repo.TreePath)
	if err != nil {
		if git.IsErrNotExist(err) {
			log.Debug("File: %s not found in repo with repoId: %d, error: %v", ctx.Repo.TreePath, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(ctx.Repo.TreePath))

		} else {
			log.Error("An error has occurred while try get file: %s in repoId: %d, error: %v", ctx.Repo.TreePath, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if entry.IsDir() || entry.IsSubModule() {
		log.Debug("Unable to get file: %s entry in repoId: %d because it is not file, error: %v", ctx.Repo.TreePath, ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusBadRequest, apiError.InvalidFilePath(ctx.Repo.TreePath))

		return
	}

	info, _, err := git.Entries([]*git.TreeEntry{entry}).GetCommitsInfo(ctx, ctx.Repo.Commit, path.Dir("/" + ctx.Repo.TreePath)[1:])
	if err != nil {
		log.Error("An error has occurred while try get commit info, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	if len(info) == 1 {
		// Not Modified
		lastModified = info[0].Commit.Committer.When
	}
	blob = entry.Blob()

	return blob, lastModified
}

// DownloadArchive метод скачивания архива репозитория
func DownloadArchive(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	uri := ctx.Params("*")
	archReq, err := archiverService.NewRequest(ctx.Repo.Repository.ID, ctx.Repo.GitRepo, uri)
	if err != nil {
		if errors.Is(err, archiverService.ErrUnknownArchiveFormat{}) {
			validationError := []apiError.ValidationError{{
				ErrorMessage: "Suffix must be in (.zip, .tar.gz, .bundle). Example: master.zip",
				FieldName:    "/{archieve}",
			}}
			log.Debug("Unknown archive format. Wrong suffix in path param: %s", uri)
			ctx.JSON(http.StatusBadRequest, apiError.RequestFieldValidationError("Validation error has occurred", validationError))
		} else if errors.Is(err, archiverService.RepoRefNotFoundError{}) {
			log.Debug("No such reference: %s in repository with repoId: %d", err.(archiverService.RepoRefNotFoundError).RefName, ctx.Repo.Repository.ID)
			ctx.JSON(http.StatusBadRequest, apiError.GitReferenceNotExist(err.(archiverService.RepoRefNotFoundError).RefName))
		} else {
			log.Error("Unknown error has occurred while creating new archive request for repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	archiver, err := archReq.Await(ctx)
	if err != nil {
		log.Error("An error has occurred while waiting for completion of archive request of repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	downloadName := ctx.Repo.Repository.Name + "-" + archReq.GetArchiveName()

	rPath := archiver.RelativePath()
	if setting.RepoArchive.ServeDirect {
		// If we have a signed url (S3, object storage), redirect to this directly.
		u, err := storage.RepoArchives.URL(rPath, downloadName)
		if u != nil && err == nil {
			ctx.Redirect(u.String())
			return
		}
	}

	// If we have matched and access to release or issue
	fr, err := storage.RepoArchives.Open(rPath)
	if err != nil {
		log.Error("An error has occurred while opening storage for getting repoId: %d archive. Error: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	defer fr.Close()

	ctx.ServeContent(fr, &context.ServeHeaderOptions{
		Filename:     downloadName,
		LastModified: archiver.CreatedUnix.AsLocalTime(),
	})
}
