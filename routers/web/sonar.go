package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	sonar_model "code.gitea.io/gitea/models/sonar/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/webhook_sonar"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/webhook"
)

// WebhookSonarQube обработчик для webhook из sonarQube
func WebhookSonarQube(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := &webhook_sonar.WebHook{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("WebhookSonarQube io.ReadAll failed while read Body: %v", err)
		return
	}
	err = json.Unmarshal(body, res)
	if err != nil {
		log.Error("WebhookSonarQube json.Unmarshal while unmarshal body failed: %v", err)
		return
	}
	// добавляем проект из sonarQube в бд
	sonarProject, err := webhook.AddSonarProjectStatus(res)
	if err != nil {
		log.Error("WebhookSonarQube webhook.AddSonarProjectStatus while adding sonar_project_status failed: %v", err)
		return
	}
	// получаем настройки для sonarQube
	sonarSettings, err := repo_model.GetSonarSettingsByProjectKeyAndServerUrl(sonarProject.ProjectKey, sonarProject.SonarServer)
	if err != nil {
		log.Error("WebhookSonarQube webhook.AddSonarProjectStatus while getting sonar settings for repository failed: %v", err)
		return
	}
	if sonarSettings == nil {
		sonarMetrics := make([]sonar_model.ScSonarProjectMetrics, len(res.QualityGate.Conditions))
		for idx, condition := range res.QualityGate.Conditions {
			sonarMetricsEntity := sonar_model.ScSonarProjectMetrics{
				ID:                   uuid.NewString(),
				SonarProjectStatusID: sonarProject.ID,
				Key:                  condition.Metric,
				Value:                condition.Value,
				IsQualityGate:        true,
			}
			sonarMetrics[idx] = sonarMetricsEntity
		}
		errUpsertSonarMetrics := sonar_model.UpsertSonarMetrics(db.DefaultContext, sonarMetrics)
		if errUpsertSonarMetrics != nil {
			log.Error("WebhookSonarQube failed sonar_model.UpsertSonarMetrics while updating or adding sonar metrics failed: %v", errUpsertSonarMetrics)
			return
		}
		return
	}
	conditions := make([]webhook_sonar.SonarConditions, len(res.QualityGate.Conditions))
	for idx, condition := range res.QualityGate.Conditions {
		conditions[idx] = condition
	}
	// обрабатываем метрики из sonarQube
	err = webhook.ControllerWebHook(conditions, sonarSettings, sonarProject)
	if err != nil {
		log.Error("WebhookSonarQube webhook.ControllerWebHook while working with metrics for sonarqube failed: %v", err)
		return
	}
}

// GetMetricsForPagePullRequest получение информации о статусе pull request из sonaraqube
func GetMetricsForPagePullRequest(ctx *context.Context) {
	formPagePullRequest := web.GetForm(ctx).(*forms.GetStatusPullRequest)
	if ctx.Written() {
		return
	}
	resp, err := webhook.GetQualityGatesForProjectByPullRequest(formPagePullRequest.RepositoryID, formPagePullRequest.Base, formPagePullRequest.Branch, strconv.Itoa(formPagePullRequest.PullRequestID))
	if err != nil {
		if resp != nil && resp.Status == "401" {
			log.Error("GetMetricsForPagePullRequest webhook.GetQualityGatesForProject failed because user not authorized or have free version sonarqube")
			ctx.JSON(http.StatusUnauthorized, nil)
			return
		} else if resp != nil && resp.Status == "400" {
			log.Error(fmt.Sprintf("GetMetricsForPagePullRequest webhook.GetQualityGatesForProject failed because we didn't find pull request in sonarqube for such branches %s and %s", formPagePullRequest.Branch, formPagePullRequest.Base))
			ctx.JSON(http.StatusBadRequest, nil)
			return
		} else {
			log.Error("GetMetricsForPagePullRequest webhook.GetQualityGatesForProject while getting quality gates for repository_id %v, branch %s and base branch %s failed: %v", formPagePullRequest.RepositoryID, formPagePullRequest.Branch, formPagePullRequest.Base, err)
			ctx.JSON(http.StatusNotFound, nil)
			return
		}
	}
	ctx.JSON(http.StatusOK, resp)
}
