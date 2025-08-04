package cron

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit_links_sender/unit_links_sender_db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/mtls"
	"code.gitea.io/gitea/modules/setting"
	setting_mtls "code.gitea.io/gitea/modules/setting/mtls"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
	"code.gitea.io/gitea/routers/private/task_tracker_sender"
)

func registerUnitLinksSender() {
	interval := setting.TaskTracker.UnitLinksSenderIntervalSeconds
	schedule := fmt.Sprintf("*/%d * * * * *", interval)

	cfg := &BaseConfig{Enabled: true, RunAtStart: true, Schedule: schedule}

	actionFunc := func(ctx context.Context, _ *user_model.User, config Config) error {
		dbEngine := db.GetEngine(ctx)
		unitLinkSenderDB := unit_links_sender_db.New(dbEngine)

		mtlsConfig := &tls.Config{} // initialize tls config with default values
		if setting_mtls.CheckMTLSConfigSecManEnabled(task_tracker_client.TaskTrackerClientName) {
			mtlsCerts := setting_mtls.GetMTLSCertsFromSecMan(task_tracker_client.TaskTrackerClientName, setting.NewGetterForSecMan())
			mtlsConfig = mtls.GenerateTlsConfigForMTLS(task_tracker_client.TaskTrackerClientName, mtlsCerts.Cert, mtlsCerts.CertKey, mtlsCerts.CaCerts)
		}
		taskTrackerClient := task_tracker_client.New(
			setting.TaskTracker.APIBaseURL,
			setting.TaskTracker.APIToken,
			&http.Client{Transport: &http.Transport{TLSClientConfig: mtlsConfig}},
		)
		sender := task_tracker_sender.NewTaskTrackerSender(
			taskTrackerClient, unitLinkSenderDB,
		)

		if err := sender.SendNewPrTasksToTaskTracker(ctx); err != nil {
			return err
		}

		return nil
	}

	RegisterTaskFatal("task_tracker", cfg, actionFunc)
}
