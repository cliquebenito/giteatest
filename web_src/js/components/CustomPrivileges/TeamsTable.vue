<template>
  <easy-data-table
    ref="dataTable"
    :headers="headers"
    :items="teams"
    :loading="loading"
    :theme-color="'#1976D2'"
    table-class-name="teams-table"
    body-row-class-name="row"
    body-item-class-name="cell"
    buttons-pagination
    hide-footer
    :rows-per-page="ROWS_PER_PAGE"
  >
    <template #item-LowerName="Team">
      <a class="team-cell" :href="`${baseLink}/${Team.LowerName}`">
        <span class="team-cell__name">
          {{ Team.Name }}
        </span>
        <span class="team-cell__desc">
          {{ Team.Description }}
        </span>
      </a>
    </template>

    <template #item-members="Team">
      <div class="members-cell">
        <svg-icon name="users" />
        {{ Team.NumMembers }}
        {{
          getNoun(Team.NumMembers, [
            t("team.members.singular").toLowerCase(),
            t("team.members.genetive").toLowerCase(),
            t("team.members.genetivePlural").toLowerCase(),
          ])
        }}
      </div>
    </template>

    <template #item-repos="Team">
      <div class="repos-cell">
        <svg-icon name="repo-book" />
        {{ Team.NumRepos }}
        {{
          getNoun(Team.NumRepos, [
            t("team.repos.singular").toLowerCase(),
            t("team.repos.genetive").toLowerCase(),
            t("team.repos.genetivePlural").toLowerCase(),
          ])
        }}
      </div>
    </template>

    <template #item-actions="Team">
      <div class="controls-cell">
        <base-button type="transparent" icon :href="`${baseLink}/${Team.Name}`" :data-tooltip-content="t('editTeamButtonTooltip')">
          <svg-icon name="octicon-pencil"/>
        </base-button>
        <base-button type="transparent" icon @click="handleDeleteTeam(Team)" :data-tooltip-content="t('deleteTeamButtonTooltip')">
          <svg-icon name="octicon-trash"/>
        </base-button>
      </div>
    </template>

    <template #empty-message>
      <p class="empty-message">{{ $t("privileges.table.emptyMessage") }}</p>
    </template>
  </easy-data-table>

  <div v-if="ROWS_PER_PAGE < teams.length" class="teams-table-footer">

    <div class="pagination">
      <base-button small type="transparent" class="button-pagination" @click="prevPage" :disabled="isFirstPage">
        <svg-icon name="octicon-chevron-left" />
      </base-button>

      <div class="pagination__pages">
        <base-button
          v-for="paginationNumber in maxPaginationNumber"
          small
          class="button-pagination"
          :type="paginationNumber === currentPaginationNumber ? 'primary' : 'transparent'"
          @click="updatePage(paginationNumber)"
        >
          {{paginationNumber}}
        </base-button>
      </div>

      <base-button small type="transparent" class="button-pagination" @click="nextPage" :disabled="isLastPage">
        <svg-icon name="octicon-chevron-right" />
      </base-button>
    </div>


  </div>

  <confirm-modal
    :isLoading="isLoading"
    @modalConfirm="handleConfirmDeleteTeam"
    @modalCancel="handleCancelDeleteTeam"
    :show="showModal"
    :title="t('modal.title')"
    :description="t('modal.desc')"
    :confirmButtonText="t('modal.confirmButton')">

  </confirm-modal>
</template>

<script setup>
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';

import EasyDataTable from 'vue3-easy-data-table';
import { usePagination } from 'use-vue3-easy-data-table';
import 'vue3-easy-data-table/dist/style.css';

import BaseButton from '../../ui/BaseButton.vue';
import ConfirmModal from '../../ui/ConfirmModal.vue';

import { getNoun } from '../../utils/get-noun';
import {useDynamicUrl} from "../useDynamicUrl.js";

const { t } = useI18n({
   inheritLocale: true,
   useScope: 'local'
});

const { csrfToken, appSubUrl} = window.config;
const { baseLink } = window.config.pageData;

const deletedTeam = ref(null);
const showModal = ref(false);
const isLoading = ref(false);

const props = defineProps({
    teams: {
      type: Array,
    },
    loading: {
      type: Boolean
    }
});

const emits = defineEmits(['onEditTeam', 'onDeleteTeam']);


const ROWS_PER_PAGE = 25;

const dataTable = ref();

const {
  maxPaginationNumber,
  currentPaginationNumber,
  isFirstPage,
  isLastPage,
  nextPage,
  prevPage,
  updatePage,
} = usePagination(dataTable);

const headers = ref([
  { text: t('team.name'), value: 'LowerName', sortable: true, width: '100%' },
  { text: t('team.members.plural'), value: 'members', width: 320 },
  { text: t('team.repos.plural'), value: 'repos', width: 320 },
  { text: t('actionsHeader'), value: 'actions', width: 90 }
]);

const handleDeleteTeam = (team) => {
  deletedTeam.value = team;
  showModal.value = true;
};

const { getFullUrl } = useDynamicUrl({
  link: baseLink,
  appSubUrl: appSubUrl,
});

const handleConfirmDeleteTeam = () => {
  const URL = getFullUrl(`${deletedTeam.value.Name}/delete`);

  const formData = new FormData();
  formData.append('_csrf', csrfToken);
  isLoading.value = true;
  fetch(URL, {
      method: 'post',
      body: formData
  }).then(() => {
      deletedTeam.value = null;
      showModal.value = false;
      location.reload();
      isLoading.value = false;
  }).catch(err => {
      console.warn('handleDeleteTeam ERROR', err)
  });

};

const handleCancelDeleteTeam = () => {
  showModal.value = false;
  deletedTeam.value = null;
};


// reset page after search
watch(() => props.teams, () => {
  updatePage(1)
})

</script>

<style scoped>
.team-cell {
  display: flex;
  flex-direction: column;
  row-gap: 4px;
}

.team-cell:hover {
  opacity: .8;
  text-decoration: none;
}

.team-cell__name {
  color: #1976D2;
  font-size: 15px;
  font-weight: 500;
  line-height: 15px;
}

.team-cell__desc {
  color: #78909C;
  font-size: 13px;
  line-height: 13px;
}

.members-cell {
  display: flex;
  align-items: center;
  color: #78909C;
  column-gap: 4px;
}

.repos-cell {
  display: flex;
  align-items: center;
  color: #78909C;
  column-gap: 4px;
}

.controls-cell {
  display: flex;
  align-items: center;
  column-gap: 8px;
}

.teams-table-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 24px;
}


.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  column-gap: 4px;
}

.pagination__pages {
  display: flex;
  align-items: center;
  justify-content: center;
  column-gap: 4px;
}

.button-pagination {
  padding: 0 8px;
}

.teams-table {
  --easy-table-border: none;
  --easy-table-row-border: 1px solid #D3DBDF;

  --easy-table-header-item-padding: 12px 16px;
  --easy-table-header-font-size: 13px;

  --easy-table-body-row-height: auto;
  --easy-table-body-item-padding: 12px 16px;
  --easy-table-body-row-font-color: #F3F5F6;
  --easy-table-body-row-font-size: 13px;

  --easy-table-body-row-hover-background-color: var(--color-hover);

  --easy-table-loading-mask-background-color: var(--color-primary);
}

.teams-table .header-text {
  font-weight: 500;
}

.theme-dark .teams-table {
  --easy-table-header-font-color: var(--color-caret);
  --easy-table-header-background-color: var(--color-body);

  --easy-table-body-row-background-color: var(--color-body);
  --easy-table-body-row-font-color: var(--color-text);

  --easy-table-body-row-hover-font-color: #737b8c;
  --easy-table-body-row-hover-background-color: #282e3f;
}
</style>

<i18n>
  {
  "en-US": {
    "actionsHeader": "Actions",
    "deleteTeamButtonTooltip": "Delete team",
    "editTeamButtonTooltip": "Edit team",
    "modal": {
      "title": "Remove",
      "desc": "Are you sure you want to remove the team from the project?",
      "confirmButton": "Remove"
    },
    "team": {
      "add": "Add team",
      "name": "Team name",
      "members": {
        "plural": "Members",
        "singular": "Member",
        "genetive": "Member",
        "genetivePlural": "Member",
      },
      "repos": {
        "singular": "Repository",
        "plural": "Repository",
        "genetive": "Repository",
        "genetivePlural": "Repositories",
      },
    }
  },

  "ru-RU": {
    "actionsHeader": "Действия",
    "deleteTeamButtonTooltip": "Удалить команду",
    "editTeamButtonTooltip": "Редактировать команду",
    "modal": {
      "title": "Удаление",
      "desc": "Вы уверены, что хотите удалить команду из проекта?",
      "confirmButton": "Удалить"
    },
    "team": {
      "add": "Добавить команду",
      "name": "Название команды",
      "members": {
        "plural": "Участники",
        "singular": "Участник",
        "genetive": "Участника",
        "genetivePlural": "Участников",
      },
      "repos": {
        "singular": "Репозиторий",
        "plural": "Репозитории",
        "genetive": "Репозитория",
        "genetivePlural": "Репозиториев",
      },
    }
  }
}
</i18n>
