<template>
  <div class="item sonar-check" v-if="status">
    <div class="sonar-check__label" :class="`sonar-check__label_${status}`">{{ statusLabel }}</div>
    <div class="sonar-check__desc">{{ descriptionLabel }}</div>
    <a v-if="link" class="sonar-check__link" :href="link" target="_blank">{{ t('read_more') }}</a>
  </div>
  <div class="ui divider"></div>
</template>

<script setup>
  import { onMounted, ref, computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  const { repositoryId, branchName, repositoryName, projectName } = window.config.pageData.sonarQualityPullRequest;
  const { csrfToken, appUrl } = window.config

  const PULL_REQUEST_API_URL = `${appUrl}${projectName}/${repositoryName}/sonarqube/metrics/pull`;

  const { t, te } = useI18n({
    inheritLocale: true,
    useScope: 'local',
    formatFallbackMessages: true,
  });

  const status = ref('');
  const link = ref('');


  const statusLabel = computed(() => {
    if (te(`${status.value}.label`)) {
      return t(`${status.value}.label`)
    } else {
      console.warn(`Can't get label by status ${status.value}`);
      return 'No status info'
    }
  });


  const descriptionLabel = computed(() => {
    if (te(`${status.value}.desc`)) {
      return t(`${status.value}.desc`)
    } else {
      console.warn(`Can't get description by status ${status.value}`);
      return '';
    }
  });

  const classNameByStatus = computed(() => {
    if (status.value) {
      return `sonar-check__label_${status.value}`
    } else {
      return 'sonar-check__label_none'
    }
  });

  onMounted(() => {
    const body = new FormData();
    body.append('_csrf', csrfToken);
    body.append('repository_id', repositoryId);
    body.append('branch', branchName);



    fetch(PULL_REQUEST_API_URL, {
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
        if (data && data.status && data.urlToSonarQube) {
          status.value = data.status.toLowerCase().trim();
          link.value = data.urlToSonarQube;
        } else {
          throw new Error(`Unexpected response: ${JSON.stringify(data)}`);
        }
      })
      .catch(err => {
        console.warn(err.message);
      })
  })

</script>

<style scoped>


  .sonar-check {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .sonar-check__label {
    padding: 0 8px;
    border-radius: 16px;
    font-size: 12px;
    height: 20px;
    display: inline-flex;
    align-items: baseline;
  }

  .sonar-check__label_success {
    background-color: #2EB873;
    color: #fff;
    text-transform: uppercase;
  }
  .sonar-check__label_warning {
    background-color: #FFD24C;
    color: var(--nav-item-color);
  }
  .sonar-check__label_error {
    background-color: #E84F30;
    color: #fff;
  }
  .sonar-check__label_none {
    background-color: #EAEEF1;
    color: var(--nav-item-color);
  }


  .sonar-check__desc {
    font-size: 13px;
    color: var(--nav-item-color);
  }


  .sonar-check__link {
    color: var(--nav-item-active-background);
    font-size: 13px;
  }
</style>

<i18n>
{
  "en-US": {
    "success": {
      "label": "OK",
      "desc": "SonarQube verification was successful"
    },
    "warning": {
      "label": "warning",
      "desc": "There are warnings as a result of checking SonarQube"
    },
    "error": {
      "label": "Failure",
      "desc": "There are errors as a result of checking SonarQube"
    },
    "none": {
      "label": "Not verify",
      "desc": "SonarQube check was not performed"
    },
    "read_more": "Read more"
  },

  "ru-RU": {
    "success": {
      "label": "OK",
      "desc": "Проверка в SonarQube успешно пройдена"
    },
    "warning": {
      "label": "Есть замечания",
      "desc": "По итогам проверки в SonarQube есть замечания"
    },
    "error": {
      "label": "Есть ошибки",
      "desc": "По итогам проверки в SonarQube найдены ошибки"
    },
    "none": {
      "label": "Не удалось проверить",
      "desc": "Проверка в SonarQube не была проведена"
    },
    "read_more": "Узнать подробнее"
  }
}
</i18n>
