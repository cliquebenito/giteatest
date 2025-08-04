<template>

  <div v-if="alert.message" class="ui message flash-message" :class="{ 'positive': alert.type === 'success',  'negative': alert.type === 'failure',}">
    <p>{{ alert.message }}</p>
  </div>

  <form class="ui form git-hook-form">
    <div class="form-row">
      <div class="field" :class="{ invalid: errors.path }">
        <label for="git-hook-file-path">{{ t('labelFile') }}</label>
        <input type="text" id="git-hook-file-path" name="path"  v-model="model.path" :placeholder="t('placeholderFile')">
        <div v-if="errors.path" class="hint-field">
          <svg-icon name="octicon-alert"></svg-icon>
          {{ errors.path }}
        </div>
      </div>

      <div class="field" :class="{ invalid: errors.timeout }">
        <label for="git-hook-timeout">{{ t('labelTimeout') }}</label>
        <input type="number" id="git-hook-timeout" name="timeout" v-model="model.timeout" :placeholder="t('placeholderTimeout')">
        <div v-if="errors.timeout" class="hint-field">
          <svg-icon name="octicon-alert"></svg-icon>
          {{ errors.timeout }}
        </div>
      </div>
    </div>

    <div class="field" :class="{ invalid: errors.parameters }">
      <label for="git-hook-script-params">{{ t('labelScriptParams') }}</label>
      <textarea id="git-hook-script-params" name="params" v-model="model.parameters"></textarea>
      <div v-if="errors.parameters" class="hint-field">
        <svg-icon name="octicon-alert"></svg-icon>
        {{ errors.parameters }}
      </div>
    </div>

    <div class="hint">
      <ul>
        <li>{{ t('hints.onePerLine') }}</li>
        <li>{{ t('hints.lineValues') }}</li>
        <li>{{ t('hints.paramsValue') }}</li>
      </ul>
    </div>

    <div class="controls">
      <button @click.prevent="submitSave" :disabled="isDisabledSaveButton" class="sc-button sc-button_primary">{{ $t('buttons.save') }}</button>
      <button @click.prevent="submitReset" :disabled="isDisabledResetButton" class="sc-button sc-button_basic">{{ $t('buttons.reset') }}</button>
    </div>
  </form>
</template>

<script>
  import {SvgIcon} from '../svg.js';
  import { useI18n } from 'vue-i18n'
  const { pageData, csrfToken } = window.config;

  const ALERT_TIMEOUT = 5000;

  export default {
    components: {SvgIcon},

    data() {
      const { positionalParameters, path, timeout } = pageData;
      return ({
        baseLink: pageData.baseLink,
        path: path,
        timeout: timeout,
        parameters: positionalParameters,
        model: {
          path: path,
          timeout: timeout,
          parameters: positionalParameters
        },
        alert: {
          type: null,
          message: ''
        },
        errors: {
          path: '',
          timeout: '',
          parameters: ''
        },
        alertTimeoutId: null
      })
    },

    setup() {
      const { t, locale } = useI18n({
        inheritLocale: true,
        useScope: 'local'
      })
      return { t, locale }
    },

    computed: {
      isDisabledSaveButton() {
        return (!this.model.path || !this.model.path.endsWith('.sh')) || !this.model.timeout
      },

      isDisabledResetButton() {
        return !this.model.path
      }
    },

    beforeDestroy() {
      clearTimeout(this.alertTimeoutId);
    },

    methods: {

      resetAlert() {
        this.alertTimeoutId = setTimeout(() => {
          this.alert = {
            type: null,
            message: ''
          }
        }, ALERT_TIMEOUT)
      },

      resetErrors() {
        this.errors = {
          path: '',
          timeout: '',
          parameters: '',
        };
      },

      submitSave() {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);
        formData.append('path', this.model.path);
        formData.append('timeout', this.model.timeout);
        formData.append('parameters', this.model.parameters);
        this.fetchData(formData, 'POST');
      },

      submitReset() {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);
        this.fetchData(formData, 'DELETE', () => {
          this.model = {
            path: '',
            timeout: '',
            parameters: ''
          }
        });
      },

      async fetchData(formData, method, callback) {
        this.resetErrors();
        clearTimeout(this.alertTimeoutId);
        try {
          const response = await fetch(this.baseLink, { method: method, body: formData})
          if (!response.ok) {
            const message = await response.text();
            throw new Error(`${message}`);
          }
          this.alert = {
            type: 'success',
            message: this.t('alertSuccess')
          }
          if (callback) {
            callback();
          }
        } catch (error) {
          this.alert = {
            type: 'failure',
            message: this.t('alertFailure', { text: error })
          }
        }
        this.resetAlert();
      },
    }
  }
</script>



<i18n>
{
  "en-US": {
    "labelFile": "Executed file",
    "placeholderFile": "Enter path to execute file",
    "labelTimeout": "Timeout",
    "placeholderTimeout": "Enter timeout in ms",
    "labelScriptParams": "Additional params for script",
    "hints": {
      "onePerLine": "Pass params one per line",
      "lineValues": "For each line, values from $ 1 to $ n are assigned ($ n-oh values are assigned by variable inside the script and replaced by passed values);",
      "paramsValue": "Any values can be pass as params"
    },
    "alertSuccess": "Data success updated",
    "alertFailure": "Can't update data: {text}"
  },

  "ru-RU": {
    "labelFile": "Исполняемый файл",
    "placeholderFile": "Выберите путь до исполняемого файла",
    "labelTimeout": "Таймаут",
    "placeholderTimeout": "Задайте время таймаута в мс",
    "labelScriptParams": "Дополнительные параметры для скрипта",
    "hints": {
      "onePerLine": "Параметры должны передаваться по одному в строке;",
      "lineValues": "Для каждой строки присваиваются значения от $1 до $N (значения $N-ое присваивается переменным внутри скрипта и заменяются передаваемыми значениями);",
      "paramsValue": "В качестве параметра может передаваться любое значение;"
    },
    "alertSuccess": "Данные успешно обновлены",
    "alertFailure": "Невозможно обновить данные. {text}"
  }
}
</i18n>


