package setting

import (
	"net/http"
	"time"

	"code.gitea.io/gitea/models/repo"
	repo2 "code.gitea.io/gitea/models/sonar/repo"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/webhook_sonar"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/webhook"
)

const (
	tplSonarSettings base.TplName = "repo/settings/sonar"
)

// GetSonarSettings Получение настроек Sonar репозитория
func GetSonarSettings(ctx *context.Context) {
	settings, err := repo.GetSonarSettings(ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while try get sonar settings of repoID: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}

	if settings != nil {
		ctx.Data["Sonar"] = settings
	} else { //кидать в этом случае ошибку не нужно, просто отдадим пустые значения
		ctx.Data["Sonar"] = repo.ScSonarSettings{Token: "", ProjectKey: ""}
	}

	ctx.HTML(http.StatusOK, tplSonarSettings)
}

// PostSonarSettings добавление или изменение настроек Sonar для репозитория (если настроек не было то они добавляются, если были то обновляются новыми значениями)
func PostSonarSettings(ctx *context.Context) {
	settings := web.GetForm(ctx).(*forms.AddSonarSettings)

	if ctx.Written() {
		return
	}

	err := repo.InsertOrUpdateSonarSettings(ctx.Repo.Repository.ID, settings.URL, settings.Token, settings.ProjectKey)
	if err != nil {
		log.Error("Error has occurred while try insert or update sonar settings of repo id: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	// получаем натсройки для sonarQube
	sonarSettings, err := repo.GetSonarSettings(ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while try get sonar settings of repoID: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	// получаем список всех проектов для конкретного репозитория
	sonarProjects, err := repo2.GetSonarProjectStatusBySettings(sonarSettings, "")
	if err != nil {
		log.Error("Error has occurred while getting sonar project status for project_key %s: %v", sonarSettings.ProjectKey, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	if len(sonarProjects) == 0 {
		log.Info("PostSonarSettings sonarProjects is empty")
		ctx.JSON(http.StatusOK, nil)
		return
	}
	// дополняем метрики информцией с помощью токена для sonarQube
	for _, sonarProject := range sonarProjects {
		sonarProjectMetrics, errGetSonarProjectMetrics := repo2.GetSonarProjectMetrics(sonarProject.ID)
		if errGetSonarProjectMetrics != nil {
			log.Error("Error has occurred while getting sonar project metrics for project_key %s: %v", sonarSettings.ProjectKey, err)
			ctx.Error(http.StatusInternalServerError)
			return
		}
		projectStatusIDsWithSonarMetrics := make(map[string][]webhook_sonar.SonarConditions)
		for _, sonarMetric := range sonarProjectMetrics {
			if sonarMetric.SonarProjectStatusID == sonarProject.ID {
				conditionMetricsSonar := webhook_sonar.SonarConditions{
					Metric: sonarMetric.Key,
					Value:  sonarMetric.Value,
				}
				projectStatusIDsWithSonarMetrics[sonarMetric.SonarProjectStatusID] = append(projectStatusIDsWithSonarMetrics[sonarMetric.SonarProjectStatusID], conditionMetricsSonar)
			}
		}
		errControlWebHookSonar := webhook.ControllerWebHook(projectStatusIDsWithSonarMetrics[sonarProject.ID], sonarSettings, &sonarProject)
		if errControlWebHookSonar != nil {
			log.Error("Error has occurred while adding or updating sonar metrics for project_key %s: %v", sonarSettings.ProjectKey, err)
			ctx.Error(http.StatusInternalServerError)
			return
		}
	}
	ctx.Status(http.StatusOK)
}

// DeleteSonarSettings Удаление настроек Sonar репозитория
func DeleteSonarSettings(ctx *context.Context) {
	err := repo.DeleteSonarSettings(ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while try delete sonar settings of repo id: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetMetricsFromSonarQube получаем метрики sonarQube для репозитория из бд
func GetMetricsFromSonarQube(ctx *context.Context) {
	formForGetMetrics := web.GetForm(ctx).(*forms.GetMetricsSonarQube)
	if ctx.Written() {
		return
	}
	// получаем натсройки для sonarQube по repository_id
	sonarSettings, err := repo.GetSonarSettings(formForGetMetrics.RepositoryID)
	if err != nil {
		log.Error("GetMetricsFromSonarQube repo.GetSonarSettings failed while getting sonar settings for repository_id %v: %v", formForGetMetrics.RepositoryID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	// получаем список всех проектов для конкретного репозитория и ветке
	sonarProjects, err := repo2.GetSonarProjectStatusBySettings(sonarSettings, formForGetMetrics.Branch)
	if err != nil {
		log.Error("GetMetricsFromSonarQube sonar_model.GetSonarProjectStatusBySettings failed while getting sonar project status for branch %s: %v", formForGetMetrics.Branch, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	if len(sonarProjects) == 0 {
		log.Info("GetMetricsFromSonarQube sonarProjects is empty")
		ctx.JSON(http.StatusOK, nil)
		return
	}
	// получаем метрики и информацию о них для конкретного проекта
	sonarProjectMetrics, err := repo2.GetSonarProjectMetrics(sonarProjects[0].ID)
	if err != nil {
		log.Error("GetMetricsFromSonarQube sonar_model.GetSonarProjectMetrics failed while getting sonar project metrics for sonar_project_id %v: %v", sonarProjects[0].ID, err)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	if len(sonarProjectMetrics) == 0 {
		log.Info("GetMetricsFromSonarQube sonarProjectMetrics is empty")
		ctx.JSON(http.StatusOK, nil)
		return
	}
	projectMetricsGroupByDomain := make(map[string][]repo2.ScSonarProjectMetrics)
	for _, sonarProjectMetric := range sonarProjectMetrics {
		projectMetricsGroupByDomain[sonarProjectMetric.Domain] = append(projectMetricsGroupByDomain[sonarProjectMetric.Domain], sonarProjectMetric)
	}
	responseMetrics := make([]webhook_sonar.ResponseMetrics, 0, len(projectMetricsGroupByDomain))
	for _, projectMetrics := range projectMetricsGroupByDomain {
		metricInfo := webhook_sonar.ResponseMetrics{}
		for _, metric := range projectMetrics {
			switch metric.IsQualityGate {
			case false:
				metricInfo.AuxMetricKey = metric.Key
				metricInfo.AuxMetricName = metric.Name
				metricInfo.AuxMetricType = metric.Type
				metricInfo.AuxMetricValue = metric.Value
			default:
				metricInfo.Key = metric.Key
				metricInfo.Name = metric.Name
				metricInfo.Value = metric.Value
				metricInfo.Type = metric.Type
				metricInfo.Domain = metric.Domain
			}
		}
		responseMetrics = append(responseMetrics, metricInfo)
	}
	ctx.JSON(http.StatusOK, &webhook_sonar.SonarResponseForRepository{
		SonarQubeStatus: sonarProjects[0].Status,
		SonarUrl:        sonarProjects[0].SonarServer,
		SonarProjectKey: sonarProjects[0].ProjectKey,
		Metrics:         responseMetrics,
		AnalysedAt:      sonarProjects[0].AnalysedAt.Format(time.RFC3339),
	})
}
