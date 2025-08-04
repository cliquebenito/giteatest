<template>
    <div class="aside-panel">
        <div class="aside-panel-content">
          <div class="aside-panel__header">
            <h2 class="aside-panel__title">{{ t('title') }}</h2>
            <button class="aside-panel__close" :disabled="isLoading" @click="cancel"><svg-icon name="octicon-x"></svg-icon></button>
          </div>

          <div v-if="allRepositories.length">
            <div class="aside-panel-content__search">
              <search-field @filterChange="handleFilterChange($event)"></search-field>
            </div>
            <div class="aside-panel-list">
              <div v-if="filteredRepos.length > 0">
                <repository-card
                  @selectRepo="handleSelectRepository($event)"
                  :isSelected="checkIsSelected(repo)"
                  :isLoading="isLoading"
                  :add-mode="true"
                  :repository="repo"
                  :key="repo.ID"
                  v-for="(repo, index) in filteredRepos">
                </repository-card>
              </div>
              <div v-else>
                {{`${t('noRepositoriesByFilter')}: "${filter}"`}}
              </div>
            </div>
          </div>
          <div v-else>
            {{ t('noRepositories') }}
            <svg-icon name="no-members"></svg-icon>
          </div>

          <div class="aside-panel-controls">
            <button :disabled="isLoading" class="ui button primary" :class="{'loading': isLoading}" @click="handleSubmit">{{ t('addButton') }}</button>
            <button :disabled="isLoading" class="ui button" @click="cancel">{{ t('cancelButton') }}</button>
          </div>
        </div>
    </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';
import RepositoryCard from './RepositoryCard.vue';
import SearchField from './SearchField.vue';
import {useDynamicUrl} from "../useDynamicUrl.js";

const { csrfToken, appSubUrl } = window.config;
const { orgLink, team: { LowerName: teamLowerName } }  = window.config.pageData;

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const props = defineProps({
  isLoading: {
    type: Boolean,
  },
  allRepositories: {
      type: Array,
      required: true
  },
  repositories: {
      type: Array,
      required: true
  }
});

const selectedRepos = ref(props.repositories.map(item => `${item.OwnerName}/${item.LowerName}/${item.ID}`));
const filter = ref('');


const emit = defineEmits(['toggleReposPanel']);

const filteredRepos = computed(() => {
  return props.allRepositories.filter(item => {
    const searchValue = `${item.LowerName}`;
    return searchValue.includes(filter.value);
  });
});

const handleFilterChange = (value) => {
  filter.value = value;
};

const { getFullUrl } = useDynamicUrl({
  link: orgLink,
  appSubUrl: appSubUrl,
});

const handleSubmit = () => {

  const URL = getFullUrl(`/teams/${teamLowerName}/action/repo/addrepos`);
  const repoIds = selectedRepos.value.map(item => Number(item.split('/')[2]));

  fetch(URL, {
    method: 'post',
    body: JSON.stringify({repo_ids: repoIds}),
    headers: { 'Content-Type': 'application/json', 'X-Csrf-Token': csrfToken },
  })
    .then(res => {
      emit('toggleReposPanel', false);
      location.reload();
    })
    .catch(err => console.warn('err', err))
};

const cancel = () => {
  emit('toggleReposPanel', false);
};

const handleSelectRepository = (data) => {
  const { action, id } = data;
  if (action === 'add') {
    selectedRepos.value = [ ...selectedRepos.value, id];
  } else if (action === 'remove') {
    selectedRepos.value = selectedRepos.value.filter(item => item !== id);
  }
};

const checkIsSelected = (repo) => {
  const id = `${repo.OwnerName}/${repo.LowerName}/${repo.ID}`;
  return selectedRepos.value.includes(id);
};

</script>

<style scoped>
.aside-panel__close {
  background: none;
  border: none;
  cursor: pointer;
}
.aside-panel__close svg {
  width: 24px;
  height: 24px;
}
.aside-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.aside-panel__title {
  font-weight: 600;
  color: #263238;
  font-size: 28px;
  margin: 0;
}
.aside-panel-content__search {
  margin-bottom: 24px;
}
.aside-panel {
    position: fixed;
    width: 443px;
    height: 100vh;
    background: var(--color-box-body);
    right: 0;
    top: 0;
    overflow: hidden;
    z-index: 1000;
  }

  .aside-panel::before {
    content: '';
    display: block;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    position: fixed;
    background: black;
    opacity: .4;
  }

  .aside-panel-content {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: var(--color-box-body);
    padding: 32px;
    box-sizing: border-box;
    overflow-y: auto;
    height: 100%;
  }

  .aside-panel-content.block::after {
    content: '';
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: black;
    opacity: .4;
    z-index: 1001;
  }

  .aside-panel-list {
    height: calc(100vh - 220px);
    overflow-y: auto;
  }
  .aside-panel-controls {
    display: flex;
    align-items: center;
    column-gap: 16px;
    justify-content: space-between;
  }
  .aside-panel-controls button {
    min-width: 180px;
  }
</style>

<i18n>
 {
  "en-US": {
    "addButton": "Add",
    "cancelButton": "Cancel",
    "title": "Add repositories",
    "noRepositories": "No repositories",
    "noRepositoriesByFilter": "No repositories by filter"
  },

  "ru-RU": {
    "addButton": "Добавить",
    "cancelButton": "Отменить",
    "noRepositories": "No members",
    "noRepositoriesByFilter": "Нет репозиториев по фильтру",
    "title": "Добавить репозиторий"
  }
}
</i18n>
