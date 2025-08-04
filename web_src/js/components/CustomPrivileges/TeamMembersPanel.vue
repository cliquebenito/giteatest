<template>
    <div class="aside-panel">
        <div class="aside-panel-content">
          <div class="aside-panel__header">
            <h2 class="aside-panel__title">{{ t('title') }}</h2>
            <button class="aside-panel__close" :disabled="isLoading" @click="cancel"><svg-icon name="octicon-x"></svg-icon></button>
          </div>
          
          <div v-if="allUsers.length">
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
          <div v-else>
            {{ t('noMembers') }}
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
import MemberCard from './MemberCard.vue';
import SearchField from './SearchField.vue';



const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const props = defineProps({
  isLoading: {
    type: Boolean,
  },
  allUsers: {
      type: Array,
      required: true
  },
  members: {
      type: Array,
      required: true
  }
});

const selectedMembers = ref(props.members.map(item => item.ID));
const filter = ref('');


const emit = defineEmits(['toggleMembersPanel', 'submitMembers']);

const filteredUsers = computed(() => {
  return props.allUsers.filter(item => {
    const searchValue = `${item.FullName} ${item.LowerName}`;
    return searchValue.includes(filter.value);
  });
});

const handleFilterChange = (value) => {
  filter.value = value;
};

const handleSubmit = () => {
  emit('submitMembers', [ ...selectedMembers.value]);
};

const cancel = () => {
  emit('toggleMembersPanel', false);
};

const handleSelectMember = (data) => {
  const { action, id } = data;
  console.log('handleSelectMember', data);
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
    "title": "Add member",
    "noMembers": "No members",
    "noMembersByFilter": "No members by filter"
  },

  "ru-RU": {
    "addButton": "Добавить",
    "cancelButton": "Отменить",
    "noMembers": "No members",
    "noMembersByFilter": "Нет участников по фильтру",
    "title": "Добавить участника"
  }
}  
</i18n>