// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cron

import (
	"context"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	actions_service "code.gitea.io/gitea/services/actions"
)

func initActionsTasks() {
	if !setting.Actions.Enabled {
		return
	}
	registerStopZombieTasks()
	registerStopEndlessTasks()
	registerCancelAbandonedJobs()
}

func registerStopZombieTasks() {
	RegisterTaskFatal("stop_zombie_tasks", &BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   "0 */5 * * * *",
	}, func(ctx context.Context, _ *user_model.User, cfg Config) error {
		return actions_service.StopZombieTasks(ctx)
	})
}

func registerStopEndlessTasks() {
	RegisterTaskFatal("stop_endless_tasks", &BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   "0 */30 * * * *",
	}, func(ctx context.Context, _ *user_model.User, cfg Config) error {
		return actions_service.StopEndlessTasks(ctx)
	})
}

func registerCancelAbandonedJobs() {
	RegisterTaskFatal("cancel_abandoned_jobs", &BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   "0 0 */6 * * *",
	}, func(ctx context.Context, _ *user_model.User, cfg Config) error {
		return actions_service.CancelAbandonedJobs(ctx)
	})
}
