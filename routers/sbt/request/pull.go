package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"fmt"
	"gitea.com/go-chi/binding"
	"net/http"
	"time"
)

// CreatePullRequest параметры создания PR
type CreatePullRequest struct {
	// ветка из которой происходит слияние
	Head string `json:"head" binding:"Required"`
	// ветка в которую происходит слияние
	Base string `json:"base" binding:"Required"`
	// заголовок запроса
	Title string `json:"title" binding:"Required"`
	// тело запроса
	Body string `json:"body"`
	// список ответственных
	Assignees []string `json:"assignees"`
	// идентификатор "этапа"
	Milestone int64 `json:"milestone"`
	// Метки
	Labels []int64 `json:"labels"`
	// Deadline
	Deadline *time.Time `json:"due_date"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *CreatePullRequest) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

func (f *CreatePullRequest) String() string {
	return fmt.Sprintf("[ Head: %s, Base: %s, Title: %s, Body: %s, Assignees: %s, Milestone: %d, Labels: %d, Deadline: %s ]",
		f.Head, f.Base, f.Title, f.Body, f.Assignees, f.Milestone, f.Labels, f.Deadline)
}

// ChangePullRequestStatus изменение статуса PR
type ChangePullRequestStatus struct {
	Comment string `json:"comment"`
	Status  string `json:"status" binding:"SbtIn(reopen,close)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *ChangePullRequestStatus) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// ChangePullRequestReviewer структура запроса на добавление/удаление ревьюера в запрос на слияние
type ChangePullRequestReviewer struct {
	ReviewerId int64  `json:"reviewerId" binding:"Required"`
	IsTeam     bool   `json:"isTeam"`
	Action     string `json:"action" binding:"Required;SbtIn(attach,detach)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *ChangePullRequestReviewer) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

/*
MergePullRequest структура запроса на создание слияния пулл реквеста
Есть возможность пометить слияние как ручное слияние, для этого нужно в Do добавить SbtIn(manually-merged),
так же поле merge_commit_id нужно только в случае ручного слияния
*/
type MergePullRequest struct {
	Do                     *string `json:"do" binding:"Required;SbtIn(merge,rebase,rebase-merge,squash)"`
	MergeTitleField        *string `json:"merge_title_field"`
	MergeMessageField      *string `json:"merge_message_field"`
	MergeCommitID          *string `json:"merge_commit_id"` // нужно если тип слияния manually-merged
	HeadCommitID           *string `json:"head_commit_id"`
	ForceMerge             *bool   `json:"force_merge"`
	MergeWhenChecksSucceed *bool   `json:"merge_when_checks_succeed"`
	DeleteBranchAfterMerge *bool   `json:"delete_branch_after_merge"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *MergePullRequest) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
