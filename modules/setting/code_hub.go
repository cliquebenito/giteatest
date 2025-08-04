package setting

import "strings"

var CodeHub struct {
	CodeHubMetricEnabled bool

	CodeHubUsagesProcessorIntervalSeconds  int64
	CodeHubCounterProcessorIntervalSeconds int64
	CodeHubMarkLabelName                   string
	CodeHubMarkEnabled                     bool
	InternalMetricsNamesList               []string
}

// loadCodeHub - метод загрузки CODEHUB_MARK_LABEL_NAME в app.ini
func loadCodeHub(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sourcecontrol.codehub")

	CodeHub.CodeHubMetricEnabled = sec.Key("CODEHUB_METRIC").MustBool(false)
	CodeHub.CodeHubUsagesProcessorIntervalSeconds = sec.Key("CODEHUB_USAGES_INTERVAL_SECONDS").MustInt64(10)
	CodeHub.CodeHubCounterProcessorIntervalSeconds = sec.Key("CODEHUB_COUNTER_INTERVAL_SECONDS").MustInt64(120)
	CodeHub.CodeHubMarkLabelName = sec.Key("CODEHUB_MARK_LABEL_NAME").MustString("InSourceHub")
	CodeHub.CodeHubMarkEnabled = sec.Key("CODEHUB_MARK_ENABLED").MustBool(false)
	CodeHub.InternalMetricsNamesList = parseInternalMetricsNamesList(sec)
}

func parseInternalMetricsNamesList(codeHubSection ConfigSection) []string {
	metricNames := codeHubSection.
		Key("INTERNAL_METRIC_NAMES_LIST").String()

	if metricNames == "" {
		return []string{}
	}

	metricNames = strings.ReplaceAll(metricNames, " ", "")

	return strings.Split(metricNames, ",")
}
