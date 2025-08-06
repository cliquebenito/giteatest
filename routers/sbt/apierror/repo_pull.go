package apiError

import (
	"code.gitea.io/gitea/modules/setting"
	"fmt"
)

//400 ошибки при работе с тегами

// PullRequestAlreadyExist ошибка в случае если запрос на слияние заданных веток уже существует
func PullRequestAlreadyExist(id int64) ApiError {
	return ApiError{Code: 4600, Message: fmt.Sprintf("%v", id)}
}

// PullRequestNotFound Ошибка в случае если пулл-реквест не найден по номеру
func PullRequestNotFound(number int64) ApiError {
	return ApiError{Code: 4601, Message: fmt.Sprintf("Pull request with number %d not found", number)}
}

// PullRequestAlreadyClosed Ошибка в случае если запрос на слияние уже закрыт
func PullRequestAlreadyClosed(number int64) ApiError {
	return ApiError{Code: 4602, Message: fmt.Sprintf("Pull request with number %d already closed", number)}
}

// PullRequestAlreadyMerged пулл реквест уже слит
func PullRequestAlreadyMerged() ApiError {
	return ApiError{Code: 4603, Message: "Pull request already merged"}
}

// PullRequestWorkInProgress пулл реквест помечен в процессе разработки
func PullRequestWorkInProgress() ApiError {
	return ApiError{Code: 4604, Message: "Pull request work in progress"}
}

// PullRequestNotMergableState пулл реквест не в состоянии мерджа
func PullRequestNotMergableState() ApiError {
	return ApiError{Code: 4605, Message: "Pull request not in mergable state"}
}

// InvalidMergeStyle ошибка в слчае если неверный стиль слияния
func InvalidMergeStyle() ApiError {
	return ApiError{Code: 4606, Message: "Invalid merge style"}
}

// MergeConflict мердж конфликт при попытке мерджа
func MergeConflict() ApiError {
	return ApiError{Code: 4607, Message: "Merge conflict"}
}

// RebaseConflict ребейз конфликт при попытке мерджа
func RebaseConflict() ApiError {
	return ApiError{Code: 4608, Message: "Rebase conflict"}
}

// UnrelatedHistories ошибка при попытке слить вместе несвязанные истории
func UnrelatedHistories() ApiError {
	return ApiError{Code: 4609, Message: "Unrelated histories"}
}

// SHADoesNotMatch ошибка в случае если SHA не совпадают
func SHADoesNotMatch() ApiError {
	return ApiError{Code: 4610, Message: "SHA does not match"}
}

// MergePushRejected ошибка в случае если пуш коммита был отклонен
func MergePushRejected() ApiError {
	return ApiError{Code: 4611, Message: "Merge push was rejected"}
}

// NotValidPullRequestReviewer ошибка в случае если ревьюер не валиден для данного запроса на слияние
func NotValidPullRequestReviewer() ApiError {
	return ApiError{Code: 4612, Message: "Reviewer is not valid"}
}

// CommentNotFound комментарий не найден
func CommentNotFound() ApiError {
	return ApiError{Code: 4613, Message: "Comment not found"}
}

// CommentHasNotContent комментарий не имеет контента (комментарием например является удаление ветки)
func CommentHasNotContent() ApiError {
	return ApiError{Code: 4614, Message: "Comment has no content"}
}

// PullRequestForCommentNotFound в случае если пулл-реквест привязанный к комментарию не найден
func PullRequestForCommentNotFound(number int64) ApiError {
	return ApiError{Code: 4615, Message: fmt.Sprintf("Pull request for comment id: %d not found", number)}
}

// ReactionActionUnknown нет такого типа действия для реакции
func ReactionActionUnknown() ApiError {
	return ApiError{Code: 4616, Message: "Reaction action type is unknown"}
}

// ReactionNotFound нет такой реакции в конфиге
func ReactionNotFound() ApiError {
	return ApiError{Code: 4617, Message: "Reaction not found"}
}

// CommentsAlreadyLocked комментирование уже заблокировано для внешних участников
func CommentsAlreadyLocked() ApiError {
	return ApiError{Code: 4618, Message: "Comments are already locked"}
}

// CommentsNotLocked комментирование не заблокировано для внешних участников
func CommentsNotLocked() ApiError {
	return ApiError{Code: 4619, Message: "Comments are not locked"}
}

// InvalidCommentLockReason не валидная причина блокировки комментов
func InvalidCommentLockReason() ApiError {
	return ApiError{Code: 4620, Message: fmt.Sprintf("Invalid lock reason. Valid reasons are: %v", setting.Repository.Issue.LockReasons)}
}

// CommentHistoryDetailNotFound не найден пункт истории комментария
func CommentHistoryDetailNotFound() ApiError {
	return ApiError{Code: 4621, Message: "Comment history detail not found"}
}
