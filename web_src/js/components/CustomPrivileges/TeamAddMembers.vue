<template>
    <div class="add-members">
        <div v-if="allUsers.length" class="add-members__content">
            <div class="add-members__search">
                <search-field @filterChange="handleFilterChange($event)"></search-field>
            </div>
            <div v-if="filteredUsers.length > 0" class="add-members__list">
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
            <div v-else class="add-members__filter">
                {{`${t('noMembersByFilter')}: "${filter}"`}}
            </div>
        </div>

        <div v-else class="add-members__empty">
            {{ t('noMembers') }}
            <svg-icon name="no-members"></svg-icon>
        </div>

        <div class="add-members__controls">
            <base-button :disabled="isLoading" type="primary" :loading="isLoading" @click="handleSubmit">{{ t('addButton') }}</base-button>
            <base-button :disabled="isLoading" @click="cancel">{{ t('cancelButton') }}</base-button>
        </div>
    </div>

</template>

<script setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';

import MemberCard from './MemberCard.vue';
import SearchField from './SearchField.vue';
import BaseButton from '../../ui/BaseButton.vue'



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

const emit = defineEmits(['submitMembers']);

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
.add-members {
    height: 100%;
    display: flex;
    flex-direction: column;
    row-gap: 24px;
}
.add-members__content {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    row-gap: 24px;
}
.add-members__search {
    flex-shrink: 0;
}
.add-members__list {
    flex-grow: 1;
    overflow-y: auto;
}
.add-members__filter {
    display: flex;
    flex-direction: column;
    align-items: center;
}
.add-members__controls {
    display: flex;
    align-items: center;
    column-gap: 16px;
    padding-top: 24px;
    justify-content: space-between;
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
