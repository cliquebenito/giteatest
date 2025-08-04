<template>
    <div class="wrap">
        <div v-if="errorMessage" class="ui negative message flash-message flash-error">
		    <p>{{ errorMessage }}</p>
	    </div>
        <div class="columns-container">
            <div class="column__left">
                <form class="form" @submit.prevent="handleSubmitForm">
                    <div class="form__header">
                        <base-title variant="h3">{{ t('formTitle')}}</base-title>
                        <base-input
                            :label="t('fieldName')"
                            :placeholder="t('fieldNamePlaceholder')"
                            :disabled="isLoading"
                            :errorMessage="teamNameErrorMessage"
                            :required="true"
                            class="form-field"
                            v-model="teamModel.team_name">
                        </base-input>
                        <base-input
                            :label="t('fieldDesc')"
                            :placeholder="t('fieldDescPlaceholder')"
                            :errorMessage="teamDescErrorMessage"
                            :disabled="isLoading"
                            class="form-field"
                            v-model="teamModel.description">
                        </base-input>
                    </div>

                    <div class="form__body">
                        <base-title variant="h3">{{ t('formSubTitle')}}</base-title>
                        <p class="form__hint">{{ t('formHint')}}</p>

                        <team-unit-fields
                            v-for="(privileges, index) in teamModel.custom_privileges"
                            @deleteUnit="handleDeleteUnit($event)"
                            @updateUnits="handleUpdateUnits($event)"
                            :totalUnitsCount="teamModel.custom_privileges.length"
                            :isLoading="isLoading"
                            :customPrivileges="privileges"
                            :repositories="allRepositories"
                            :index="index"
                            :key="privileges.id"
                        />

                    </div>

                    <div class="form__footer">
                        <base-button :disabled="isLoading" type="transparent" class="add-rule" @click.prevent="handleAddPrivileges">
                            <svg-icon name="octicon-plus" />
                            {{ t('addRuleButton')}}
                        </base-button>
                        <div class="buttons-row">
                            <base-button type="primary" :loading="isLoading" :disabled="disableSubmit || isLoading || !!teamNameErrorMessage.length || !!teamDescErrorMessage.length">
                                {{ t('buttonSave')}}
                            </base-button>
                            <base-button v-if="mode === 'new'" :disabled="isLoading" @click.prevent="handleCancel">
                                {{ t('buttonCancel')}}
                            </base-button>
                            <base-button :disabled="isLoading" v-else-if="mode === 'edit'" @click.prevent="handleDeleteTeam">
                                {{ t('deleteTeam')}}
                            </base-button>
                        </div>
                    </div>
                </form>
            </div>
            <div class="column__right">
                <team-tabs
                    @toggleMembersPanel="toggleMembersPanel"
                    @submitMembers="handleSumbitMembers($event)"
                    :isLoading="isLoading"
                    :members="teamMembers"
                    :repositories="teamRepositories">
                </team-tabs>
            </div>
        </div>
        <!-- <team-members-panel
            v-if="panelMembersVisible"
            :allUsers="allUsers"
            :members="teamMembers"
            :isLoading="isLoading"
            @toggleMembersPanel="toggleMembersPanel"
            @submitMembers="handleSumbitMembers($event)">
        </team-members-panel> -->
    </div>
    <confirm-modal
        @modalCancel="handleCancelDeleteTeam"
        @modalConfirm="handleConfirmDeleteTeam"
        :isLoading="isLoading"
        :show="showDeleteTeamModal"
        :title="t('modal.title')"
        :description="t('modal.desc')"
        :confirmButtonText="t('modal.buttonConfirm')">
    </confirm-modal>

    <!-- <aside-dialog
        @cancel="handleCancelAsideDialog($event)"
        :show="visible"
        title="some text"
    >
        <team-add-members
            v-if="panelContent === 'members'"
            :allUsers="allUsers"
            :members="teamMembers"
            :isLoading="isLoading"
            @submitMembers="handleSumbitMembers($event)">
        </team-add-members>
        <div v-else-if="panelContent === 'repositories'">
            repositories
        </div>
        <div v-else>
            no content
        </div>
    </aside-dialog>

    <button @click="onClick">click me</button> -->

    <add-members
        :show="panelMembersVisible"
        :users="noneTeamUsers"
        :isLoading="isLoading"
        :isAllUsersInTeam="isAllUsersInTeam"
        @toggleMembersPanel="toggleMembersPanel"
        @submitMembers="handleSumbitMembers($event)"
    ></add-members>

</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { v4 as uuidv4 } from 'uuid';
import { SvgIcon } from '../../svg';

import ConfirmModal from '../../ui/ConfirmModal.vue';
import BaseInput from '../../ui/BaseInput.vue';
import BaseTitle from '../../ui/BaseTitle.vue';
import BaseButton from '../../ui/BaseButton.vue';

import TeamUnitFields from './TeamUnitFields.vue';
import TeamTabs from './TeamTabs.vue';

import AddMembers from './AddMembers.vue';

import { getSafetyUrl, compareArrays } from '../../utils';
import {useDynamicUrl} from "../useDynamicUrl.js";


const teamNameErrorMessage = ref('');
const teamDescErrorMessage = ref('');
const errorMessage = ref('')

const { csrfToken, appSubUrl} = window.config;
const { orgLink, teamName, mode, team = {}, customPrivilegesUnits = [] } = window.config.pageData;

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const isLoading = ref(false);
const allRepositories = ref([]);
const allUsers = ref([]);
const isAllUsersInTeam = ref(false);
const abortController  = new AbortController();
const noneTeamUsers = ref([]);
const teamMembers = ref(mode === 'new' ? [] : (team.Members || []));
const teamRepositories = ref(mode === 'new' ? [] : (team.Repos || []));
const ALL_SELECT_VALUE = '@ALL';

const panelMembersVisible = ref(false);
const showDeleteTeamModal = ref(false);

const areAllUnitsValid = computed(() => {
  return teamModel.custom_privileges.every(unit => unit.privileges?.length > 0);
});

const emptyCustomPrivileges = {
    all_repositories: true,
    repo_id: null,
    privileges: []
};

const teamModel = reactive({
    team_name: mode === 'edit' ? team.Name : '',
    description: mode === 'edit' ? team.Description : '',
    user_ids: [],
    custom_privileges: mode === 'new' ? [{
        id: uuidv4(),
        all_repositories: true,
        repo_id: null,
        privileges: [1, 2, 3, 4, 5]
    }] : normalizeCustomPrivilegesUnits(customPrivilegesUnits)
});

onMounted(() => {
    fetchAllRepos();
    fetchAllUsers();
    window.addEventListener('beforeunload', () => {
        abortController.abort();
    });
});

onBeforeUnmount(() => {
    console.log('unmount');
    abortController.abort();
});

watch(() => teamModel.team_name, (value) => {
    if (!isValidTeamName(value) && value.length) {
        teamNameErrorMessage.value = t('teamNameErrorMessage');
    } else {
        teamNameErrorMessage.value = '';
    }
});


watch(() => teamModel.description, (value) => {
    if (!isValidTeamDesc(value) && value.length) {
        teamDescErrorMessage.value = t('teamDescErrorMessage');
    } else {
        teamDescErrorMessage.value = '';
    }
})

watch([allUsers, teamMembers], ([allUsersValue, teamMembersValue]) => {
    const teamMembersIds = teamMembersValue.map(user => user.ID);
    noneTeamUsers.value = allUsersValue.filter(user => !teamMembersIds.includes(user.ID));
    if (mode === 'new') {
        isAllUsersInTeam.value = false;
    } else {
        const allUsersIds = allUsersValue.map(item => item.ID);
        isAllUsersInTeam.value = compareArrays(allUsersIds, teamMembersIds);
    }
});

const { getFullUrl } = useDynamicUrl({
  link: orgLink,
  appSubUrl: appSubUrl,
});

const handleSubmitForm = () => {
    const endpoint = mode === 'new' ? 'new' : `${teamName}/edit`
    const URL = getFullUrl(`/teams/${endpoint}`);

    const team = { ...teamModel};
    team.custom_privileges = [ ...team.custom_privileges].map(item => {
        const { id, ...rest}  = item;
        return rest;
    })

    isLoading.value = true;

  fetch(URL, {
    method: 'post',
    body: JSON.stringify({ ...team }),
    headers: {
      'Content-Type': 'application/json',
      'X-Csrf-Token': csrfToken
    },
  })
    .then((res) => {
      if (res.status === 303) {
        window.location.href = getSafetyUrl(res.headers.get('Location'));
      }
      else if (res.ok) {
        window.location.href = getFullUrl(`/teams/${team.team_name}`);
      }
      else {
        return res.json().then(error => {
          throw new Error(error.message || 'Ошибка сервера');
        });
      }
    })
    .catch(error => {
      errorMessage.value = error.message;
    })
    .finally(() => {
      isLoading.value = false;
    });
}

const handleUpdateUnits = (event) => {
  const { index, value, field } = event;
  if (field === 'repository') {
    teamModel.custom_privileges[index].all_repositories = value === ALL_SELECT_VALUE;
    teamModel.custom_privileges[index].repo_id = value === ALL_SELECT_VALUE ? null : Number(value);
  } else if (field === 'privileges') {
    teamModel.custom_privileges[index].privileges = value.map(Number);
  } else {
    teamModel.custom_privileges[index][field] = value;
  }
};

const handleAddPrivileges = () => {
    const customPrivileges = teamModel.custom_privileges;
    teamModel.custom_privileges = [ ...customPrivileges, { ...emptyCustomPrivileges, id: uuidv4()}]
};

const handleCancel = () => {
    window.location.href = getSafetyUrl(getFullUrl(`/teams`));
};

const handleDeleteTeam = () => {
    showDeleteTeamModal.value = true;
};

const handleConfirmDeleteTeam = () => {
    const URL = getSafetyUrl(getFullUrl(`/teams/${teamName}/delete`));

    const formData = new FormData();
    formData.append('_csrf', csrfToken);
    isLoading.value = true;
    fetch(URL, {
        method: 'post',
        body: formData,
    }).then((response) => {
        window.location.href = getSafetyUrl(getFullUrl(`/teams`));
        isLoading.value = false;
    }).catch(err => {
        console.warn('handleDeleteTeam ERROR', err);
        isLoading.value = false;
    });
};

const handleCancelDeleteTeam = () => {
    showDeleteTeamModal.value = false;
}

const handleDeleteUnit = (id) => {
    const privileges = [ ...teamModel.custom_privileges]
    teamModel.custom_privileges = privileges.filter(item => item.id !== id);
}

const handleSumbitMembers = (data) => {
    teamModel.user_ids = data.map(item => Number(item));
    handleSubmitForm();
};

const toggleMembersPanel = (state) => {
    panelMembersVisible.value = state;
};

const disableSubmit = computed(() => {
  return !teamModel.team_name.trim() ||
    teamModel.custom_privileges.length === 0 ||
    !areAllUnitsValid.value;
});

const fetchAllRepos = () => {
    const URL = getFullUrl(`/all/repositories`);

    isLoading.value = true;

    fetch(URL)
        .then(response => response.json())
        .then(response => {
            isLoading.value = false;
            allRepositories.value = (response || []).sort((a, b) =>
              a.Name.localeCompare(b.Name, undefined, { sensitivity: 'base' })
        )})
        .catch(err => {
            isLoading.value = false;
            console.warn('fetchAllRepos ERROR', err);
        })
};

const fetchAllUsers = () => {
    const searchParams = mode === 'new' ? '' : `?team=${team.LowerName}`;
    const URL = getFullUrl(`/all/users`);

    fetch(URL)
        .then(response => response.json())
        .then(response => {
            allUsers.value = response || [];
        }).catch(err => {
            console.warn('fetchAllUsers ERROR', err)
        })
}

function normalizeCustomPrivilegesUnits(data) {
  if (!data) return [];
  return data.map((item) => {
    return {
      id: item.ID,
      all_repositories: item.AllRepositories,
      repo_id: item.RepositoryID || null,
      privileges: !item.CustomPrivileges ? [] : item.CustomPrivileges.split(',').map(val => Number(val))
    }
  });
}


function isValidTeamName(value) {
    const regex = /^[a-zA-Z0-9]((?![_\-\.]{2})[a-zA-Z0-9._-])*$/;
    return regex.test(value) && value.length <= 30;
}

function isValidTeamDesc(value) {
    return value.length <= 255;
}

</script>

<style scoped>
    .form-field {
        margin-bottom: 24px;
    }
    .wrap {
        padding-bottom: 24px;
    }
    .columns-container {
        display: flex;
        flex-direction: row;
        column-gap: 32px;
    }
    .column__left {
        flex-grow: 1;
        width: 100%;
    }
    .column__right {
        flex-shrink: 0;
        width: 404px;
    }
    .form {
        display: flex;
        flex-direction: column;
    }
    .form__title {
        font-weight: 600;
        color: #263238;
        font-size: 28px;
        margin-bottom: 16px;
    }
    .form__header {
        margin-bottom: 12px;
    }
    .form__hint {
        margin-bottom: 16px;
    }
    .add-rule {
        margin-bottom: 32px;
    }
    .add-rule svg {
        margin-right: 8px;
        color: #263238;
        width: 16px;
        height: 16px;
    }
    .add-rule:disabled {
        opacity: .7;
        cursor: not-allowed;
    }
    .buttons-row {
        display: flex;
        column-gap: 16px;
    }
</style>

<i18n>
{
  "en-US": {
    "teamNameErrorMessage": "The team name can contain no more than 30 characters. Latin letters, numbers, dashes, dots, underscores are allowed.",
    "teamDescErrorMessage": "The length of the text should not more then 255 characters",
    "buttonSave": "Save",
    "buttonCancel": "Cancel",
    "formTitle": "Team info",
    "formSubTitle": "Access to repositories",
    "formHint": "Set up custom team privileges to access repositories",
    "fieldName": "Title",
    "fieldNamePlaceholder": "Enter title",
    "fieldDesc": "Description",
    "fieldDescPlaceholder": "Enter description",
    "deleteTeam": "Remove Team",
    "addRuleButton": "Add privilege",
    "modal": {
        "title": "Remove",
        "desc": "Are you sure you want to remove the team from the project?",
        "buttonConfirm": "Remove"
    }
  },

  "ru-RU": {
    "teamNameErrorMessage": "Название команды может содержать не более 30 символов. Допускаются латинские буквы, цифры, тире, точки, знаки подчеркивания.",
    "teamDescErrorMessage": "Длина текста не должна превышать 255 символов",
    "buttonSave": "Сохранить",
    "buttonCancel": "Отменить",
    "formTitle": "Данные команды",
    "formSubTitle": "Доступ к репозиториям",
    "formHint": "Настройте кастомные привилегии команды для доступа к репозиториям",
    "fieldName": "Название",
    "fieldNamePlaceholder": "Введите название",
    "fieldDesc": "Описание",
    "fieldDescPlaceholder": "Введите описание",
    "deleteTeam": "Удалить команду",
    "addRuleButton": "Добавить привилегию",
    "modal": {
        "title": "Удаление",
        "desc": "Вы уверены, что хотите удалить команду из проекта?",
        "buttonConfirm": "Удалить"
    }
  }
}
</i18n>
