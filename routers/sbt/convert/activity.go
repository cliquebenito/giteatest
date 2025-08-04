package convert

import (
	"code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/routers/sbt/response"
	"strings"
	"time"
)

// ToHeatMapData конвертирует таблицу активностей пользователей в ответ в виде мапы (дата - количество активностей)
func ToHeatMapData(data []*activities.UserHeatmapData) map[time.Time]int64 {
	responseData := make(map[time.Time]int64)

	for i := range data {
		t := truncateToDay(data[i].Timestamp.AsTime())
		responseData[t] = responseData[t] + data[i].Contributions
	}

	return responseData
}

// truncateToDay усекает время до дней
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ToAction конвертирует модель активности в ДТО
func ToAction(action *activities.Action) *response.Action {

	result := &response.Action{
		ID:        action.ID,
		OpType:    action.OpType.String(),
		Comment:   action.Comment,
		IsDeleted: action.IsDeleted,
		RefName:   action.RefName,
		IsPrivate: action.IsPrivate,
		Content:   action.Content,
		Created:   action.CreatedUnix.AsTime(),
	}

	if action.Repo != nil {
		result.RepoID = action.Repo.ID
		result.RepoName = action.Repo.Name
		result.RepoOwnerName = action.Repo.OwnerName
	}

	if action.ActUser != nil {
		result.UserID = action.ActUser.ID
		result.UserName = action.ActUser.Name
	}

	switch action.OpType {
	case activities.ActionCommitRepo, activities.ActionMirrorSyncPush:
		if action.OpType == activities.ActionCommitRepo && len(action.GetContent()) == 0 {
			//костыль, так как нет такого события изначально
			result.OpType = "create_branch"
			result.AdditionalInfo = action.GetBranch()
		} else {
			//подчищаем "старый" URL
			push := repository.NewPushCommits()
			err := json.Unmarshal([]byte(action.GetContent()), push)
			if err != nil {
				log.Error("Action id: %d, broken content has found: %s, error: %v  ", action.ID, action.GetContent(), err)
			} else {
				parts := strings.Split(push.CompareURL, "/")
				if len(parts) > 0 {
					//добавляем сравниваемые SHA
					result.AdditionalInfo = parts[len(parts)-1]
					//убираем "старый" URL
					result.Content = strings.ReplaceAll(result.Content, push.CompareURL, "")
				}
			}
		}
	case activities.ActionCreateIssue, activities.ActionCreatePullRequest:
		result.AdditionalInfo = strings.Join(action.GetIssueInfos(), "|")
	case activities.ActionCommentIssue, activities.ActionApprovePullRequest, activities.ActionRejectPullRequest, activities.ActionCommentPull:
		result.AdditionalInfo = action.GetIssueTitle() + "|" + action.GetIssueInfos()[1]
	case activities.ActionMergePullRequest, activities.ActionAutoMergePullRequest:
		result.AdditionalInfo = action.GetIssueInfos()[1]
	case activities.ActionCloseIssue, activities.ActionReopenIssue, activities.ActionClosePullRequest, activities.ActionReopenPullRequest:
		result.AdditionalInfo = action.GetIssueTitle()
	case activities.ActionPullReviewDismissed:
		result.AdditionalInfo = action.GetIssueInfos()[2]
	}

	return result
}
