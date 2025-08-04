<template>
  <div>
    <EmptyTeams v-if="teams.length <= 0" />
    <div v-else>
      <div class="teams-table-header">
        <base-input
          class="teams-table-header__search"
          type="text"
          icon="octicon-search"
          v-model="searchInput"
          :placeholder="t('searchPlaceholder')"
        />
        <base-button
          class="teams-table-header__button"
          type="primary"
          :href="`${baseLink}/new`"
        >
          <svg-icon name="octicon-plus" />
          {{ t("buttonAdd") }}
        </base-button>
      </div>
      <TeamsTable :teams="filteredTeams" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';

import EmptyTeams from './EmptyTeams.vue';
import TeamsTable from './TeamsTable.vue';

import BaseButton from '../../ui/BaseButton.vue';
import BaseInput from '../../ui/BaseInput.vue';


const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const { baseLink } = window.config.pageData;
const { pageData } = window.config;
const { teams: teamsData } = pageData;

const teams = ref(teamsData || []);
const searchInput = ref("");

const filteredTeams = computed(() => {
  if (!searchInput.value) return teams.value;

  const query = searchInput.value.toLowerCase();

  return teams.value.filter(team => {
    if (team.Name?.toLowerCase().includes(query)) return true;
    if (team.LowerName?.toLowerCase().includes(query)) return true;

    return false;
  });
});
</script>


<style scoped>
.teams-table-header {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
  column-gap: 24px;
}

.teams-table-header__search {
  flex-grow: 1;
}

.teams-table-header__button {
  flex-shrink: 0;
  width: 186px;
  white-space: nowrap;
}

.teams-table-header__button svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
</style>

<i18n>
  {
   "en-US": {
    "searchPlaceholder": "Search...",
     "buttonAdd": "Add team"
   },

   "ru-RU": {
    "searchPlaceholder": "Поиск...",
     "buttonAdd": "Добавить команду"
   }
 }
 </i18n>
