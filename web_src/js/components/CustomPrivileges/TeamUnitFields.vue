<template>
  <div class="form__container">
    <div class="form__column">
      <base-select
        :label="t('repositoriesLabel')"
        :disabled="isDisabled()"
        :options="optionsRepository"
        name="repository-name"
        v-model="selectModelRepositories"
        @update:modelValue="handleRepositoriesSelectChange($event)"
      ></base-select>
      <div v-show="!isCollapsed" class="checkbox-group">
        <base-checkbox
          v-model="units"
          name="view-pr"
          :value="UNITS_MAP.viewBranch"
          :disabled="isDisabled() || isIncludePriorityUnit(units, [UNITS_MAP.createPR, UNITS_MAP.approvePR, UNITS_MAP.mergePR, UNITS_MAP.changeBranch])"
          :label="t('prView')"
          @change="handlePrivilegeUnitsCheckboxChange($event)">
        </base-checkbox>
        <base-checkbox
          v-model="units"
          name="view-pr"
          :value="UNITS_MAP.changeBranch"
          :disabled="isDisabled() || units.includes(UNITS_MAP.mergePR)"
          :label="t('prWrite')"
          @change="handlePrivilegeUnitsCheckboxChange($event)">
        </base-checkbox>
        <base-checkbox class="invisible">
        </base-checkbox>
        <base-checkbox
          v-model="units"
          name="create-pr"
          :value="UNITS_MAP.createPR"
          :disabled="isDisabled()"
          :label="t('prCreate')"
          @change="handlePrivilegeUnitsCheckboxChange($event)">
        </base-checkbox>
        <base-checkbox
          v-model="units"
          name="approve-pr"
          :value="UNITS_MAP.approvePR"
          :disabled="isDisabled()"
          :label="t('prApprove')"
          @change="handlePrivilegeUnitsCheckboxChange($event)">
        </base-checkbox>
        <base-checkbox
          v-model="units"
          name="merge-pr"
          :value="UNITS_MAP.mergePR"
          :disabled="isDisabled()"
          :label="t('prMerge')"
          @change="handlePrivilegeUnitsCheckboxChange($event)">
        </base-checkbox>
      </div>
    </div>
    <div class="form__controls-group">
      <base-button
        :disabled="isDisabled() || totalUnitsCount === 1"
        :data-tooltip-content="t('deleteUnitTooltip')"
        type="transparent"
        icon
        @click.prevent="handleDelete"
      >
        <svg-icon name="octicon-trash"></svg-icon>
      </base-button>
      <base-button
        :data-tooltip-content="isCollapsed ? t('explandUnitTooltip') : t('collapseUnitTooltip')"
        type="transparent"
        icon
        @click.prevent="handleCollapse"
      >
        <svg-icon :name="isCollapsed ? 'octicon-chevron-up' : 'octicon-chevron-down'"></svg-icon>
      </base-button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';
import BaseCheckbox from '../../ui/BaseCheckbox.vue';
import BaseButton from '../../ui/BaseButton.vue';
import BaseSelect from '../../ui/BaseSelect.vue';

const props = defineProps({
  isLoading: {
    type: Boolean
  },
  totalUnitsCount: {
    type: Number,
    required: true
  },
  customPrivileges: {
    type: Object,
    required: true
  },
  repositories: {
    type: Array,
    required: true,
    default: () => {
      return [];
    }
  },
  index: {
    type: Number,
    required: true
  }
});

const { t } = useI18n({
  inheritLocale: true,
  useScope: 'local'
});

const UNITS_MAP = {
  viewBranch: 1,
  changeBranch: 2,
  createPR: 3,
  approvePR: 4,
  mergePR: 5
}

const emit = defineEmits(['updateUnits', 'deleteUnit']);

const units = ref(props.customPrivileges.privileges);
const isCollapsed = ref(false);
const ALL_VALUE = '@ALL';

const ALL_REPOSITORIES_OPTION = { value: ALL_VALUE, label: t('allRepositoriesOption') };
const selectModelRepositories = ref(ALL_REPOSITORIES_OPTION);
const optionsRepository = computed(() => {
  return [ALL_REPOSITORIES_OPTION].concat(props.repositories.map(item => ({ value: item.ID, label: item.Name })))
});

watch(() => props.repositories, (repositories) => {
  const repoId = props.customPrivileges.repo_id;

  if (repositories.length && repoId) {
    selectModelRepositories.value = getRepositoryOptionById(repoId, repositories);
  }
});

function handlePrivilegeUnitsCheckboxChange(event) {
  const checked = event.target.checked;
  const value = Number(event.target.value);
  let newUnits = [...units.value];

  if (checked) {
    if (!newUnits.includes(value)) {
      newUnits.push(value);
    }

    if (value === UNITS_MAP.mergePR) {
      if (!newUnits.includes(UNITS_MAP.viewBranch)) newUnits.push(UNITS_MAP.viewBranch);
      if (!newUnits.includes(UNITS_MAP.changeBranch)) newUnits.push(UNITS_MAP.changeBranch);
    }
    else if (value === UNITS_MAP.changeBranch) {
      if (!newUnits.includes(UNITS_MAP.viewBranch)) newUnits.push(UNITS_MAP.viewBranch);
    }
    else if ([UNITS_MAP.createPR, UNITS_MAP.approvePR].includes(value)) {
      if (!newUnits.includes(UNITS_MAP.viewBranch)) newUnits.push(UNITS_MAP.viewBranch);
    }
  } else {
    newUnits = newUnits.filter(unit => unit !== value);
  }

  units.value = [...new Set(newUnits)];

  emit('updateUnits', {
    index: props.index,
    value: units.value,
    field: 'privileges'
  });
}

function handleRepositoriesSelectChange(event) {
  const repoId = event.value;
  const payload = {
    index: props.index,
    value: repoId,
    field: 'repository'
  };
  emit('updateUnits', payload);
}

function handleDelete() {
  emit('deleteUnit', props.customPrivileges.id);
}

function handleCollapse() {
  isCollapsed.value = !isCollapsed.value;
}

function getRepositoryOptionById(id, repositories) {
  const repo = repositories.find(item => item.ID === id);
  if (repo) {
    return { value: repo.ID, label: repo.Name }
  }
  console.warn(`Can't find repo with id = ${id}, in list:`, repositories);
  return null
}

function isIncludePriorityUnit(sourceUnits, priorityUnits) {
  return sourceUnits.some(item => priorityUnits.includes(item))
}

function isDisabled() {
  return props.isLoading
}
</script>

<style scoped>
.form__container {
  display: flex;
  column-gap: 32px;
  border-bottom: 1px solid #D3DBDF;
  margin-bottom: 16px;
}
.form__column {
  width: 100%;
  padding-bottom: 24px;
}
.checkbox-group {
  height: 108px;
  display: flex;
  flex-direction: column;
  flex-wrap: wrap;
  row-gap: 16px;
  column-gap: 32px;
  margin-top: 24px;
}
.form__controls-group  {
  padding-top: 25px;
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  align-self: flex-start;
}
</style>

<i18n>
{
  "en-US": {
    "prView": "Read",
    "prWrite": "Write",
    "prCreate": "Create PR",
    "prApprove": "Approve PR",
    "prMerge": "Merge PR",
    "allRepositoriesOption": "All repositories",
    "repositoriesLabel": "Repositories",
    "deleteUnitTooltip": "Delete rule",
    "explandUnitTooltip": "Expand block",
    "collapseUnitTooltip": "Collapse block"
  },
  "ru-RU": {
    "prView": "Чтение",
    "prWrite": "Запись",
    "prCreate": "Создание PR",
    "prApprove": "Одобрение PR",
    "prMerge": "Слияние PR",
    "allRepositoriesOption": "Все репозитории",
    "repositoriesLabel": "Репозитории",
    "deleteUnitTooltip": "Удалить правило",
    "explandUnitTooltip": "Раскрыть блок",
    "collapseUnitTooltip": "Свернуть блок"
  }
}
</i18n>
