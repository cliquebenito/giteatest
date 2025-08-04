package repo_server

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"code.gitea.io/gitea/modules/trace"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/upload"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/gitdiff"
)

const (
	tplCompare base.TplName = "repo/diff/compare"
	tplDiffBox base.TplName = "repo/diff/box"
)

var pullRequestTemplateCandidates = []string{
	"PULL_REQUEST_TEMPLATE.md",
	"PULL_REQUEST_TEMPLATE.yaml",
	"PULL_REQUEST_TEMPLATE.yml",
	"pull_request_template.md",
	"pull_request_template.yaml",
	"pull_request_template.yml",
	".gitea/PULL_REQUEST_TEMPLATE.md",
	".gitea/PULL_REQUEST_TEMPLATE.yaml",
	".gitea/PULL_REQUEST_TEMPLATE.yml",
	".gitea/pull_request_template.md",
	".gitea/pull_request_template.yaml",
	".gitea/pull_request_template.yml",
	".github/PULL_REQUEST_TEMPLATE.md",
	".github/PULL_REQUEST_TEMPLATE.yaml",
	".github/PULL_REQUEST_TEMPLATE.yml",
	".github/pull_request_template.md",
	".github/pull_request_template.yaml",
	".github/pull_request_template.yml",
}

// CompareDiff show different from one commit to another commit
func (s *Server) CompareDiff(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Errorf("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Errorf("Error has occurred while creating trace time message: %v", err)
		}
	}()

	ci := repo.ParseCompareInfo(ctx)
	defer func() {
		if ci != nil && ci.HeadGitRepo != nil {
			ci.HeadGitRepo.Close()
		}
	}()
	if ctx.Written() {
		return
	}

	ctx.Data["PullRequestWorkInProgressPrefixes"] = setting.Repository.PullRequest.WorkInProgressPrefixes
	ctx.Data["DirectComparison"] = ci.DirectComparison
	ctx.Data["OtherCompareSeparator"] = ".."
	ctx.Data["CompareSeparator"] = "..."
	if ci.DirectComparison {
		ctx.Data["CompareSeparator"] = ".."
		ctx.Data["OtherCompareSeparator"] = "..."
	}

	nothingToCompare := repo.PrepareCompareDiff(ctx, ci,
		gitdiff.GetWhitespaceFlag(ctx.Data["WhitespaceBehavior"].(string)))
	if ctx.Written() {
		return
	}

	baseTags, err := repo_model.GetTagNamesByRepoID(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		ctx.ServerError("GetTagNamesByRepoID", err)
		return
	}
	ctx.Data["Tags"] = baseTags

	fileOnly := ctx.FormBool("file-only")
	if fileOnly {
		ctx.HTML(http.StatusOK, tplDiffBox)
		return
	}

	headBranches, _, err := ci.HeadGitRepo.GetBranchNames(0, 0)
	if err != nil {
		ctx.ServerError("GetBranches", err)
		return
	}

	action := role_model.WRITE
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         action,
	})
	if err != nil {
		log.Errorf("Error has occurred while checking user's permission: %v", err)
		ctx.ServerError("Error has occurred while checking user's permission: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil {
			log.Errorf("Error has occurred while checking user's permission: %v", err)
			ctx.ServerError("Error has occurred while checking user's permission: %v", err)
			return
		}
		if !allow {
			ctx.Error(http.StatusForbidden, "You don't have permission to view this branch.")
			return
		}
	}

	ctx.Data["HeadBranches"] = headBranches

	headTags, err := repo_model.GetTagNamesByRepoID(ctx, ci.HeadRepo.ID)
	if err != nil {
		ctx.ServerError("GetTagNamesByRepoID", err)
		return
	}
	ctx.Data["HeadTags"] = headTags

	if ctx.Data["PageIsComparePull"] == true {
		pr, err := issues_model.GetUnmergedPullRequest(ctx, ci.HeadRepo.ID, ctx.Repo.Repository.ID, ci.HeadBranch, ci.BaseBranch, issues_model.PullRequestFlowGithub)
		if err != nil {
			if !issues_model.IsErrPullRequestNotExist(err) {
				ctx.ServerError("GetUnmergedPullRequest", err)
				return
			}
		} else {
			ctx.Data["HasPullRequest"] = true
			if err := pr.LoadIssue(ctx); err != nil {
				ctx.ServerError("LoadIssue", err)
				return
			}
			ctx.Data["PullRequest"] = pr
			ctx.HTML(http.StatusOK, repo.TplCompareDiff)
			return
		}

		if !nothingToCompare {
			// Setup information for new form.
			repo.RetrieveRepoMetas(ctx, ctx.Repo.Repository, true)
			if ctx.Written() {
				return
			}
		}
	}
	beforeCommitID := ctx.Data["BeforeCommitID"].(string)
	afterCommitID := ctx.Data["AfterCommitID"].(string)

	separator := "..."
	if ci.DirectComparison {
		separator = ".."
	}
	ctx.Data["Title"] = "Comparing " + base.ShortSha(beforeCommitID) + separator + base.ShortSha(afterCommitID)

	ctx.Data["IsRepoToolbarCommits"] = true
	ctx.Data["IsDiffCompare"] = true
	templateErrs := repo.SetTemplateIfExists(ctx, pullRequestTemplateKey, pullRequestTemplateCandidates)

	if len(templateErrs) > 0 {
		ctx.Flash.Warning(s.renderErrorOfTemplates(ctx, templateErrs), true)
	}

	if content, ok := ctx.Data["content"].(string); ok && content != "" {
		// If a template content is set, prepend the "content". In this case that's only
		// applicable if you have one commit to compare and that commit has a message.
		// In that case the commit message will be prepend to the template body.
		if templateContent, ok := ctx.Data[pullRequestTemplateKey].(string); ok && templateContent != "" {
			// Re-use the same key as that's priortized over the "content" key.
			// Add two new lines between the content to ensure there's always at least
			// one empty line between them.
			ctx.Data[pullRequestTemplateKey] = content + "\n\n" + templateContent
		}

		// When using form fields, also add content to field with id "body".
		if fields, ok := ctx.Data["Fields"].([]*api.IssueFormField); ok {
			for _, field := range fields {
				if field.ID == "body" {
					if fieldValue, ok := field.Attributes["value"].(string); ok && fieldValue != "" {
						field.Attributes["value"] = content + "\n\n" + fieldValue
					} else {
						field.Attributes["value"] = content
					}
				}
			}
		}
	}

	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")

	ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWrite(unit.TypePullRequests)

	if unit, err := ctx.Repo.Repository.GetUnit(ctx, unit.TypePullRequests); err == nil {
		config := unit.PullRequestsConfig()
		ctx.Data["AllowMaintainerEdit"] = config.DefaultAllowMaintainerEdit
	} else {
		ctx.Data["AllowMaintainerEdit"] = false
	}

	ctx.HTML(http.StatusOK, tplCompare)
}

func (s *Server) renderErrorOfTemplates(ctx *context.Context, errs map[string]error) string {
	var files []string
	for k := range errs {
		files = append(files, k)
	}
	sort.Strings(files) // keep the output stable

	var lines []string
	for _, file := range files {
		lines = append(lines, fmt.Sprintf("%s: %v", file, errs[file]))
	}

	flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
		"Message": ctx.Tr("repo.Issues.choose.ignore_invalid_templates"),
		"Summary": ctx.Tr("repo.Issues.choose.invalid_templates", len(errs)),
		"Details": utils.SanitizeFlashErrorString(strings.Join(lines, "\n")),
	})
	if err != nil {
		log.Debugf("render flash error: %v", err)
		flashError = ctx.Tr("repo.Issues.choose.ignore_invalid_templates")
	}
	return flashError
}
