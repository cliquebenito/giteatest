<template>
  <div v-if="showSuccessAlert" class="ui positive message flash-message flash-success">
    <p>{{ text.alertSuccess }}</p>
  </div>

  <div v-if="showFailureAlert" class="ui negative message flash-message flash-success">
    <p>{{ text.alertFailure }}</p>
  </div>

  <form class="ui form sonar-connect-form" >
    <div class="form-row">
      <div class="field field-url">
        <label for="sonar-server-url">{{ text.labelServerUrl }}</label>
        <input type="text" id="sonar-server-url" name="url" v-model="model.url" :placeholder="text.placeholderServerUrl" autocomplete="off" autofocus required >
      </div>
    </div>
    <div class="form-row">
      <div class="field field-token">
        <label for="sonar-token-auth">{{ text.labelTokenAuth }}</label>
        <input type="text" id="sonar-token-auth" name="token" v-model="model.tokenAuth" :placeholder="text.placeholderTokenAuth" autocomplete="off" autofocus required>
      </div>
      <div class="field field-key">
        <label for="sonar-key-project">{{ text.labelProjectKey }}</label>
        <input type="text" id="sonar-key-project" name="project_key" v-model="model.projectKey" :placeholder="text.placeholderProjectKey" autocomplete="off" required>
      </div>
    </div>
    <div class="form-row">
      <base-button @click.prevent="submitSave" :disabled="isDisabledSaveButton" type="primary">{{ text.buttonSubmit }}</base-button>
      <base-button @click.prevent="submitReset" :disabled="isDisabledResetButton" type="base">{{ text.buttonReset }}</base-button>
    </div>
  </form>
</template>

<script>
  import BaseButton from '../ui/BaseButton.vue'
  const { pageData, csrfToken } = window.config;

  const ALERT_TIMEOUT = 5000;

  export default {
    components: {
      BaseButton
    },
    data: () => ({
      baseLink: pageData.baseLink,
      text: pageData.text,
      url: pageData.url,
      tokenAuth: pageData.tokenAuth,
      projectKey: pageData.projectKey,
      // для разграничеиня модели данных формы и данных пришедших от сервера
      model: {
        url: pageData.url,
        tokenAuth: pageData.tokenAuth,
        projectKey: pageData.projectKey,
      },
      showSuccessAlert: false,
      showFailureAlert: false,
      alertTimeoutId: null,
      errorMessage: null

    }),

    computed: {
      isDisabledSaveButton() {
        const { url, projectKey, tokenAuth } = this.model;
        return (!url || !tokenAuth || !projectKey) ||
          (this.url === url && this.tokenAuth === tokenAuth && this.projectKey === projectKey);
      },

      isDisabledResetButton() {
        const { url, projectKey, tokenAuth } = this.model;
        return (!url || !tokenAuth || !projectKey);
      }
    },


    beforeDestroy() {
      clearTimeout(this.alertTimeoutId);
    },

    methods: {
      submitSave() {
        const formData = new FormData();
        formData.append('_csrf', csrfToken);
        formData.append('token', this.model.tokenAuth);
        formData.append('project_key', this.model.projectKey);
        formData.append('url', this.model.url);

        fetch(this.baseLink, {
          method: 'post',
          body: formData
        }).then(res => {
          if (!res.ok) {
            return Promise.reject({ message: `${res.status}: ${res.statusText}`});
          }
          this.showSuccessAlert = true;
          this.projectKey = this.model.projectKey;
          this.url = this.model.url;
          this.tokenAuth = this.model.tokenAuth;
          this.alertTimeoutId = setTimeout(() => this.showSuccessAlert = false, ALERT_TIMEOUT);
        }).catch(err => {
          this.errorMessage = err.message;
          this.showFailureAlert = true;
          this.alertTimeoutId = setTimeout(() => {
            this.showFailureAlert = false;
            this.errorMessage = null;
          }, ALERT_TIMEOUT);
        });
      },

      submitReset() {
        fetch(this.baseLink, {
          method: 'delete'
        }).then((res) => {
          if (!res.ok) {
            return Promise.reject({ message: `${res.status}: ${res.statusText}`});
          }
          this.showSuccessAlert = true;
          this.model.tokenAuth = '';
          this.model.url = '';
          this.model.projectKey = '';
          this.alertTimeoutId = setTimeout(() => this.showSuccessAlert = false, ALERT_TIMEOUT);
        }).catch(err => {
          this.errorMessage = err.message;
          this.showFailureAlert = true;
          this.alertTimeoutId = setTimeout(() => {
            this.showFailureAlert = false;
            this.errorMessage = null;
          }, ALERT_TIMEOUT);
        });
      },
    }
  }
</script>
