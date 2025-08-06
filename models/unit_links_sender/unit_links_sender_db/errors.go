package unit_links_sender_db

import "fmt"

type TaskAlreadyLockedError struct {
	TaskID int64
}

func NewTaskAlreadyLockedError(taskID int64) *TaskAlreadyLockedError {
	return &TaskAlreadyLockedError{TaskID: taskID}
}

func (e *TaskAlreadyLockedError) Error() string {
	return fmt.Sprintf("task '%d' already locked", e.TaskID)
}
