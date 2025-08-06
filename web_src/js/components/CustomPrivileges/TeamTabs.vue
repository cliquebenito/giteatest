<template>
    <div class="tabs">
        <div class="tabs-nav">
            <button class="tab-item" :class="activeTab === MEMBERS ? 'active' : ''" @click="handleToggleTab(MEMBERS)">{{ t('members') }}</button>
            <button class="tab-item" :class="activeTab === REPOSITORIES ? 'active' : ''" @click="handleToggleTab(REPOSITORIES)">{{ t('repos') }}</button>
        </div>
        <div v-if="activeTab === MEMBERS" class="tabs-content">
            <div class="content-wrap" v-if="members.length">
                <div class="search-wrap">
                    <search-field @filterChange="handleMembersFilterChange($event)"></search-field>
                    <base-button type="base" icon class="button-show-panel" @click="handleShowMembersPanel">
                        <svg-icon name="octicon-plus"></svg-icon>
                    </base-button>
                </div>
                <div v-if="filteredMembers.length" class="tabs-scroll-wrap">
                    <member-card
                        @removeMember="handleRemoveMember($event)"
                        :add-mode="false"
                        :data="member"
                        :key="member.ID"
                        v-for="member in filteredMembers">
                    </member-card>
                </div>
                <div v-else class="empty-data">
                    {{`${t('noMembersByFilter')}: "${filterMembers}" `}}
                </div>
            </div>
            <div class="empty-data-wrap" v-else>
                <div v-if="mode === 'edit'" class="empty-data">
                    <svg-icon class="svg-card" name="no-members" />
                    {{t('noMembers')}}
                    <base-button type="primary" @click="handleShowMembersPanel" :disabled="isLoading">
                      <svg-icon name="octicon-plus" />
                      {{ t('addMembersButton') }}
                    </base-button>
                </div>
                <div v-else class="tabs-action-hint">
                    <svg-icon class="svg-card" name="no-members" />
                    {{t('addMembersHint')}}
                </div>
            </div>
        </div>
        <div v-else-if="activeTab === REPOSITORIES" class="tabs-content">
            <div class="content-wrap" v-if="repositories.length">
                <div class="search-wrap">
                    <search-field @filterChange="handleRepositoryFilterChange($event)"></search-field>
                </div>
                <div v-if="filteredRepositories.length" class="tabs-scroll-wrap">
                    <repository-card
                        :repository="repository"
                        :key="repository.ID"
                        v-for="repository in filteredRepositories">
                    </repository-card>
                </div>
                <div v-else class="empty-data">
                    {{`${t('noReposByFilter')}: "${filterRepositories}" `}}
                </div>
            </div>
            <div class="empty-data-wrap" v-else>
                <div v-if="mode === 'edit'" class="empty-data">
                    <svg-icon class="svg-card" name="no-members"></svg-icon>
                    {{t('noRepos')}}
                </div>
                <div v-else class="tabs-action-hint">
                    <svg-icon class="svg-card" name="no-members"></svg-icon>
                    {{t('addReposHint')}}
                </div>
            </div>
        </div>
    </div>
    <confirm-modal
        @modalConfirm="confirmRemoveMember"
        @modalCancel="cancelRemoveMember"
        :isLoading="isLoading"
        :show="showDeleteMemberModal"
        :title="t('modal.title')"
        :description="t('modal.desc')"
        :confirmButtonText="t('modal.buttonConfirm')">
    </confirm-modal>
</template>

<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';
import SearchField from './SearchField.vue'
import MemberCard from './MemberCard.vue';
import RepositoryCard from './RepositoryCard.vue';

import ConfirmModal from '../../ui/ConfirmModal.vue';
import BaseButton from '../../ui/BaseButton.vue';
import {useDynamicUrl} from "../useDynamicUrl.js";

const { csrfToken, appSubUrl} = window.config;
const { orgLink, teamName }  = window.config.pageData;

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const props = defineProps({
    isLoading: {
        type: Boolean
    },
    members: {
        type: Array,
        required: true
    },
    repositories: {
        type: Array,
        required: true
    }
});

const { mode } = window.config.pageData;

const emit = defineEmits(['toggleMembersPanel', 'submitMembers']);
const REPOSITORIES = 'respositories';

const MEMBERS = 'members';

const activeTab = ref(MEMBERS);
const filterRepositories = ref('');

const filterMembers = ref('');

const removedMember = ref(null);

const showDeleteMemberModal = ref(false);

const filteredRepositories = computed(() => {
    return props.repositories.filter(item => {
        return item.Name.includes(filterRepositories.value)
    });
});

const filteredMembers = computed(() => {
    return props.members.filter(item => {
        return item.Name.includes(filterMembers.value)
    });
});
const handleRepositoryFilterChange = (value) => {
  filterRepositories.value = value;

};
const handleMembersFilterChange = (value) => {
  filterMembers.value = value;

};
const handleToggleTab = (tab) => {
  activeTab.value = tab;

}
const handleShowMembersPanel = () => {
  emit('toggleMembersPanel', true);


};
const handleRemoveMember = (data) => {
  showDeleteMemberModal.value = true;
  removedMember.value = data;

};

const { getFullUrl } = useDynamicUrl({
  link: orgLink,
  appSubUrl: appSubUrl,
});

const confirmRemoveMember = () => {
 const { team: { LowerName: teamLowerName } } = window.config.pageData

    const URL = getFullUrl(`teams/${teamLowerName}/action/remove`);

    const formData = new URLSearchParams();
    formData.append('_csrf', csrfToken);
    formData.append('uid', removedMember.value.ID);
    formData.append('TeamName', teamName);

    fetch(URL, {
        method: 'post',
        headers:{
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: formData.toString()
    }).then((response) => {
        cancelRemoveMember();
        location.reload();
        console.log(response)
    }).catch(err => {
        console.log(err)
    });
};

const cancelRemoveMember = () => {
    removedMember.value = null;
    showDeleteMemberModal.value = false;
};


</script>

<style scoped>
.tabs {
    height: calc(100vh - 220px);
    position: sticky;
    top: 24px;
}
.tabs-nav {
    display: flex;
    background-color: #F3F5F6;
    border: 1px solid #D3DBDF;
    border-radius: 8px;
    padding: 4px;
    margin-bottom: 16px;
    column-gap: 2px;
}

.tab-item {
    height: 32px;
    width: 197px;
    font-size: 15px;
    border-radius: 4px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
}
.tab-item.active {
    background-color: #78909C;
    color: #fff;
}
.tabs-content {
    height: 100%;
}
.tabs-scroll-wrap {
    max-height: 680px;
    overflow-y: auto;
    margin-top: 16px;
}

.search-wrap {
    display: flex;
    align-items: center;
    column-gap: 16px;
}
.button-show-panel {
    flex-shrink: 0;
}
.empty-data-wrap {
    height: 100%;
}
.tabs-action-hint,
.empty-data {
    display: flex;
    height: 100%;
    flex-direction: column;
    align-items: center;
    row-gap: 16px;
    font-size: 15px;
    color: #263238;
    justify-content: center;
    white-space: pre-line;
    text-align: center;
}

.tabs-action-hint .svg-card,
.empty-data .svg-card {
    width: 112px;
    height: 80px;
}
.content-wrap {
    height: 100%;
}
.tabs-action-hint {
    text-align: center;
}

</style>

<i18n>
 {
   "en-US": {
    "addMember": "Add member",
    "members": "Members",
    "repos": "Repositories",
    "noMembers": "No one members in team",
    "noMembersByFilter": "No members by filter",
    "noRepos": "No one repositories in team",
    "noReposByFilter": "No repositories by filter",
    "addReposHint": "You can add repositories after create team",
    "addMembersHint": "You can add members after create team",
    "addMembersButton": "Add member",
    "modal": {
        "title": "Remove",
        "desc": "Are you sure you want to delete the user?",
        "buttonConfirm": "Remove"
    }
   },

   "ru-RU": {
     "addMember": "Добавить участника",
     "members": "Участники",
     "repos": "Репозитории",
     "noMembers": "В этой команде пока нет\n ни одного участника",
     "noMembersByFilter": "Нет участников по фильтру",
     "noRepos": "В этой команде пока нет ни одного репозитория",
     "noReposByFilter": "Нет репозиториев по фильтру",
     "addReposHint": "Вы можете добавить репозитории после создания команды",
     "addMembersHint": "Вы можете добавить участников после создания команды",
     "addMembersButton": "Добавить участника",
     "modal": {
        "title": "Удаление",
        "desc": "Вы уверены, что хотите удалить пользователя?",
        "buttonConfirm": "Удалить"
     }
    }
 }

</i18n>
