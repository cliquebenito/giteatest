package sbt

import (
	"bytes"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	baseLog "code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/common"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/orgs"
	"code.gitea.io/gitea/routers/sbt/repo"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/routers/sbt/user"
	authService "code.gitea.io/gitea/services/auth"
	contextService "code.gitea.io/gitea/services/context"
	gocontext "context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"gitea.com/go-chi/binding"
	"github.com/go-chi/cors"
	swagger "github.com/swaggo/http-swagger/v2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

/*
Routes создание роутера "/sbt/api/v1"
*/
func Routes(ctx gocontext.Context) *web.Route {
	reqSignIn := authService.VerifyAuthWithOptionsSbt(&authService.VerifyOptions{SignInRequired: true})
	reqRepoAdmin := authService.RequireRepoAdmin()
	reqRepoEditor := authService.RequireRepoEditor()
	reqRepoCodeWriter := authService.RequireRepoWriter(unit.TypeCode)
	reqRepoPullsWriter := authService.RequireRepoWriter(unit.TypePullRequests)

	optionalSignIn := authService.VerifyAuthWithOptionsSbt(&authService.VerifyOptions{SignInRequired: setting.Service.RequireSignInView})
	optionalExploreSignIn := authService.VerifyAuthWithOptionsSbt(&authService.VerifyOptions{SignInRequired: setting.Service.RequireSignInView || setting.Service.Explore.RequireSigninView})

	m := web.NewRoute()

	if setting.CORSConfig.Enabled {
		m.Use(cors.Handler(cors.Options{
			AllowedOrigins:   setting.CORSConfig.AllowDomain,
			AllowedMethods:   setting.CORSConfig.Methods,
			AllowCredentials: setting.CORSConfig.AllowCredentials,
			MaxAge:           int(setting.CORSConfig.MaxAge.Seconds()),
		}))
	}

	var mid []any

	mid = append(mid, common.Sessioner(), context.Contexter())

	group := buildAuthGroup()
	if err := group.Init(ctx); err != nil {
		baseLog.Error("Could not initialize '%s' auth method, error: %s", group.Name(), err)
	}

	mid = append(mid, authService.Auth(group))

	m.Use(mid...)

	m.Post("/signUp", checkProofOfWork, bind(request.RegisterUser{}, "create user"), user.PostCreateUser)
	m.Post("/signIn", checkProofOfWork, bind(request.SignIn{}, "sign in"), user.AuthUser)
	m.Post("/signOut", user.LogoutUser)

	m.Group("/repos", func() {
		m.Post("", reqSignIn, bind(request.CreateRepo{}, "create repo"), repo.CreateRepo)

		m.Group("/{username}/{reponame}", func() {
			m.Get("", contextService.RepoAssigmentSbt, repo.GetRepo)
			m.Delete("", reqSignIn, contextService.RequireRepoOwner(), contextService.RepoAssigmentByAuthSbt(), repo.DeleteRepo)

			m.Group("/branches", func() {
				m.Post("", reqSignIn, repoMustNotBeArchived, reqRepoCodeWriter, bind(request.CreateRepoBranch{}, "create branch"), repo.CreateBranch)
				m.Get("", repo.GetRepoBranchesList)
				m.Get("/*", repo.GetRepoBranch)
				m.Put("/rename", reqSignIn, repoMustNotBeArchived, reqRepoCodeWriter, bind(request.UpdateBranchName{}, "update branch name"), repo.RenameBranch)
				m.Delete("/*", reqSignIn, repoMustNotBeArchived, reqRepoCodeWriter, repo.DeleteRepoBranch)
			}, contextService.GitRepoAssigmentSbt, repoMustBeNotEmpty, contextService.RepoRefByTypeSbt(context.RepoRefBranch))
			m.Get("/branches_list", contextService.GitRepoAssigmentSbt, repoMustBeNotEmpty, contextService.RepoRefByTypeSbt(context.RepoRefBranch), repo.GetRepoBranchesNamesList)

			m.Group("/commits", func() {
				m.Get("", repo.GetRepoCommitsList)
				m.Get("/{sha}", repo.GetRepoCommit)
				m.Get("/{sha}/diff", repo.GetRepoCommitDiff)
			}, contextService.GitRepoAssigmentSbt, repoMustBeNotEmpty, contextService.RepoRefByTypeSbt(context.RepoRefBranch))

			m.Group("/contents", func() {
				m.Get("", contextService.RepoAssigmentSbt, repo.GetRootContentsList)
				m.Get("/*", contextService.RepoAssigmentSbt, repo.GetContents)
				m.Group("/*", func() {
					m.Post("", bind(request.CreateFileOptions{}, "create file"), repo.CreateFile)
					m.Put("", bind(request.UpdateFileOptions{}, "update file"), repo.UpdateFile)
					m.Delete("", bind(request.DeleteFileOptions{}, "delete file"), repo.DeleteFile)
				}, reqSignIn, contextService.RepoAssigmentSbt, repoMustNotBeArchived, reqRepoEditor)
			})

			m.Group("/raw", func() {
				m.Get("/branch/*", contextService.RepoRefByTypeSbt(context.RepoRefBranch), repo.SingleDownload)
				m.Get("/commit/*", contextService.RepoRefByTypeSbt(context.RepoRefCommit), repo.SingleDownload)
			}, contextService.RepoAssigmentSbt)

			m.Get("/archive/*", contextService.GitRepoAssigmentSbt, repoMustBeNotEmpty, repo.DownloadArchive)

			m.Group("/pulls", func() {
				m.Post("", reqSignIn, repoMustNotBeArchived, reqRepoCodeWriter, bind(request.CreatePullRequest{}, "create pull request"), repo.CreatePullRequest)
				m.Get("", repo.ListPullRequests)
				m.Group("/{index}", func() {
					m.Get("", repo.GetPullRequest)
					m.Group("/comments", func() {
						m.Group("", func() {
							m.Post("", reqSignIn, repoMustNotBeArchived, bind(request.CreateComment{}, "create comment"), repo.CreateComment)
							m.Get("", repo.GetComments)
						})
						m.Group("/{id}", func() {
							m.Delete("", repo.DeleteComment)
							m.Put("", bind(request.UpdateComment{}, "update comment"), repo.UpdateComment)
							m.Get("/history", repo.GetCommentHistory)
							m.Get("/history/detail", repo.GetCommentHistoryDetail)
							m.Delete("/history/delete", repo.SoftDeleteCommentHistory)
							m.Post("/reactions/{action}", bind(request.Reaction{}, "put reaction"), repo.ChangeCommentReaction)
						}, reqSignIn, repoMustNotBeArchived)
					})

					m.Post("/lock", reqSignIn, reqRepoPullsWriter, bind(request.IssueLock{}, "lock comments"), repo.LockIssue)
					m.Post("/unlock", reqSignIn, reqRepoPullsWriter, repo.UnlockIssue)

					m.Patch("/status", reqSignIn, repoMustNotBeArchived, bind(request.ChangePullRequestStatus{}, "change pull request's status"), repo.ChangePullRequestStatus)
					m.Patch("/reviewer", reqSignIn, repoMustNotBeArchived, bind(request.ChangePullRequestReviewer{}, "change pull request's reviewers"), repo.ChangePullRequestReviewer)
					m.Get("/diff", repo.GetPullRequestDiff)
					m.Get("/commits", repo.GetPullRequestCommits)
					m.Get("/files", repo.GetPullRequestDiffFileList)
					m.Post("/merge", reqSignIn, repoMustNotBeArchived, reqRepoCodeWriter, bind(request.MergePullRequest{}, "merge pull request"), repo.CreateMergePullRequest)
				})
				m.Post("/attachments", reqSignIn, repoMustNotBeArchived, repo.UploadIssueAttachment)
				m.Delete("/attachments/{uuid}", reqSignIn, repoMustNotBeArchived, repo.DeleteAttachment)
			}, contextService.GitRepoAssigmentSbt, repoMustBeNotEmpty)

			m.Group("/compare", func() {
				m.Get("/diff/*", repo.GetRepoBranchesDiff)
				m.Get("/commits/*", repo.GetRepoBranchesCommitDiff)
				m.Get("/files/*", repo.GetRepoBranchesDiffFileList)
			}, contextService.GitRepoAssigmentSbt)

			//repo settings
			m.Group("/settings", func() {
				m.Patch("", bind(request.RepoBaseSettingsOptional{}, "update repository settings"), repo.UpdateRepoSettings)
				m.Patch("/pulls", bind(request.RepoPullsSettingsOptional{}, "update repository pulls settings"), repo.UpdateRepoPullsSettings)

				m.Group("/collaboration", func() {
					m.Get("", repo.GetCollaboration)
					m.Post("/{collaborator}", repo.CreateCollaboration)
					m.Put("/{collaborator}/{action}", repo.ChangeCollaborationAccessMode)
					m.Delete("/{collaborator}", repo.DeleteCollaboration)
				},
				)
			}, reqSignIn, contextService.RepoAssigmentSbt, reqRepoAdmin, context.RepoRef())

			m.Group("/settings", func() {
				m.Post("/archive", repo.Archive)
				m.Post("/unarchive", repo.Unarchive)
				m.Post("/transfer", bind(request.TransferRepoOptional{}, "transfer repository"), repo.Transfer)
			}, reqSignIn, contextService.RepoAssigmentSbt, contextService.RequireRepoOwner(), context.RepoRef())

			m.Post("/forks", reqSignIn, contextService.RepoAssigmentSbt, bind(request.ForkRepo{}, "fork repo"), repo.ForkRepo)

			m.Patch("/action/{action}", reqSignIn, contextService.RepoAssigmentSbt, repo.UpdateAction)
			m.Get("/watchers", contextService.RepoAssigmentSbt, repo.GetWatchersList)
			m.Get("/stargazers", contextService.RepoAssigmentSbt, repo.GetStargazersList)
		})
		m.Post("/migrate", reqSignIn, bind(request.MigrateRepo{}, "migrate repo"), repo.Migrate)

		m.Get("/search", optionalExploreSignIn, repo.Search)
	})

	m.Group("/user", func() {
		m.Get("", user.GetAuthenticatedUser)
		m.Group("/dashboard", func() {
			m.Get("", user.GetActivities)
			m.Get("/heatmap", user.GetActivityHeatMap)
		})
		m.Group("/settings", func() {
			m.Get("", user.GetUserSettings)
			m.Patch("", bind(request.UserSettingsOptional{}, "update user settings"), user.UpdateUserSettings)
			m.Delete("/account/delete", bind(request.DeleteUserAccount{}, "delete user account"), user.DeleteAccountUser)
			m.Group("/avatar", func() {
				m.Post("", bind(request.UserAvatar{}, "update user avatar"), user.UpdateAvatar)
				m.Delete("", user.DeleteAvatar)
			})
			m.Put("/password", bind(request.ChangePassword{}, "change password"), user.ChangePassword)
		})
		m.Group("/keys", func() {
			m.Post("", bind(request.CreateUserSshKey{}, "create user ssh key"), user.CreateSshKey)
			m.Get("", user.GetListSshKey)
			m.Group("/{keyId}", func() {
				m.Get("", user.GetUserSshKeyById)
				m.Delete("", user.DeleteUserSshKeyById)
			})
		})
		m.Get("/watching", repo.GetCurrentUserWatchingRepoList)
		m.Get("/orgs", orgs.GetCurrentUserListOrgs)
	}, reqSignIn)

	m.Group("/users", func() {
		m.Group("/{username}", func() {
			m.Get("", optionalExploreSignIn, user.GetUserInfo)
			m.Get("/repos", optionalExploreSignIn, repo.ListUserRepos)
			m.Group("/dashboard", func() {
				m.Get("", user.GetUserActivities)
				m.Get("/heatmap", user.GetUserActivityHeatMap)
			}, optionalExploreSignIn)
		}, contextService.UserAssignmentSbt())
	})

	m.Get("/users/search", optionalExploreSignIn, user.Search)

	m.Group("/orgs", func() {
		m.Post("", bind(request.CreateOrganizationOptional{}, "create organization"), orgs.CreateOrganization)
		m.Group("/{org}/settings", func() {
			m.Get("", orgs.GetOrgSettings)
			m.Patch("", bind(request.UpdateOrgSettingsOptional{}, "update organization's settings"), orgs.UpdateOrgSettings)
			m.Group("/avatar", func() {
				m.Post("", bind(request.OrganizationAvatar{}, "update organization's avatar"), orgs.UpdateAvatar)
				m.Delete("", orgs.DeleteAvatar)
			})
		}, orgAssignment(), reqOrgOwnership())
	}, reqSignIn)

	m.Get("/orgs/{org}", optionalSignIn, orgAssignment(), orgs.GetOrganizationByName)

	m.Group("/explore", func() {
		m.Get("", func(ctx *context.Context) {
			ctx.Redirect(setting.AppSubURL + "/explore/repos")
		})
		m.Get("/repos", repo.SearchRepos)
		m.Get("/users", user.Search)
		m.Get("/orgs", orgs.SearchOrgs)
	}, optionalExploreSignIn)

	m.Get("/attachments/{uuid}", repo.GetAttachment)

	//todo need to be deleted - only for development needs
	// страница с контрактом доступна по этому адресу: http://localhost:3000/sbt/api/v1/openapi/view/index.html#/
	m.Group("/openapi", func() {
		m.Get("/view/*", swagger.Handler(
			swagger.URL("../doc.yaml"),
		))
		m.Get("/doc.yaml", openapiYaml)
	})

	return m
}

func buildAuthGroup() *authService.Group {
	group := authService.NewGroup(
		&authService.Session{},
	)

	return group
}

// bind an obj to a func(ctx *context.Context)
func bind[T any](_ T, method string) any {
	return func(ctx *context.Context) {
		theObj := new(T) // create a new form obj for every request but not use obj directly
		errs := binding.Bind(ctx.Req, theObj)
		if len(errs) > 0 {
			apiError.HandleValidationErrors(ctx, errs, method)
			return
		}
		web.SetForm(ctx, theObj)
	}
}

// repoMustBeNotEmpty проверяет что репозиторий не пустой git каталог
func repoMustBeNotEmpty(ctx *context.Context) {
	if ctx.Repo.Repository.IsEmpty {
		ctx.JSON(http.StatusForbidden, apiError.RepoIsEmpty())
	}
}

// repoMustNotBeArchived проверяет что репозиторий не заархивирован
func repoMustNotBeArchived(ctx *context.Context) {
	if ctx.Repo.Repository.IsArchived {
		ctx.JSON(http.StatusForbidden, apiError.RepoIsArchived())
	}
}

// orgAssignment метод, который "подтягивает" организацию в контекст.
// Метод написан по аналогии с методом orgAssignment из v1/api.go (из метода были удалены команды).
func orgAssignment() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		log := logger.Logger{}
		log.SetTraceId(ctx)

		ctx.Org = new(context.Organization)

		var err error
		ctx.Org.Organization, err = organization.GetOrgByName(ctx, ctx.Params(":org"))
		if err != nil {
			if organization.IsErrOrgNotExist(err) {
				log.Debug("Organization with name: %s is not exist", ctx.Params(":org"))
				ctx.JSON(http.StatusBadRequest, apiError.OrganizationNotFoundByNameError(ctx.Params(":org")))
			} else {
				log.Error("While getting organization with name: %s unknown error type has occurred: %v", ctx.Params(":org"), err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			}
			return
		}
		ctx.ContextUser = ctx.Org.Organization.AsUser()
	}
}

// reqOrgOwnership пользователь должен быть владельцем организации или администратором сайта
func reqOrgOwnership() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		log := logger.Logger{}
		log.SetTraceId(ctx)

		if ctx.IsUserSiteAdmin() {
			return
		}

		var orgID int64
		if ctx.Org.Organization != nil {
			orgID = ctx.Org.Organization.ID
		} else if ctx.Org.Team != nil {
			orgID = ctx.Org.Team.OrgID
		} else {
			log.Error("There is no organization in context")
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		isOwner, err := organization.IsOrganizationOwner(ctx, orgID, ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while checking is current user with username: %s owner of organization: %s", ctx.Doer.Name, ctx.Org.Organization.Name)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return

		} else if !isOwner {
			if ctx.Org.Organization != nil {
				log.Debug("User with username: %s is not owner of organization: %s", ctx.Doer.Name, ctx.Org.Organization.Name)
				ctx.JSON(http.StatusBadRequest, apiError.UserIsNotOwner())
			}
			return
		}
	}
}

func openapiYaml(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	pwd, _ := os.Getwd()

	yamlBytes, err := os.ReadFile(filepath.Join(pwd, "docs/gitverse-openapi.yaml"))
	if err != nil {
		log.Error("Can not read swagger file ", err)
		ctx.Status(http.StatusNotFound)
	} else {
		ctx.PlainTextBytes(http.StatusOK, yamlBytes)
	}
}

// jsonWithPayload структура для получения поля payload из тела запроса для проверки Proof-Of-Work (остальные поля запроса отбрасываются)
type jsonWithPayload struct {
	Payload *int
}

// checkProofOfWork Проверка Proof-Of-Work запросов на регистрацию и аутентификацию
// подробности тут: https://dzo.sw.sbc.space/wiki/display/GITRU/Proof-Of-Work
func checkProofOfWork(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if setting.EnableProofOfWork {
		proofOfWork := ctx.Req.Header.Get("Proof-Of-Work")
		if len(proofOfWork) == 0 {
			log.Debug("Proof-Of-Work is enabled, but header not exist")
			ctx.JSON(http.StatusBadRequest, apiError.ProofOfWorkValidation())
			return
		}

		body, err := io.ReadAll(ctx.Req.Body)
		if err != nil {
			log.Debug("Error reading request body, error: %v", err)
			ctx.JSON(http.StatusBadRequest, apiError.ProofOfWorkValidation())
			return
		}
		ctx.Req.Body = io.NopCloser(bytes.NewBuffer(body))

		var j jsonWithPayload
		err = json.Unmarshal(body, &j)
		if err != nil {
			log.Debug("Error while unmarshal request body, err: %v", err)
			ctx.JSON(http.StatusBadRequest, apiError.ProofOfWorkValidation())
			return
		}

		if j.Payload == nil {
			log.Debug("Proof-Of-Work is enabled, but payload is nil")
			ctx.JSON(http.StatusBadRequest, apiError.ProofOfWorkValidation())
			return
		}

		sha := newSHA256(body)
		bodyHash := hex.EncodeToString(sha)

		if proofOfWork != bodyHash || !setting.RegexpWithZeroCount.MatchString(proofOfWork) {
			log.Error("Proof-Of-Work validation failed: header value: %s, computed body hash: %s, payload: %v, zero count: %d", proofOfWork, bodyHash, *j.Payload, setting.ZeroCount)
			ctx.JSON(http.StatusBadRequest, apiError.ProofOfWorkValidation())
			return
		}
	}
}

// newSHA256 вычисляет SHA265, предварительно удалив из строки все переносы строк и пробелы
func newSHA256(data []byte) []byte {
	trimmedText := strings.ReplaceAll(string(data), "\n", "") //удалим переводы строк
	trimmedText = strings.ReplaceAll(trimmedText, " ", "")    // удалим пробелы
	hash := sha256.Sum256([]byte(trimmedText))
	return hash[:]
}
