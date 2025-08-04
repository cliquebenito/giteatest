<template>
    <div class="card" :class="{'disabled': isSelected}">
        <div v-if="addMode" class="card__checkbox">
            <div class="ui checkbox" :class="{'disabled': isLoading}">
                <input :disabled="isLoading" type="checkbox" :id="fieldId" :value="data.ID" class="hidden" v-model="checkedState" @change="handleCheckboxChange($event)"/>
                <label :for="fieldId"></label>
            </div>
        </div>
        <div class="card__image">
            <img :src="data.Avatar" alt="user avatar" />
        </div>
        <div class="card__body">
            <p class="card__name">{{ fullName }}</p>
            <p class="card__login">{{ data.LowerName }}</p>
        </div>
        <base-button type="transparent" icon :disabled="isLoading" v-if="!addMode" class="card__button" @click="handleDelete(data)">
            <svg-icon name="octicon-trash"></svg-icon>
        </base-button>
    </div>
</template>

<script setup>
import { computed } from 'vue';
import { SvgIcon } from '../../svg';
import { ref } from 'vue';

import BaseButton from '../../ui/BaseButton.vue';

const { appUrl } = window.config;

const props = defineProps({
    isSelected: {
        type: Boolean,
    },
    isLoading: {
        type: Boolean
    },
    addMode: {
        type: Boolean,
        required: true
    },
    data: {
        type: Object,
        required: true
    }
});

const emit = defineEmits(['removeMember', 'selectMember']);
const checkedState = ref(props.isSelected);

const handleDelete = (data) => {
    emit('removeMember', data);
};

const handleCheckboxChange = (event,) => {
    const { value, checked } = event.target;
    const payload = {
        action: checked ? 'add' : 'remove',
        id: value
    };
    emit('selectMember', payload)
};


const avatar = computed(() => {
    return `${appUrl}avatar/${data.avatar}`
});

const fullName = computed(() => {
    return props.data.FullName || props.data.Name;
});

const fieldId = computed(() => {
    return `team-member-${props.data.ID}`;
});

</script>

<style scoped>
    .card {
        display: flex;
        column-gap: 16px;
        border-bottom: 1px solid #D3DBDF;
        min-height: 76px;
        width: 100%;
        padding: 14px 8px;
    }
    .card__checkbox {
        align-content: center;
    }
    .card:first-child {
        border-top: 1px solid #D3DBDF;
    }
    .card:nth-child(odd) {
        background-color: #F3F5F6;;
    }
    .card__image {
        overflow: hidden;
        width: 48px;
        height: 48px;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .card__image img {
        max-width: 100%;
    }
    .card__body {
        display: flex;
        flex-direction: column;
        row-gap: 8px;
        flex-grow: 1;
    }
    .card__name {
        font-size: 15px;
        color: #263238;
        margin: 0;
    }
    .card__login {
        font-size: 13px;
        color: #78909C;
        margin: 0;
    }
    .card__button {
        width: 40px;
        height: 40px;
        color: #263238;
    }
    .card__button:hover {
        border-radius: 8px;
        background-color: #F3F5F6;
    }
</style>

<i18n>

</i18n>