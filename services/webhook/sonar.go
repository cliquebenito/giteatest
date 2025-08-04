package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/sonar/repo"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/webhook_sonar"
)

const (
	// для получения measures из sonarqube
	urlApiMeasureSearch = "/api/measures/search"
	// для получения информации для pull request
	urlApiListPullRequest = "/api/project_pull_requests/list"
	// для получения metrics из sonarqube
	urlApiMetricsSearch = "/api/metrics/search"
)

// AddSonarProjectStatus добавление или обновление информации о проекте из сонара
func AddSonarProjectStatus(webhookInfo *webhook_sonar.WebHook) (*repo.ScSonarProjectStatus, error) {
	timeAnalyzedAt, err := time.Parse("2006-01-02T15:04:05+0300", webhookInfo.AnalysedAt)
	if err != nil {
		return nil, fmt.Errorf("time parse: %w", err)
	}
	var analyzedAt timeutil.TimeStamp
	sonarProjectStatusEntity := &repo.ScSonarProjectStatus{
		ID:          uuid.NewString(),
		SonarServer: webhookInfo.ServerUrl,
		ProjectKey:  webhookInfo.Project.Key,
		Branch:      webhookInfo.Branch.Name,
		AnalysedAt:  analyzedAt.Add(int64(timeAnalyzedAt.Second())),
		Status:      webhookInfo.QualityGate.Status,
	}
	sonarProjectStatus, err := repo.UpsertSonarProjectStatus(db.DefaultContext, sonarProjectStatusEntity)
	if err != nil {
		return nil, fmt.Errorf("upsert sonar project status: %w", err)
	}
	return sonarProjectStatus, nil
}

// ControllerWebHook обработчик информации из webhook
func ControllerWebHook(conditions []webhook_sonar.SonarConditions, sonarSettings *repo_model.ScSonarSettings, sonarProject *repo.ScSonarProjectStatus) error {
	mapUniqueCondition := make(map[string]webhook_sonar.SonarConditions)
	for _, cond := range conditions {
		mapUniqueCondition[cond.Metric] = cond
	}
	_, err := GetAllMetrics(mapUniqueCondition, sonarSettings, sonarProject)
	if err != nil {
		return err
	}
	return nil
}

// GetAllMetrics получаем все информацию о метриках из webhook и добавляем информацию о них в бд
func GetAllMetrics(mapUniqueCondition map[string]webhook_sonar.SonarConditions, sonarSettingsForRepository *repo_model.ScSonarSettings, sonarProject *repo.ScSonarProjectStatus) ([]webhook_sonar.SonarMetrics, error) {
	// получаем все возможные метрики
	reqUrl := sonarSettingsForRepository.URL + urlApiMetricsSearch
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.SetBasicAuth(sonarSettingsForRepository.Token, "")
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var res webhook_sonar.MetricsSonar
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respBody, &res)
	if err != nil {
		return nil, err
	}

	// формируем массив с информацией о метриках, которые есть в webhook
	sonarMetrics := make([]webhook_sonar.SonarMetrics, 0, len(mapUniqueCondition))
	upsertSonarSetting := make([]repo.ScSonarProjectMetrics, 0)
	for _, metricSonar := range res.Metrics {
		if val, ok := mapUniqueCondition[metricSonar.Key]; ok {
			entitySonarProject := repo.ScSonarProjectMetrics{
				ID:                   uuid.NewString(),
				SonarProjectStatusID: sonarProject.ID,
				Key:                  metricSonar.Key,
				Name:                 metricSonar.Name,
				Type:                 metricSonar.Type,
				Domain:               metricSonar.Domain,
				Value:                val.Value,
				IsQualityGate:        true,
			}
			sonarMetrics = append(sonarMetrics, metricSonar)
			upsertSonarSetting = append(upsertSonarSetting, entitySonarProject)
		}
	}

	// получаем инфомация о дополнительных метоирках, если такие есть
	sonarMeasures, err := SonarMeasuresGet(sonarMetrics, sonarSettingsForRepository, sonarSettingsForRepository.ProjectKey)
	if err != nil {
		return nil, err
	}
	uniqueMeasure := make(map[string]webhook_sonar.SonarMeasures)
	for _, sonarMeasure := range sonarMeasures.Measures {
		uniqueMeasure[sonarMeasure.Metric] = sonarMeasure
	}
	for _, sonarMeasure := range res.Metrics {
		if val, ok := uniqueMeasure[sonarMeasure.Key]; ok {
			valueMeasure := val.Value
			if val.Period.Index != 0 {
				valueMeasure = val.Period.Value
			}
			entitySonarMeasure := repo.ScSonarProjectMetrics{
				ID:                   uuid.NewString(),
				SonarProjectStatusID: sonarProject.ID,
				Key:                  sonarMeasure.Key,
				Name:                 sonarMeasure.Name,
				Type:                 sonarMeasure.Type,
				Domain:               sonarMeasure.Domain,
				Value:                valueMeasure,
				IsQualityGate:        false,
			}
			upsertSonarSetting = append(upsertSonarSetting, entitySonarMeasure)
		}
	}
	// добавляем или обновляем информацию о метриках
	err = repo.UpsertSonarMetrics(db.DefaultContext, upsertSonarSetting)
	if err != nil {
		return nil, err
	}
	return sonarMetrics, nil
}

// SonarMeasuresGet получение информации о дополнительных метриках
func SonarMeasuresGet(sonarMetrics []webhook_sonar.SonarMetrics, sonarSettingsForRepository *repo_model.ScSonarSettings, projectKey string) (*webhook_sonar.MeasureSonar, error) {
	// доменя для получения дополнительных метрик
	qualityGate := map[string]string{
		"Reliability":     "bugs",
		"Maintainability": "code_smells",
		"Security":        "vulnerabilities",
		"SecurityReview":  "security_hotspots",
		"Coverage":        "new_lines_to_cover",
		"Duplications":    "new_duplicated_lines",
	}
	metricKeysUnique := make(map[string]struct{})
	for _, sonarMetric := range sonarMetrics {
		if val, ok := qualityGate[sonarMetric.Domain]; ok {
			if _, okGetMetric := metricKeysUnique[val]; !okGetMetric {
				metricKeysUnique[val] = struct{}{}
			}
		}
	}
	metricKeys := make([]string, 0, len(metricKeysUnique))
	for metricKey := range metricKeysUnique {
		metricKeys = append(metricKeys, metricKey)
	}
	reqUrl := sonarSettingsForRepository.URL + urlApiMeasureSearch
	req, err := http.NewRequest("GET", reqUrl, nil)
	queryReqValue := req.URL.Query()
	queryReqValue.Add("metricKeys", strings.Join(metricKeys, ","))
	queryReqValue.Add("projectKeys", projectKey)
	req.URL.RawQuery = queryReqValue.Encode()
	req.SetBasicAuth(sonarSettingsForRepository.Token, "")
	if err != nil {
		log.Error("SonarMeasuresGet failed while create request: %v", err)
		return nil, err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("SonarMeasuresGet failed while send request ulr:%s : %v", reqUrl, err)
		return nil, err
	}

	var listSonarMeasures webhook_sonar.MeasureSonar
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("SonarMeasuresGet failed while read response body: %v", err)
		return nil, err
	}
	err = json.Unmarshal(respBody, &listSonarMeasures)
	if err != nil {
		log.Error("SonarMeasuresGet failed while unmarshal response body: %v", err)
		return nil, err
	}
	return &listSonarMeasures, nil
}

// GetQualityGatesForProjectByPullRequest получаем информацию о quality gates, которые пришли из webhook
func GetQualityGatesForProjectByPullRequest(repositoryID int64, baseBranch, branch, pullRequestID string) (*webhook_sonar.ResponseForPagePullRequest, error) {
	// получаем натсройки для sonarQube по repository_id
	sonarSettings, err := repo_model.GetSonarSettings(repositoryID)
	if err != nil {
		log.Error("GetQualityGatesForProjectByPullRequest repo_model.GetSonarSettings failed while getting sonar settings for repository_id %v: %v", repositoryID, err)
		return nil, err
	}
	if sonarSettings == nil {
		log.Error("GetQualityGatesForProjectByPullRequest sonarSettings is nil for repositoryID: %v", repositoryID)
		return &webhook_sonar.ResponseForPagePullRequest{}, fmt.Errorf("GetQualityGatesForProjectByPullRequest sonarSettings is empty")
	}
	reqUrl := sonarSettings.URL + urlApiListPullRequest
	req, err := http.NewRequest("GET", reqUrl, nil)
	queryReqValue := req.URL.Query()
	queryReqValue.Add("project", sonarSettings.ProjectKey)
	req.URL.RawQuery = queryReqValue.Encode()
	req.SetBasicAuth(sonarSettings.Token, "")
	if err != nil {
		log.Error("GetQualityGatesForProjectByPullRequest http.NewRequest failed while create request: %v", err)
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("GetQualityGatesForProjectByPullRequest failed while send request ulr:%s : %v", reqUrl, err)
		return nil, err
	}
	if response.StatusCode == http.StatusUnauthorized {
		return &webhook_sonar.ResponseForPagePullRequest{Status: "401"}, fmt.Errorf("response.StatusCode == 401")
	}
	var ResponsePullRequestInformation webhook_sonar.PullRequestsInformation
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("GetQualityGatesForProjectByPullRequest failed while read response body: %v", err)
		return nil, err
	}
	err = json.Unmarshal(respBody, &ResponsePullRequestInformation)
	if err != nil {
		log.Error("GetQualityGatesForProjectByPullRequest failed while unmarshal response body: %v", err)
		return nil, err
	}
	for _, pullStatus := range ResponsePullRequestInformation.PullRequests {
		if pullStatus.Branch == branch && pullStatus.Base == baseBranch && pullStatus.Key == pullRequestID {
			return &webhook_sonar.ResponseForPagePullRequest{
				Status:         pullStatus.Status.QualityGateStatus,
				UrlToSonarQube: fmt.Sprintf("%s/dashboard?id=%s&pullRequest=%s", sonarSettings.URL, sonarSettings.ProjectKey, pullRequestID),
			}, nil
		}
	}
	return &webhook_sonar.ResponseForPagePullRequest{Status: "400"}, fmt.Errorf("GetQualityGatesForProjectByPullRequest check pull request not found in sonarqube")
}
