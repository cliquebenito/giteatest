package repo_server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/typesniffer"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/forms"
	files_service "code.gitea.io/gitea/services/repository/files"
)

const (
	frmCommitChoiceDirect    string       = "direct"
	frmCommitChoiceNewBranch string       = "commit-to-new-branch"
	tplEditFile              base.TplName = "repo/editor/edit"
)

func renderCommitRights(ctx *context.Context) bool {
	canCommitToBranch, err := ctx.Repo.CanCommitToBranch(ctx, ctx.Doer)
	if err != nil {
		log.Error("CanCommitToBranch: %v", err)
	}
	ctx.Data["CanCommitToBranch"] = canCommitToBranch

	return canCommitToBranch.CanCommitToBranch
}

// getParentTreeFields returns list of parent tree names and corresponding tree paths
// based on given tree path.
func getParentTreeFields(treePath string) (treeNames, treePaths []string) {
	if len(treePath) == 0 {
		return treeNames, treePaths
	}

	treeNames = strings.Split(treePath, "/")
	treePaths = make([]string, len(treeNames))
	for i := range treeNames {
		treePaths[i] = strings.Join(treeNames[:i+1], "/")
	}
	return treeNames, treePaths
}

// NewFile render create file page
func (s *Server) NewFile(ctx *context.Context) {
	editFile(ctx, true)
}

// NewFilePost response for creating file
func (s *Server) NewFilePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.EditRepoFileForm)
	editFilePost(ctx, *form, true)
}

func editFilePost(ctx *context.Context, form forms.EditRepoFileForm, isNewFile bool) {
	canCommit := renderCommitRights(ctx)
	treeNames, treePaths := getParentTreeFields(form.TreePath)
	branchName := ctx.Repo.BranchName

	organizationEntity, err := organization.GetOrgByID(ctx, ctx.Repo.Owner.ID)
	if err != nil {
		log.Error("Error has occurred while getting organization by id: %v", err)
		ctx.ServerError("Error has occurred while getting organization by id: %v", err)
		return
	}

	ctx.Org.Organization = organizationEntity
	ctx.Org.IsOwner = true
	if form.CommitChoice == frmCommitChoiceNewBranch {
		branchName = form.NewBranchName
	}

	ctx.Data["PageIsEdit"] = true
	ctx.Data["PageHasPosted"] = true
	ctx.Data["IsNewFile"] = isNewFile
	ctx.Data["TreePath"] = form.TreePath
	ctx.Data["TreeNames"] = treeNames
	ctx.Data["TreePaths"] = treePaths
	ctx.Data["BranchLink"] = ctx.Repo.RepoLink + "/src/branch/" + util.PathEscapeSegments(ctx.Repo.BranchName)
	ctx.Data["FileContent"] = form.Content
	ctx.Data["commit_summary"] = form.CommitSummary
	ctx.Data["commit_message"] = form.CommitMessage
	ctx.Data["commit_choice"] = form.CommitChoice
	ctx.Data["new_branch_name"] = form.NewBranchName
	ctx.Data["last_commit"] = ctx.Repo.CommitID
	ctx.Data["PreviewableExtensions"] = strings.Join(markup.PreviewableExtensions(), ",")
	ctx.Data["LineWrapExtensions"] = strings.Join(setting.Repository.Editor.LineWrapExtensions, ",")
	ctx.Data["Editorconfig"] = GetEditorConfig(ctx, form.TreePath)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplEditFile)
		return
	}

	// Cannot commit to a an existing branch if user doesn't have rights
	if branchName == ctx.Repo.BranchName && !canCommit {
		ctx.Data["Err_NewBranchName"] = true
		ctx.Data["commit_choice"] = frmCommitChoiceNewBranch
		ctx.RenderWithErr(ctx.Tr("repo.editor.cannot_commit_to_protected_branch", branchName), tplEditFile, &form)
		return
	}

	// CommitSummary is optional in the web form, if empty, give it a default message based on add or update
	// `message` will be both the summary and message combined
	message := strings.TrimSpace(form.CommitSummary)
	if len(message) == 0 {
		if isNewFile {
			message = ctx.Tr("repo.editor.add", form.TreePath)
		} else {
			message = ctx.Tr("repo.editor.update", form.TreePath)
		}
	}
	form.CommitMessage = strings.TrimSpace(form.CommitMessage)
	if len(form.CommitMessage) > 0 {
		message += "\n\n" + form.CommitMessage
	}
	auditParams := map[string]string{
		"branch_name": branchName,
	}

	if _, err := files_service.CreateOrUpdateRepoFile(ctx, ctx.Repo.Repository, ctx.Doer, &files_service.UpdateRepoFileOptions{
		LastCommitID: form.LastCommit,
		OldBranch:    ctx.Repo.BranchName,
		NewBranch:    branchName,
		FromTreePath: ctx.Repo.TreePath,
		TreePath:     form.TreePath,
		Message:      message,
		Content:      strings.ReplaceAll(form.Content, "\r", ""),
		IsNewFile:    isNewFile,
		Signoff:      form.Signoff,
	}); err != nil {
		// This is where we handle all the errors thrown by files_service.CreateOrUpdateRepoFile
		if git.IsErrNotExist(err) {
			ctx.RenderWithErr(ctx.Tr("repo.editor.file_editing_no_longer_exists", ctx.Repo.TreePath), tplEditFile, &form)
		} else if git_model.IsErrLFSFileLocked(err) {
			ctx.Data["Err_TreePath"] = true
			ctx.RenderWithErr(ctx.Tr("repo.editor.upload_file_is_locked", err.(git_model.ErrLFSFileLocked).Path, err.(git_model.ErrLFSFileLocked).UserName), tplEditFile, &form)
		} else if models.IsErrFilenameInvalid(err) {
			ctx.Data["Err_TreePath"] = true
			ctx.RenderWithErr(ctx.Tr("repo.editor.filename_is_invalid", form.TreePath), tplEditFile, &form)
		} else if models.IsErrFilePathInvalid(err) {
			ctx.Data["Err_TreePath"] = true
			if fileErr, ok := err.(models.ErrFilePathInvalid); ok {
				switch fileErr.Type {
				case git.EntryModeSymlink:
					ctx.RenderWithErr(ctx.Tr("repo.editor.file_is_a_symlink", fileErr.Path), tplEditFile, &form)
				case git.EntryModeTree:
					ctx.RenderWithErr(ctx.Tr("repo.editor.filename_is_a_directory", fileErr.Path), tplEditFile, &form)
				case git.EntryModeBlob:
					ctx.RenderWithErr(ctx.Tr("repo.editor.directory_is_a_file", fileErr.Path), tplEditFile, &form)
				default:
					ctx.Error(http.StatusInternalServerError, err.Error())
				}
			} else {
				ctx.Error(http.StatusInternalServerError, err.Error())
			}
		} else if models.IsErrRepoFileAlreadyExists(err) {
			ctx.Data["Err_TreePath"] = true
			ctx.RenderWithErr(ctx.Tr("repo.editor.file_already_exists", form.TreePath), tplEditFile, &form)
		} else if git.IsErrBranchNotExist(err) {
			// For when a user adds/updates a file to a branch that no longer exists
			if branchErr, ok := err.(git.ErrBranchNotExist); ok {
				ctx.RenderWithErr(ctx.Tr("repo.editor.branch_does_not_exist", branchErr.Name), tplEditFile, &form)
			} else {
				ctx.Error(http.StatusInternalServerError, err.Error())
			}
		} else if models.IsErrBranchAlreadyExists(err) {
			// For when a user specifies a new branch that already exists
			ctx.Data["Err_NewBranchName"] = true
			if branchErr, ok := err.(models.ErrBranchAlreadyExists); ok {
				ctx.RenderWithErr(ctx.Tr("repo.editor.branch_already_exists", branchErr.BranchName), tplEditFile, &form)
				auditParams["error"] = "Branch already exists"
				audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			} else {
				ctx.Error(http.StatusInternalServerError, err.Error())
				auditParams["error"] = "Error has occurred while creating new branch"
				audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
		} else if models.IsErrCommitIDDoesNotMatch(err) {
			ctx.RenderWithErr(ctx.Tr("repo.editor.file_changed_while_editing", ctx.Repo.RepoLink+"/compare/"+util.PathEscapeSegments(form.LastCommit)+"..."+util.PathEscapeSegments(ctx.Repo.CommitID)), tplEditFile, &form)
		} else if git.IsErrPushOutOfDate(err) {
			ctx.RenderWithErr(ctx.Tr("repo.editor.file_changed_while_editing", ctx.Repo.RepoLink+"/compare/"+util.PathEscapeSegments(form.LastCommit)+"..."+util.PathEscapeSegments(form.NewBranchName)), tplEditFile, &form)
		} else if git.IsErrPushRejected(err) {
			errPushRej := err.(*git.ErrPushRejected)
			if len(errPushRej.Message) == 0 {
				ctx.RenderWithErr(ctx.Tr("repo.editor.push_rejected_no_message"), tplEditFile, &form)
			} else {
				flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
					"Message": ctx.Tr("repo.editor.push_rejected"),
					"Summary": ctx.Tr("repo.editor.push_rejected_summary"),
					"Details": utils.SanitizeFlashErrorString(errPushRej.Message),
				})
				if err != nil {
					ctx.ServerError("editFilePost.HTMLString", err)
					return
				}
				ctx.RenderWithErr(flashError, tplEditFile, &form)
			}
		} else {
			flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
				"Message": ctx.Tr("repo.editor.fail_to_update_file", form.TreePath),
				"Summary": ctx.Tr("repo.editor.fail_to_update_file_summary"),
				"Details": utils.SanitizeFlashErrorString(err.Error()),
			})
			if err != nil {
				ctx.ServerError("editFilePost.HTMLString", err)
				return
			}
			ctx.RenderWithErr(flashError, tplEditFile, &form)
		}
	}

	if ctx.Repo.Repository.IsEmpty {
		_ = repo_model.UpdateRepositoryCols(ctx, &repo_model.Repository{ID: ctx.Repo.Repository.ID, IsEmpty: false}, "is_empty")
	}

	if form.CommitChoice == frmCommitChoiceNewBranch && ctx.Repo.Repository.UnitEnabled(ctx, unit.TypePullRequests) {
		ctx.Redirect(ctx.Repo.RepoLink + "/compare/" + util.PathEscapeSegments(ctx.Repo.BranchName) + "..." + util.PathEscapeSegments(form.NewBranchName))
	} else {
		ctx.Redirect(ctx.Repo.RepoLink + "/src/branch/" + util.PathEscapeSegments(branchName) + "/" + util.PathEscapeSegments(form.TreePath))
	}
}

// EditFilePost response for editing file
func (s *Server) EditFilePost(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	form := web.GetForm(ctx).(*forms.EditRepoFileForm)

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.ContextUser.ID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         role_model.WRITE,
	})
	if err != nil {
		log.Error("Error has occurred while check user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, "User does not have enough custom privileges to edit file.")
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.ContextUser.ID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ChangeBranch.String(),
		})
		if err != nil || !allow {
			log.Error("Error has occurred while check user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, "User does not have enough custom privileges to edit file.")
			return
		}
	}

	editFilePost(ctx, *form, false)
}

func editFile(ctx *context.Context, isNewFile bool) {
	ctx.Data["PageIsEdit"] = true
	ctx.Data["IsNewFile"] = isNewFile
	canCommit := renderCommitRights(ctx)

	treePath := cleanUploadFileName(ctx.Repo.TreePath)
	if treePath != ctx.Repo.TreePath {
		if isNewFile {
			ctx.Redirect(path.Join(ctx.Repo.RepoLink, "_new", util.PathEscapeSegments(ctx.Repo.BranchName), util.PathEscapeSegments(treePath)))
		} else {
			ctx.Redirect(path.Join(ctx.Repo.RepoLink, "_edit", util.PathEscapeSegments(ctx.Repo.BranchName), util.PathEscapeSegments(treePath)))
		}
		return
	}

	// ctx.Repo.TreePath содержит путь от корня репозитория до файла или пустую строку в случае создания нового файла
	treeNames, treePaths := getParentTreeFields(ctx.Repo.TreePath)

	if !isNewFile {
		entry, err := ctx.Repo.Commit.GetTreeEntryByPath(ctx.Repo.TreePath)
		if err != nil {
			ctx.NotFoundOrServerError("GetTreeEntryByPath", git.IsErrNotExist, fmt.Errorf("editFile error: %w", err))
			return
		}

		// Нельзя редактировать директорию
		if entry.IsDir() {
			ctx.Error(http.StatusBadRequest, "Нельзя редактировать директорию.")
			return
		}

		blob := entry.Blob()
		if blob.Size() >= setting.UI.MaxDisplayFileSize {
			ctx.NotFound("blob.Size", fmt.Errorf("editFile error: %v", err))
			return
		}

		dataRc, err := blob.DataAsync()
		if err != nil {
			ctx.NotFound("blob.Data", fmt.Errorf("editFile error: %v", err))
			return
		}

		defer dataRc.Close()

		ctx.Data["FileSize"] = blob.Size()
		ctx.Data["FileName"] = blob.Name()

		buf := make([]byte, 1024)
		n, _ := util.ReadAtMost(dataRc, buf)
		buf = buf[:n]

		// Only some file types are editable online as text.
		if !typesniffer.DetectContentType(buf).IsRepresentableAsText() {
			log.Error("Error has occurred while trying to edit uneditable file with extension %s", path.Ext(ctx.Repo.TreePath))
			ctx.Data["error"] = "Содержимое данного файла не подлежит изменению."
			ctx.Error(http.StatusBadRequest, "Содержимое данного файла не подлежит изменению.")
			return
		}

		d, _ := io.ReadAll(dataRc)
		if err := dataRc.Close(); err != nil {
			log.Error("Error whilst closing blob data: %v", err)
		}

		buf = append(buf, d...)
		if content, err := charset.ToUTF8WithErr(buf); err != nil {
			log.Error("ToUTF8WithErr: %v", err)
			ctx.Data["FileContent"] = string(buf)
		} else {
			ctx.Data["FileContent"] = content
		}
	} else {
		// Пустая строка означает пустое название создаваемого файла
		treeNames = append(treeNames, "")
	}

	ctx.Data["TreeNames"] = treeNames
	ctx.Data["TreePaths"] = treePaths
	ctx.Data["BranchLink"] = ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL()
	ctx.Data["commit_summary"] = ""
	ctx.Data["commit_message"] = ""
	if canCommit {
		ctx.Data["commit_choice"] = frmCommitChoiceDirect
	} else {
		ctx.Data["commit_choice"] = frmCommitChoiceNewBranch
	}
	ctx.Data["new_branch_name"] = repo.GetUniquePatchBranchName(ctx)
	ctx.Data["last_commit"] = ctx.Repo.CommitID
	ctx.Data["PreviewableExtensions"] = strings.Join(markup.PreviewableExtensions(), ",")
	ctx.Data["LineWrapExtensions"] = strings.Join(setting.Repository.Editor.LineWrapExtensions, ",")
	ctx.Data["Editorconfig"] = GetEditorConfig(ctx, treePath)

	ctx.HTML(http.StatusOK, tplEditFile)
}

// GetEditorConfig returns a editorconfig JSON string for given treePath or "null"
func GetEditorConfig(ctx *context.Context, treePath string) string {
	ec, _, err := ctx.Repo.GetEditorconfig()
	if err == nil {
		def, err := ec.GetDefinitionForFilename(treePath)
		if err == nil {
			jsonStr, _ := json.Marshal(def)
			return string(jsonStr)
		}
	}
	return "null"
}

// EditFile render edit file page TODO +
func (s *Server) EditFile(ctx *context.Context) {
	editFile(ctx, false)
}

func cleanUploadFileName(name string) string {
	// Rebase the filename
	name = util.PathJoinRel(name)
	// Git disallows any filenames to have a .git directory in them.
	for _, part := range strings.Split(name, "/") {
		if strings.ToLower(part) == ".git" {
			return ""
		}
	}
	return name
}
