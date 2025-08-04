package cron

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_counter_task/code_hub_counter_task_db"
	"code.gitea.io/gitea/models/code_hub_unique_usages/code_hub_unique_usages_db"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/internal_metric_counter/internal_metric_counter_db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/private/code_hub_counter"
)

func registerCodeHubCounterTasksProcessor() {
	// сделать настройку
	interval := setting.CodeHub.CodeHubUsagesProcessorIntervalSeconds
	schedule := fmt.Sprintf("*/%d * * * * *", interval)

	cfg := &BaseConfig{Enabled: true, RunAtStart: true, Schedule: schedule}

	actionFunc := func(ctx context.Context, _ *user_model.User, config Config) error {
		dbEngine := db.GetEngine(ctx)

		codeHubTasksDB := code_hub_counter_task_db.New(dbEngine)
		codeHubUniqueUsagesDB := code_hub_unique_usages_db.New(dbEngine)
		codeHubCounterDB := internal_metric_counter_db.New(dbEngine)

		counter := code_hub_counter.NewCodeHubCounter(codeHubTasksDB, codeHubUniqueUsagesDB, codeHubCounterDB)

		if err := counter.ProcessNewUsageTasks(ctx); err != nil {
			return fmt.Errorf("error has occurred while proccessing new tasks: %w", err)
		}

		return nil
	}

	RegisterTaskFatal("code_hub_tasks_processor", cfg, actionFunc)
}

func registerCodeHubCounterStatsProcessor() {
	interval := setting.CodeHub.CodeHubCounterProcessorIntervalSeconds
	schedule := fmt.Sprintf("*/%d * * * * *", interval)

	cfg := &BaseConfig{Enabled: true, RunAtStart: true, Schedule: schedule}

	actionFunc := func(ctx context.Context, _ *user_model.User, config Config) error {
		dbEngine := db.GetEngine(ctx)

		codeHubTasksDB := code_hub_counter_task_db.New(dbEngine)
		codeHubUniqueUsagesDB := code_hub_unique_usages_db.New(dbEngine)
		codeHubCounterDB := internal_metric_counter_db.New(dbEngine)

		counter := code_hub_counter.NewCodeHubCounter(codeHubTasksDB, codeHubUniqueUsagesDB, codeHubCounterDB)

		if err := counter.CalculateRepoCounters(ctx); err != nil {
			return fmt.Errorf("error has occurred while calculating repo counters: %w", err)
		}

		return nil
	}
	RegisterTaskFatal("code_hub_stats_processor", cfg, actionFunc)
}
