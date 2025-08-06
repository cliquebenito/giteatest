// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cron

import (
	"code.gitea.io/gitea/models/db"
	system_model "code.gitea.io/gitea/models/system"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/translation"
	"context"
	"fmt"
	"github.com/google/uuid"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	lock     = sync.Mutex{}
	started  = false
	tasks    = []*Task{}
	tasksMap = map[string]*Task{}
)

// Task represents a Cron task
type Task struct {
	lock        sync.Mutex
	Name        string
	config      Config
	fun         func(context.Context, *user_model.User, Config) error
	Status      string
	LastMessage string
	LastDoer    string
	ExecTimes   int64
}

// DoRunAtStart returns if this task should run at the start
func (t *Task) DoRunAtStart() bool {
	return t.config.DoRunAtStart()
}

// IsEnabled returns if this task is enabled as cron task
func (t *Task) IsEnabled() bool {
	return t.config.IsEnabled()
}

// GetConfig will return a copy of the task's config
func (t *Task) GetConfig() Config {
	if reflect.TypeOf(t.config).Kind() == reflect.Ptr {
		// Pointer:
		return reflect.New(reflect.ValueOf(t.config).Elem().Type()).Interface().(Config)
	}
	// Not pointer:
	return reflect.New(reflect.TypeOf(t.config)).Elem().Interface().(Config)
}

// Run will run the task incrementing the cron counter with no user defined
func (t *Task) Run() {
	t.RunWithUser(&user_model.User{
		ID:        -1,
		Name:      "(Cron)",
		LowerName: "(cron)",
	}, t.config, audit.EmptyRequiredField)
}

// RunWithUser will run the task incrementing the cron counter at the time with User
func (t *Task) RunWithUser(doer *user_model.User, config Config, remoteAddress string) {
	log.Debug("Run task: %s", t.Name)

	if config == nil {
		config = t.config
	}

	auditUserName := audit.EmptyRequiredField
	auditUserId := audit.EmptyRequiredField
	auditParams := map[string]string{
		"task_name":    t.Name,
		"enabled":      strconv.FormatBool(config.IsEnabled()),
		"run_at_start": strconv.FormatBool(config.DoRunAtStart()),
		"schedule":     config.GetSchedule(),
	}

	if doer.ID != -1 {
		auditUserId = strconv.FormatInt(doer.ID, 10)
		auditUserName = doer.Name
	}

	processUUID := uuid.NewString()
	inProgress := true
	lockErr := system_model.LockCronJob(t.Name, time.Now(), processUUID)
	if lockErr != nil {
		if system_model.IsErrLockCronJob(lockErr) {
			log.Debug(lockErr.Error())
			auditParams["error"] = "Cron task is already locked"
		} else {
			log.Error("Unable to run cron task %s Error %v", t.Name, lockErr)
			auditParams["error"] = "Unable to lock cron task to run"
		}
		audit.CreateAndSendEvent(audit.CronTaskLock, auditUserName, auditUserId, audit.StatusFailure, remoteAddress, auditParams)
		audit.CreateAndSendEvent(audit.CronTaskRun, auditUserName, auditUserId, audit.StatusFailure, remoteAddress, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.CronTaskLock, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)
	if !TaskStatusTable.StartIfNotRunning(t.Name) {
		auditParams["error"] = "Unable to run cron task"
		audit.CreateAndSendEvent(audit.CronTaskRun, auditUserName, auditUserId, audit.StatusFailure, remoteAddress, auditParams)
		return
	}
	t.lock.Lock()
	t.ExecTimes++
	t.lock.Unlock()
	audit.CreateAndSendEvent(audit.CronTaskRun, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)
	defer func() {
		TaskStatusTable.Stop(t.Name)
		if err := recover(); err != nil {
			// Recover a panic within the
			combinedErr := fmt.Errorf("%s\n%s", err, log.Stack(2))
			log.Error("PANIC whilst running task: %s Value: %v", t.Name, combinedErr)
		}
		if !system_model.IsErrLockCronJob(lockErr) {
			var nextTime time.Time
			for _, entr := range c.Entries() {
				if entr.Description == t.Name {
					nextTime = entr.Next
				}
			}
			count, err := system_model.UnlockCronJob(t.Name, nextTime, t.Status, processUUID)
			if err != nil {
				log.Error("Unable to unlock cron task %s Error %v", t.Name, err)
				auditParams["error"] = "Unable to unlock cron task"
				audit.CreateAndSendEvent(audit.CronTaskUnlock, auditUserName, auditUserId, audit.StatusFailure, remoteAddress, auditParams)
				return
			}
			if count > 0 {
				log.Debug("Unlock task: %s", t.Name)
				audit.CreateAndSendEvent(audit.CronTaskUnlock, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)
			} else {
				log.Debug("Task: %s stopped", t.Name)
			}
		}
	}()
	graceful.GetManager().RunWithShutdownContext(func(baseCtx context.Context) {
		pm := process.GetManager()
		doerName := ""
		if doer != nil && doer.ID != -1 {
			doerName = doer.Name
		}

		ctx, cancel, finished := pm.AddContext(baseCtx, config.FormatMessage(translation.NewLocale("en-US"), t.Name, "process", doerName))
		defer finished()

		go func() {
			for {
				time.Sleep(time.Duration(setting.Cron.CheckDelayUnlocked) * time.Second)
				cronJob, err := system_model.GetCronJobLock(t.Name)
				if err != nil || cronJob.ProcessUUID != processUUID && inProgress {
					log.Debug("Cron job: %s cancelled", cronJob.Name)
					audit.CreateAndSendEvent(audit.CronTaskCancel, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)
					cancel()
					return
				}
			}
		}()

		if err := t.fun(ctx, doer, config); err != nil {
			var message string
			var status string
			if db.IsErrCancelled(err) {
				status = "cancelled"
				message = err.(db.ErrCancelled).Message
				auditParams["error"] = message
				audit.CreateAndSendEvent(audit.CronTaskCancel, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)
			} else {
				status = "error"
				message = err.Error()
				auditParams["error"] = message
				audit.CreateAndSendEvent(audit.CronTaskFinished, auditUserName, auditUserId, audit.StatusFailure, remoteAddress, auditParams)
			}

			t.lock.Lock()
			t.LastMessage = message
			t.Status = status
			t.LastDoer = doerName
			inProgress = false
			t.lock.Unlock()

			if err := system_model.CreateNotice(ctx, system_model.NoticeTask, config.FormatMessage(translation.NewLocale("en-US"), t.Name, "cancelled", doerName, message)); err != nil {
				log.Error("CreateNotice: %v", err)
			}
			return
		}

		t.lock.Lock()
		t.Status = "finished"
		t.LastMessage = ""
		t.LastDoer = doerName
		inProgress = false
		t.lock.Unlock()

		audit.CreateAndSendEvent(audit.CronTaskFinished, auditUserName, auditUserId, audit.StatusSuccess, remoteAddress, auditParams)

		if config.DoNoticeOnSuccess() {
			if err := system_model.CreateNotice(ctx, system_model.NoticeTask, config.FormatMessage(translation.NewLocale("en-US"), t.Name, "finished", doerName)); err != nil {
				log.Error("CreateNotice: %v", err)
			}
		}
	})
}

// GetTask gets the named task
func GetTask(name string) *Task {
	lock.Lock()
	defer lock.Unlock()
	log.Info("Getting %s in %v", name, tasksMap[name])

	return tasksMap[name]
}

// RegisterTask allows a task to be registered with the cron service
func RegisterTask(name string, config Config, fun func(context.Context, *user_model.User, Config) error) error {
	log.Debug("Registering task: %s", name)

	auditParams := map[string]string{
		"task_name":    name,
		"enabled":      strconv.FormatBool(config.IsEnabled()),
		"run_at_start": strconv.FormatBool(config.DoRunAtStart()),
		"schedule":     config.GetSchedule(),
	}

	i18nKey := "admin.dashboard." + name
	if value := translation.NewLocale("en-US").Tr(i18nKey); value == i18nKey {
		auditParams["error"] = "Translation is missing for task"
		audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("translation is missing for task %q, please add translation for %q", name, i18nKey)
	}

	_, err := setting.GetCronSettings(name, config)
	if err != nil {
		log.Error("Unable to register cron task with name: %s Error: %v", name, err)
		auditParams["error"] = "Unable to register cron task"
		audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}

	task := &Task{
		Name:   name,
		config: config,
		fun:    fun,
	}
	lock.Lock()
	locked := true
	defer func() {
		if locked {
			lock.Unlock()
		}
	}()
	if _, has := tasksMap[task.Name]; has {
		log.Error("A task with this name: %s has already been registered", name)
		auditParams["error"] = "Task with this name has already been registered"
		audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("duplicate task with name: %s", task.Name)
	}

	if config.IsEnabled() {
		// We cannot use the entry return as there is no way to lock it
		if _, err = c.AddJob(name, config.GetSchedule(), task); err != nil {
			log.Error("Unable to register cron task with name: %s Error: %v", name, err)
			auditParams["error"] = "Unable to register cron task"
			audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return err
		}
	}

	tasks = append(tasks, task)
	tasksMap[task.Name] = task
	var nextTime time.Time
	if len(c.Entries()) > 0 {
		for _, entry := range c.Entries() {
			if entry.Description == task.Name {
				nextTime = entry.Next
			}
		}
	}
	err = system_model.RegisterCronJob(task.Name, nextTime)
	if err != nil {
		auditParams["error"] = "Unable to register cron task"
		audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}
	if started && config.IsEnabled() && config.DoRunAtStart() {
		lock.Unlock()
		locked = false
		task.Run()
	}

	audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return nil
}

// RegisterTaskFatal will register a task but if there is an error log.Fatal
func RegisterTaskFatal(name string, config Config, fun func(context.Context, *user_model.User, Config) error) {
	auditParams := map[string]string{
		"task_name":    name,
		"enabled":      strconv.FormatBool(config.IsEnabled()),
		"run_at_start": strconv.FormatBool(config.DoRunAtStart()),
		"schedule":     config.GetSchedule(),
	}
	if strings.Contains(config.GetSchedule(), "@every") {
		auditParams["error"] = "@every don't allowed in cron"
		audit.CreateAndSendEvent(audit.CronTaskRegistered, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Fatal("Unable to register cron task %s Error: %v", name, fmt.Errorf("you can't use @every in cron"))
	}
	if err := RegisterTask(name, config, fun); err != nil {
		log.Fatal("Unable to register cron task %s Error: %v", name, err)
	}
}
