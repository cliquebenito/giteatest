<template>
  <div class="metric" v-if="!isEmptyRating" :data-tooltip-content="metric.name">
    <div class="metric__header">{{ metric.domain }}</div>

    <metric-rating v-if="normalizedMetricType === 'rating'" :value="metric.value"></metric-rating>
    <metric-progress v-else-if="normalizedMetricType === 'percent'" :is-invert="isInvert" :value="metric.value"></metric-progress>
    <metric-bool v-else-if="normalizedMetricType === 'bool'" :value="metric.value"></metric-bool>
    <metric-time v-else-if="normalizedMetricType === 'millisec'" :value="metric.value"></metric-time>
    <metric-int v-else-if="normalizedMetricType === 'int'" :value="metric.value"></metric-int>
    <metric-float v-else-if="normalizedMetricType === 'float'" :value="metric.value"></metric-float>
    <metric-fallback v-else :value="metric.value"></metric-fallback>

    <div class="metric__footer" v-if="hasAdditionalMetric">
      <a :href="additionalMetricUrl" target="_blank" class="metric__aux-value">{{ auxValue }}</a>
      <span class="metric__aux-name" :data-tooltip-content="auxName">{{ auxName }}</span>
    </div>
  </div>
</template>

<script setup>
  import { computed } from 'vue';
  import MetricRating from './MetricRating.vue';
  import MetricBool from './MetricBool.vue';
  import MetricProgress from './MetricProgress.vue';
  import MetricInt from './MetricInt.vue';
  import MetricFloat from './MetricFloat.vue';
  import MetricTime from './MetricTime.vue';
  import MetricFallback from './MetricFallback.vue';
  import { convertToBool, convertToFloat, convertToInt, convertToPercent, convertToRating, normalizedString } from './helpers.js';

  const props = defineProps({
    metric: {
      type: Object,
      required: true
    },
    sonarServerHost: {
      type: String
    },
    projectKey: {
      type: String
    },
    branch: {
      type: String
    }
  });

  const issueMetricUrlList = ['code_smells', 'bugs', 'vulnerability'];

  let auxValue = props.metric.aux_metric_value;
  const auxName = props.metric.aux_metric_name;
  const auxType = normalizedString(props.metric.aux_metric_type);
  const normalizedMetricType = normalizedString(props.metric.type);

  const hasAdditionalMetric = auxValue && auxName;

  const INVERTED_METRICS_LIST = ['new_duplicated_lines_density']
  const isInvert = INVERTED_METRICS_LIST.includes(props.metric.key);

  const isEmptyRating = computed(() => {
    return normalizedMetricType === 'rating' && !props.metric.value;
  });

  const additionalMetricUrl = computed(() => {
    const auxMetricKey = props.metric.aux_metric_key;
    const projectKey = props.projectKey;
    const sonarServerHost = props.sonarServerHost;
    const branch = props.branch;
    if (auxMetricKey.includes(issueMetricUrlList)) {
      return `${sonarServerHost}/project/issues?id=${projectKey}&branch=${branch}&resolved=false&types=${auxMetricKey}`;
    } else {
      return `${sonarServerHost}/component_measures?id=${projectKey}&branch=${branch}&metric=${auxMetricKey}`;
    }

  });


  const auxMetricConvertFnMap = {
    int: convertToInt,
    bool: convertToBool,
    percent: convertToPercent,
    float: convertToFloat,
    rating: convertToRating
  };


  if (hasAdditionalMetric && auxMetricConvertFnMap[auxType]) {
    auxValue = auxMetricConvertFnMap[auxType](auxValue);
  }


</script>

<style scoped>

  .metric {
    width: 120px;
    display: flex;
    flex-direction: column;
    align-items: center;
    row-gap: 8px;
  }

  .metric__header {
    color: var(--color-text);
    font-size: 13px;
    text-align: center;
    line-height: 1;
    font-weight: 600;
  }

  .metric__footer {
    text-align: center;
    font-size: 13px;
    display: flex;
    column-gap: 4px;
    align-items: center;
    line-height: 1;
  }

  .metric__aux-value {
    color: #144BB8
  }

  .metric__aux-name {
    color: #737B8C;
    white-space: nowrap;
    text-overflow: ellipsis;
    overflow: hidden;
    max-width: 90px;
  }
</style>
