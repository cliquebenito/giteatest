<template>
  <div class="wrap" v-if="qualityGateStatus">
    <div class="header">
      <div class="status">
        {{ t('codeStatus') }} <quality-label :type="qualityGateStatus" />
      </div>
      <div class="updated">
        <span>{{ t('updated') }}:</span> {{ analysedAt }}
      </div>
    </div>

    <div class="metrics-container">
      <metric-item v-for="metric in metrics" :key="metric.key" :metric="metric" :sonarServerHost="serverUrl" :projectKey="projectKey" :branch="branchName" />
    </div>
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue';

  import MetricItem from './MetricItem.vue';
  import QualityLabel from './QualityLabel.vue';
  import { useI18n } from 'vue-i18n';

  const { baseUrl, csrfToken, repositoryId, branchName } = window.config;

  const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
  });

  const qualityGateStatus = ref('');
  const analysedAt = ref('');
  const metrics = ref([]);
  const serverUrl = ref('');
  const projectKey = ref('');

  const baseLinkClean = baseUrl.replace(/\/src\/branch\/[^/]+$/, '');
  const METRICS_API_URL = `${baseLinkClean}/sonarqube/metrics`;

  onMounted(() => {
    const body = new FormData();
    body.append('_csrf', csrfToken);
    body.append('repository_id', repositoryId);
    body.append('branch', branchName);

    fetch(METRICS_API_URL, {
      method: 'post',
      body: body
    })
      .then(res => {
        if (!res.ok) {
          throw new Error(`Can't get SonarQube quality gate status`)
        } else {
          return res.json();
        }
      })
      .then((data) => {
        if (!data) {
          throw new Error(`Unexpected response: ${JSON.stringify(data)}`);
        }
        qualityGateStatus.value = data.sonarQubeStatus.toLowerCase();
        analysedAt.value = (new Date(data.analysedAt)).toLocaleString();
        metrics.value = data.metrics;
        serverUrl.value = data.sonarUrl;
        projectKey.value = data.sonarProjectKey;
      })
      .catch(err => {
        console.warn(err.message);
      });

  });
</script>

<style scoped>

  .wrap {
    --category-a-color: #2EB873;
    --category-b-color: #6FA8F7;
    --category-c-color: #FFD24C;
    --category-d-color: #F49D25;
    --category-e-color: #E84F30;
  }

  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
    font-weight: 400;
  }

  .status {
    font-size: 15px;
    color: var(--color-text-grey);
  }

  .updated {
    font-size: 13px;
    color: #737B8C;
  }

  .metrics-container {
    margin: 40px 0;
    display: flex;
    flex-direction: row;
    align-items: flex-start;
    row-gap: 20px;
    flex-wrap: wrap;
    justify-content: space-around;
  }

  .container::after {
    content: '';
    flex: auto;
  }
</style>


<i18n>
{
  "en-US": {
    "codeStatus": "Code check",
    "updated": "Updated"
  },

  "ru-RU": {
    "codeStatus": "Проверка кода",
    "updated": "Обновлено"
  }
}
</i18n>
