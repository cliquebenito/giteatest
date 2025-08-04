<template>
    <div class="tt-card">
        <div class="tt-card__priority" :class="normalizedPriority">
            <svg-icon v-show="priority" :name="`priority-${normalizedPriority}`"/>
        </div>
        <div class="tt-card__content">
            <div class="tt-card__name">
                <a v-if="url" :href="url" target="_blank" class="tt-card__name-link">{{ code }}</a>
                <span v-else class="tt-card__name-unlink">{{ code }}</span>
                <base-badge v-if="state" :type="normalizedState" :text="state"></base-badge>
            </div>
            <p v-if="name" class="tt-card__description">{{ name }}</p>
            <p v-else-if="description" class="tt-card__description">{{ normalizedDescription }}</p>
        </div>
        <!-- <button class="tt-card__close" @click="onClick">
            <svg-icon name="octicon-x"></svg-icon>
        </button> -->
    </div>
    <confirm-modal
        :show="showDeleteModal"
        :title="t('modalDeleteTitle')"
        :description="t('modalDeleteDesc')"
        :confirmButtonText="t('modalDeleteConfirmButton')"
        @modalConfirm="handleConfirm"
        @modalCancel="handleCancel">
    </confirm-modal>
</template>


<script setup>
import { ref, computed } from 'vue';
import BaseBadge from '../../ui/BaseBadge.vue';
import ConfirmModal from '../../ui/ConfirmModal.vue';
import { SvgIcon } from '../../svg.js';
import { useI18n } from 'vue-i18n';
import * as PRIORITY from './constants/priority.js'
import * as STATE from './constants/state.js'

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const props = defineProps({
    url: {
        type: String,
        required: true
    },
    priority: {
        type: String,
        required: true,
        default: 'normal'
    },
    code: {
        type: String,
        required: true
    },
    name: {
        type: String,
    },
    state: {
        type: String,
        required: true
    },
    description: {
        type: String,
    }
});

const showDeleteModal = ref(false);

function onClick() {
    showDeleteModal.value = true;
}

function handleConfirm() {
    console.log('confirm');
}

function handleCancel() {
    showDeleteModal.value = false;
}

const normalizedState = computed(() => {
    if (!props.state) {
        return null;
    }

    switch(props.state.toLowerCase().trim()) {
        case STATE.STATE_FINISHED:
        case STATE.STATE_RESLOVED:
        case STATE.STATE_TO_FINISHED:
        case STATE.STATE_TO_CLOSED:
            return 'positive';
        case STATE.STATE_IN_PROGRESS:
        case STATE.STATE_TO_WORK:
        case STATE.STATE_TO_RETURN:
            return 'default';
        case STATE.STATE_TO_INFO:
        case STATE.STATE_NEED_INFO:
            return 'warning';
        case STATE.STATE_CANCELED:
        case STATE.STATE_TO_CANCEL:
            return 'negative';
        case STATE.STATE_TO_REOPEN:
            return 'outlined';
        default:
            return 'base';
    }
})


const normalizedPriority = computed(() => {
    if (!props.priority) {
        return 'normal';
    }

    switch(props.priority.toLowerCase().trim()) {
        case PRIORITY.PRIORITY_MAJOR:
            return 'major';
        case PRIORITY.PRIORITY_MEDIUM:
            return 'normal';
        case PRIORITY.PRIORITY_MINOR:
            return 'minor';
        default:
            return 'normal'
    }
})

const normalizedDescription = computed(() => {
    if (!props.description) {
        return ''
    }

    switch(props.description.toLowerCase()) {
        case 'not found':
            return t('errors.notFound')
        default:
            return props.description;
    }
})
</script>

<style scoped>
.tt-card {
    display: flex;
    align-items: flex-start;
    column-gap: 2px;
    padding: 8px;
    overflow: hidden;
    border-radius: 8px;
    position: relative;
}
.tt-card:hover {
    background-color: var(--sc-color-fill);
}
.tt-card:hover .tt-card__close {
    visibility: visible;
}
.tt-card__content {
    display: flex;
    flex-direction: column;
    row-gap: 4px;
}
.tt-card__priority {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    margin-top: 2px;
}
.tt-card__priority svg {
    width: 100%;
    height: 100%;
}
.tt-card__priority.minor {
    color: var(--sc-color-blue-light)
}
.tt-card__priority.major {
    color: var(--sc-color-red);
}
.tt-card__priority.normal {
    color: var(--sc-color-green);
}
.tt-card__name {
    display: flex;
    align-items: center;
    column-gap: 6px;
    color: var(--sc-color-primary);
    font-size: 15px;
    text-transform: uppercase;
    white-space: nowrap;
}

.tt-card__name-unlink {
    color: var(--sc-color-grey-dark);
}
.tt-card__description {
    color: var(--sc-color-grey-dark);
    font-size: 13px;
    line-height: 17px;
    overflow: hidden;
    text-overflow: ellipsis;
    width: 100%;
    white-space: nowrap;
    max-width: 290px;
}
.tt-card__close {
    border: none;
    background-color: transparent;
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    padding: 0;
    justify-content: center;
    cursor: pointer;
    color: var(--sc-color-text-primary);
    margin-left: auto;
    margin-right: 0;
    visibility: hidden;
    padding: 4px;
    flex-shrink: 0;
    position: absolute;
    top: 8px;
    right: 8px;
}
.tt-card__close svg {
    width: 100%;
    height: 100%;
}
</style>

<i18n>
{
    "en-US": {
        "modalDeleteConfirmButton": "Remove",
        "modalDeleteTitle": "Remove",
        "modalDeleteDesc": "Are you sure you want to remove this task from the linked ones?",
        "errors": {
            "notFound": "Information not found"
        }
    },

    "ru-RU": {
        "modalDeleteConfirmButton": "Удалить",
        "modalDeleteTitle": "Удаление",
        "modalDeleteDesc": "Вы уверены, что хотите удалить эту задачу из связанных?",
        "errors": {
            "notFound": "Информация не найдена"
        }
    }
}
</i18n>
