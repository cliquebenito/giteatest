<template>
  <!--
  if this component is shown, either the user is an admin (can do a merge without checks), or they are a writer who has the permission to do a merge
  if the user is a writer and can't do a merge now (canMergeNow==false), then only show the Auto Merge for them
  How to test the UI manually:
  * Method 1: manually set some variables in pull.tmpl, eg: {{$notAllOverridableChecksOk = true}} {{$canMergeNow = false}}
  * Method 2: make a protected branch, then set state=pending/success :
    curl -X POST ${root_url}/api/v1/repos/${owner}/${repo}/statuses/${sha} \
      -H "accept: application/json" -H "authorization: Basic $base64_auth" -H "Content-Type: application/json" \
      -d '{"context": "test/context", "description": "description", "state": "${state}", "target_url": "http://localhost"}'
  -->
  <div>
    <div class="item sonar-check" v-if="sonarQubeQualityGate.status">
      <div class="sonar-check__label" :class="`sonar-check__label_${sonarQubeQualityGate.status}`">{{ sonarQubeQualityGateLabel }}</div>
      <div class="sonar-check__desc">{{ sonarQubeQualityGateDescription}}</div>
      <a v-if="sonarQubeQualityGate.link" class="sonar-check__link" :href="sonarQubeQualityGate.link" target="_blank">{{ t('read_more') }}</a>
    </div>
    <div class="ui divider"></div>

    <div class="item item-section">
      <div class="item-section-left">
        <div v-if="buildState === 'pending'">
          <svg-icon name="octicon-history" class="icon-pending"></svg-icon>
          <span class="status">{{ mergeForm.textBuildPending }}</span>
        </div>
        <div v-else-if="buildState === 'success'">
          <svg-icon name="octicon-check" class="text green"></svg-icon>
          <span class="status">{{ mergeForm.textBuildSuccess }}</span>
          <span class="updated">{{ updated }}</span>
        </div>
        <div v-else-if="buildState === 'failure'">
          <svg-icon name="octicon-x-circle" class="text red"></svg-icon>
          <span class="status">{{ mergeForm.textBuildFailure }}</span>
          <span class="updated">{{ updated }}</span>
        </div>
        <div v-else-if="buildState === 'error'">
          <svg-icon name="octicon-alert" class="text red"></svg-icon>
          <span class="status">{{ mergeForm.textBuildError }}</span>
        </div>
      </div>
      <div class="item-section-right">
        <base-button type="base" @click="restartJenkinsBuild"  v-bind:disabled="disabledBuildButton">
              {{ mergeForm.textRunBuildButton }}
        </base-button>
      </div>
    </div>
    <div class="ui divider"></div>

    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-if="mergeForm.hasPendingPullRequestMerge" v-html="mergeForm.hasPendingPullRequestMergeTip" class="ui info message"/>

    <div class="ui form" v-if="showActionForm">
      <form :action="mergeForm.baseLink+'/merge'" method="post">
        <input type="hidden" name="_csrf" :value="csrfToken">
        <input type="hidden" name="head_commit_id" v-model="mergeForm.pullHeadCommitID">
        <input type="hidden" name="merge_when_checks_succeed" v-model="autoMergeWhenSucceed">
        <input type="hidden" name="force_merge" v-model="forceMerge">

        <template v-if="!mergeStyleDetail.hideMergeMessageTexts">
          <div class="field">
            <input type="text" name="merge_title_field" v-model="mergeTitleFieldValue">
          </div>
          <div class="field">
            <textarea name="merge_message_field" rows="5" :placeholder="mergeForm.mergeMessageFieldPlaceHolder" v-model="mergeMessageFieldValue"/>
            <template v-if="mergeMessageFieldValue !== mergeForm.defaultMergeMessage">
              <button @click.prevent="clearMergeMessage" class="ui tertiary button">
                {{ mergeForm.textClearMergeMessage }}
              </button>
              <div class="ui label">
                <!-- TODO: Convert to tooltip once we can use tooltips in Vue templates -->
                {{ mergeForm.textClearMergeMessageHint }}
              </div>
            </template>
          </div>
        </template>

        <div class="field" v-if="mergeStyle === 'manually-merged'">
          <input type="text" name="merge_commit_id" :placeholder="mergeForm.textMergeCommitId">
        </div>

        <button class="ui button" :class="mergeButtonStyleClass" type="submit" name="do" :value="mergeStyle">
          {{ mergeStyleDetail.textDoMerge }}
          <template v-if="autoMergeWhenSucceed">
            {{ mergeForm.textAutoMergeButtonWhenSucceed }}
          </template>
        </button>

        <button class="ui button merge-cancel" @click="toggleActionForm(false)">
          {{ mergeForm.textCancel }}
        </button>

        <div class="ui checkbox gt-ml-2" v-if="mergeForm.isPullBranchDeletable && !autoMergeWhenSucceed && mergeForm.canDeleteBranch">
          <input name="delete_branch_after_merge" type="checkbox" v-model="deleteBranchAfterMerge" id="delete-branch-after-merge">
          <label for="delete-branch-after-merge">{{ mergeForm.textDeleteBranch }}</label>
        </div>
      </form>
    </div>

    <div v-if="!showActionForm" class="gt-df">
      <!-- the merge button -->
      <div class="ui buttons merge-button" :class="[mergeForm.emptyCommit ? 'grey' : mergeForm.allOverridableChecksOk ? 'basic' : 'red', accessToMerge ? 'red': 'grey disabled' ]"  @click="toggleActionForm(true)" >
        <button class="ui button" :disabled="!accessToMerge" >
          <svg-icon name="octicon-git-merge"/>
          <span class="button-text">
            {{ mergeStyleDetail.textDoMerge }}
            <template v-if="autoMergeWhenSucceed">
              {{ mergeForm.textAutoMergeButtonWhenSucceed }}
            </template>
          </span>
        </button>
        <div class="ui dropdown icon button no-text" @click.stop="showDropdown" v-if="mergeStyleAllowedCount>1">
          <svg-icon name="octicon-triangle-down" :size="14"/>
          <div class="menu" :class="{'show':showMergeStyleMenu}">
            <template v-for="msd in mergeForm.mergeStyles">
              <!-- if can merge now, show one action "merge now", and an action "auto merge when succeed" -->
              <div class="item" v-if="msd.allowed && mergeForm.canMergeNow" :key="msd.name" @click.stop="switchMergeStyle(msd.name)">
                <div class="action-text">
                  {{ msd.textDoMerge }}
                </div>
                <div v-if="!msd.hideAutoMerge" class="auto-merge-small" @click.stop="switchMergeStyle(msd.name, true)">
                  <svg-icon name="octicon-clock" :size="14"/>
                  <div class="auto-merge-tip">
                    {{ mergeForm.textAutoMergeWhenSucceed }}
                  </div>
                </div>
              </div>

              <!-- if can NOT merge now, only show one action "auto merge when succeed" -->
              <div class="item" v-if="msd.allowed && !mergeForm.canMergeNow && !msd.hideAutoMerge" :key="msd.name" @click.stop="switchMergeStyle(msd.name, true)">
                <div class="action-text">
                  {{ msd.textDoMerge }} {{ mergeForm.textAutoMergeButtonWhenSucceed }}
                </div>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- the cancel auto merge button -->
      <form v-if="mergeForm.hasPendingPullRequestMerge" :action="mergeForm.baseLink+'/cancel_auto_merge'" method="post" class="gt-ml-4">
        <input type="hidden" name="_csrf" :value="csrfToken">
        <button class="ui button">
          {{ mergeForm.textAutoMergeCancelSchedule }}
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import {SvgIcon} from '../svg.js';
import { formatDistanceToNow } from 'date-fns';
import ruLocale from 'date-fns/locale/ru';
import BaseButton from '../ui/BaseButton.vue'
import { useI18n } from 'vue-i18n';

const {csrfToken, pageData} = window.config;
const currentLanguage = document.documentElement.getAttribute('lang');
const POLL_BUILD_STATE = 5000;

export default {
  components: {SvgIcon, BaseButton},

  data: () => ({
    csrfToken,

    mergeForm: pageData.pullRequestMergeForm,

    mergeTitleFieldValue: '',
    mergeMessageFieldValue: '',
    deleteBranchAfterMerge: false,
    autoMergeWhenSucceed: false,

    mergeStyle: '',
    mergeStyleDetail: { // dummy only, these values will come from one of the mergeForm.mergeStyles
      hideMergeMessageTexts: false,
      textDoMerge: '',
      mergeTitleFieldText: '',
      mergeMessageFieldText: '',
      hideAutoMerge: false,
    },
    mergeStyleAllowedCount: 0,

    showMergeStyleMenu: false,
    showActionForm: false,

    buildProgress: false,
    timerId: null,
    buildState: null,
    buildTime: null,

    enableSonarQubeProtectBranch: pageData.sonarQualityPullRequest.enableSonarQubeProtectBranch,
    hasSonarSettings: pageData.sonarQualityPullRequest.hasSonarSettings,
    isAdminCanMergeWithoutChecks: pageData.sonarQualityPullRequest.isAdminCanMergeWithoutChecks,

    sonarQubeQualityGate: {
      label: null,
      desc: null,
      status: null,
      link: null,
    }
  }),

  setup() {
    const { t, te, locale } = useI18n({
      inheritLocale: true,
      useScope: 'local'
    })
    return { t, te, locale }
  },

  computed: {
    disabledBuildButton() {
      return this.buildProgress || this.buildState === 'pending'
    },
    mergeButtonStyleClass() {
      if (this.mergeForm.allOverridableChecksOk) return 'primary';
      return this.autoMergeWhenSucceed ? 'blue' : 'red';
    },
    forceMerge() {
      return this.mergeForm.canMergeNow && !this.mergeForm.allOverridableChecksOk;
    },
    updated() {
      const options = {
        addSuffix: true
      };
      if (currentLanguage.match('ru') !== null) {
        options.locale = ruLocale;
      }
      if (this.buildTime ) {
        return formatDistanceToNow(convertDateWithTimezone(this.buildTime), options);
      } else {
        return ''
      }
    },
    sonarQubeQualityGateLabel() {
      const status = this.sonarQubeQualityGate.status;
      if (this.te(`${status}.label`)) {
        return this.t(`${status}.label`)
      } else {
        console.warn(`Can't get label by status ${status}`);
        return 'No status info'
      }
    },
    sonarQubeQualityGateDescription() {
      const status = this.sonarQubeQualityGate.status;
      if (this.te(`${status}.desc`)) {
        return this.t(`${status}.desc`)
      } else {
        console.warn(`Can't get description by status ${status}`);
        return '';
      }
    },
    sonarQubeQualityGateClassName() {
      const status = this.sonarQubeQualityGate.status;
      if (status) {
        return `sonar-check__label_${status}`;
      } else {
        return 'sonar-check__label_none';
      }
    },
    accessToMerge() {
      return ((this.enableSonarQubeProtectBranch && this.sonarQubeQualityGate.status === 'ok') || !this.enableSonarQubeProtectBranch || this.isAdminCanMergeWithoutChecks) && !this.mergeForm.emptyCommit;
    }
  },

  watch: {
    mergeStyle(val) {
      this.mergeStyleDetail = this.mergeForm.mergeStyles.find((e) => e.name === val);
    }
  },

  created() {
    this.mergeStyleAllowedCount = this.mergeForm.mergeStyles.reduce((v, msd) => v + (msd.allowed ? 1 : 0), 0);

    let mergeStyle = this.mergeForm.mergeStyles.find((e) => e.allowed && e.name === this.mergeForm.defaultMergeStyle)?.name;
    if (!mergeStyle) mergeStyle = this.mergeForm.mergeStyles.find((e) => e.allowed)?.name;
    this.switchMergeStyle(mergeStyle, !this.mergeForm.canMergeNow);
  },

  mounted() {
    this.getBuildState();
    this.getSonarQubeQualityGateStatus();
    document.addEventListener('mouseup', this.hideMergeStyleMenu);
  },

  unmounted() {
    document.removeEventListener('mouseup', this.hideMergeStyleMenu);
    clearInterval(this.timerId);
  },

  methods: {

    getBuildState() {
      return pollBuildStatus.call(this)
        .then((state) => {
          if (state === 'unknown' || state === 'pending') {
            this.timerId = setInterval(pollBuildStatus.bind(this), POLL_BUILD_STATE)
          } else {
            this.buildProgress = false;
            clearInterval(this.timerId);
          }
        });
    },

    restartJenkinsBuild() {
      clearInterval(this.timerId);
      this.buildProgress = true;

      const formData = new FormData();
      formData.append('_csrf', this.csrfToken);

      fetch(`${this.mergeForm.baseLink}/rebuild`, {
        method: 'POST',
        body: formData
      }).then(() => {
        setTimeout(() => this.getBuildState(), POLL_BUILD_STATE)
      })
    },

    hideMergeStyleMenu() {
      this.showMergeStyleMenu = false;
    },

    showDropdown() {
      if (!this.accessToMerge) return;
      this.showMergeStyleMenu = !this.showMergeStyleMenu;
    },

    toggleActionForm(show) {
      if (!this.accessToMerge) return;
      this.showActionForm = show;
      if (!show) return;
      this.deleteBranchAfterMerge = this.mergeForm.defaultDeleteBranchAfterMerge;
      this.mergeTitleFieldValue = this.mergeStyleDetail.mergeTitleFieldText;
      this.mergeMessageFieldValue = this.mergeStyleDetail.mergeMessageFieldText;
    },
    switchMergeStyle(name, autoMerge = false) {
      this.mergeStyle = name;
      this.autoMergeWhenSucceed = autoMerge;
    },
    clearMergeMessage() {
      this.mergeMessageFieldValue = this.mergeForm.defaultMergeMessage;
    },
    getSonarQubeQualityGateStatus() {
      if (!this.hasSonarSettings) {
        return;
      }
      const { repositoryId, headBranch, baseBranch, pullRequestId } = window.config.pageData.sonarQualityPullRequest;
      const { csrfToken, baseLink } = window.config;
      const baseLinkClean = baseLink.replace(/\/pulls\/\d+$/, '');
      const PULL_REQUEST_API_URL = `${baseLinkClean}/sonarqube/metrics/pull`;

      const body = new FormData();
      body.append('_csrf', csrfToken);
      body.append('repository_id', repositoryId);
      body.append('branch', headBranch);
      body.append('base', baseBranch);
      body.append('pull_request_id', pullRequestId);

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
            this.sonarQubeQualityGate.status = data.status.toLowerCase().trim();
            this.sonarQubeQualityGate.link = data.urlToSonarQube;
          } else {
            throw new Error(`Unexpected response: ${JSON.stringify(data)}`);
          }
        })
        .catch(err => {
          console.warn(err.message);
        })
    }

  },
};


function convertDateWithTimezone(updated) {
  const date = new Date(updated)
  return date.toString();
}

function pollBuildStatus() {
  return fetch(`${this.mergeForm.baseLink}/state`)
    .then(res => res.json())
    .then(res => {
      this.buildState = res.state;
      this.buildTime = res.updated;
      return res.state;
    });
}

</script>

<style scoped>
.status,
.updated {
  margin-left: 4px;
}
/* to keep UI the same, at the moment we are still using some Fomantic UI styles, but we do not use their scripts, so we need to fine tune some styles */
.ui.dropdown .menu.show {
  display: block;
}
.ui.checkbox label {
  cursor: pointer;
}

/* make the dropdown list left-aligned */
.ui.merge-button {
  position: relative;
  border: none !important;
}

.ui.merge-button > *:last-child {
  border-bottom-left-radius: 0px !important;
  border-top-left-radius: 0px !important;
  border-top-right-radius: 8px !important;
  border-bottom-right-radius: 8px !important;
}

.ui.merge-button > *:first-child {
  border-bottom-left-radius: 8px !important;
  border-top-left-radius: 8px !important;
  border-top-right-radius: 0px !important;
  border-bottom-right-radius: 0px !important;
}
.ui.merge-button .ui.dropdown {
  position: static;
}
.ui.merge-button > .ui.dropdown:last-child > .menu:not(.left) {
  left: 0;
  right: auto;
}
.ui.merge-button .ui.dropdown .menu > .item {
  display: flex;
  align-items: stretch;
  padding: 0 !important; /* polluted by semantic.css: .ui.dropdown .menu > .item { !important } */
}

/* merge style list item */
.action-text {
  padding: 0.8rem;
  flex: 1
}

.auto-merge-small {
  width: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}
.auto-merge-small .auto-merge-tip {
  display: none;
  left: 38px;
  top: -1px;
  bottom: -1px;
  position: absolute;
  align-items: center;
  color: var(--color-info-text);
  background-color: var(--color-info-bg);
  border: 1px solid var(--color-info-border);
  border-left: none;
  padding-right: 1rem;
}

.auto-merge-small:hover {
  color: var(--color-info-text);
  background-color: var(--color-info-bg);
  border: 1px solid var(--color-info-border);
}

.auto-merge-small:hover .auto-merge-tip {
  display: flex;
}


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

.sonar-check__label_ok {
  background-color: #2EB873;
  color: #fff;
  text-transform: uppercase;
}
.sonar-check__label_warn {
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
  color: var(--color-text);
}


.sonar-check__link {
  color: var(--nav-item-active-background);
  font-size: 13px;
}

.merge-button.disabled:hover,
.merge-button.disabled,
.merge-button.disabled .ui.button {
  cursor: not-allowed;
  opacity: .5;
}



</style>

<i18n>
{
  "en-US": {
    "ok": {
      "label": "OK",
      "desc": "SonarQube verification was successful"
    },
    "warn": {
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
    "ok": {
      "label": "OK",
      "desc": "Проверка в SonarQube успешно пройдена"
    },
    "warn": {
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



