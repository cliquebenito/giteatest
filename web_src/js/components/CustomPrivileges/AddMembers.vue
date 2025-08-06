<template>
    <aside-dialog
        :show="show"
        :title="t('title')"
        @cancel="cancel"
    >
        <div v-if="users.length">
            <div class="aside-panel-content__search">
                <search-field @filterChange="handleFilterChange($event)"></search-field>
            </div>
            <div class="aside-panel-list">
                <div v-if="filteredUsers.length > 0">
                <member-card 
                    @selectMember="handleSelectMember($event)"
                    :isSelected="checkIsSelected(user)"
                    :isLoading="isLoading"
                    :add-mode="true" 
                    :data="user" 
                    :key="user.ID" 
                    v-for="(user) in filteredUsers">
                </member-card>
                </div>
                <div v-else>
                  {{`${t('noMembersByFilter')}: "${filter}"`}}
                </div>
            </div>
        </div>
        <div v-else-if="isAllUsersInTeam" class="empty-users-list">
            <svg-icon name="no-members"></svg-icon>
            {{ t('allMembersAdded') }}
        </div>
        <div v-else class="empty-users-list">
            <svg-icon name="no-members"></svg-icon>
            {{ t('noMembers') }}
        </div>

        <template #footer>
            <div class="aside-controls-group">
              <base-button :disabled="isLoading || !selectedMembers.length" type="primary" :loading="isLoading" @click="handleSubmit">{{ t('addButton') }}</base-button>
              <base-button :disabled="isLoading" @click="cancel">{{ t('cancelButton') }}</base-button>
            </div>
        </template>
    </aside-dialog>
</template>


<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';
import MemberCard from './MemberCard.vue';
import SearchField from './SearchField.vue';

import AsideDialog from '../../ui/AsideDialog.vue';
import BaseButton from '../../ui/BaseButton.vue';



const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const props = defineProps({
  show: {
    type: Boolean
  }, 
  
  isLoading: {
    type: Boolean,
  },

  users: {
      type: Array,
      required: true
  },
  isAllUsersInTeam: {
      type: Boolean
  }
});

const selectedMembers = ref([]);
const filter = ref('');


const emit = defineEmits(['toggleMembersPanel', 'submitMembers']);

const filteredUsers = computed(() => {
  return props.users.filter(item => {
    const searchValue = `${item.FullName} ${item.LowerName}`.toLowerCase();
    return searchValue.includes(filter.value.toLowerCase());
  });
});

const handleFilterChange = (value) => {
  filter.value = value;
};

const handleSubmit = () => {
  emit('submitMembers', [ ...selectedMembers.value]);
};

const cancel = () => {
  filter.value = '';
  emit('toggleMembersPanel', false);
};

const handleSelectMember = (data) => {
  const { action, id } = data;
  if (action === 'add') {
    selectedMembers.value = [ ...selectedMembers.value, Number(id)];
  } else if (action === 'remove') {
    selectedMembers.value = selectedMembers.value.filter(item => item !== Number(id));
  }
};

const checkIsSelected = (member) => {
  return selectedMembers.value.includes(member.ID);
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
  .aside-controls-group {
    display: flex;
    align-items: center;
    column-gap: 16px;
    justify-content: space-between;
  }
  .aside-controls-group button {
    min-width: 180px;
  }

  .empty-users-list {
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    row-gap: 12px;
  }

  .empty-users-list svg {
    width: 120px;
    height: 120px;
  }
</style>

<i18n>
 {
  "en-US": {
    "addButton": "Add",
    "cancelButton": "Cancel",
    "title": "Add member",
    "noMembers": "No members",
    "allMembersAdded": "All users already added",
    "noMembersByFilter": "No members by filter"
  },

  "ru-RU": {
    "addButton": "Добавить",
    "cancelButton": "Отменить",
    "noMembers": "Нет участников для добавления в команду",
    "allMembersAdded": "Все пользователи добавлены",
    "noMembersByFilter": "Нет участников по фильтру",
    "title": "Добавить участника"
  }
}  
</i18n>