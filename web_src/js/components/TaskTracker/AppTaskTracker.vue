<template>
    <div v-if="!loading" class="tt-widget">
        <h4 class="tt-header">
            <strong>{{ t('title') }}</strong>
            <!-- <button
                v-if="!isRepoArchived && hasWritePermission"
                class="tt-button-add"
                :data-tooltip-content="t('addTaskTooltip')"
                @click="handleClickShowAddModal"
            >
                <svg-icon name="octicon-gear"></svg-icon>
            </button> -->
        </h4>
        <div v-if="data.units.length" class="tt-list">
            <task-card
                v-for="task of data?.units"
                :key="task.code"
                :url="task.url"
                :code="task.code"
                :state="task.state"
                :priority="task.priority"
                :name="task.name"
                :description="task.description"
            >
            </task-card>
        </div>
        <div v-else class="tt-empty">
            {{t('notLinkedTasksMessage')}}
        </div>
    </div>

    <div v-else>
        <BaseSpinner/>
    </div>

    <base-modal
        :show="showAddTaskModal"
        :title="t('modalAddTaskTitle')"
        @modalCancel="handleCancel"
    >
        <template #default>
            <base-input
                :label="t('modalAddTaskLabel')"
                :placeholed="t('modalAddTaskPlaceholder')"
                v-model="newTaskName">
            </base-input>
        </template>

        <template #footer>
            <base-button fluid type="primary" @click="handleConfirm">{{ t('modalAddTaskButtonConfirm') }}</base-button>
            <base-button fluid @click="handleCancel">{{ t('modalAddTaskButtonCancel') }}</base-button>
        </template>
    </base-modal>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue';
import TaskCard from './TaskCard.vue';
import BaseInput from '../../ui/BaseInput.vue';
import BaseModal from '../../ui/BaseModal.vue';
import BaseButton from '../../ui/BaseButton.vue';
import BaseSpinner from '../../ui/BaseSpinner.vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const { baseLink, csrfToken, issuePullId: issueId, pageData: { isRepoArchived, hasWritePermission } } = window.config;
const { globalFetch } = window;
const loading = ref(false);

const newTaskName = ref('');
const showAddTaskModal = ref(false);

const handleClickShowAddModal = () => {
    showAddTaskModal.value = true;
};

const handleDeleteTask = (payload) => {
    console.log('delete task', payload);
}

const EMPTY_TASK = { state: '', priority: '', name: '', code: '', url: ''};

let data = reactive({ units: [], errors: [] });

onMounted(() => {
    loading.value = true;
    globalFetch(`${baseLink}/unit_links`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application-json',
            'X-Csrf-Token': csrfToken
        },
        body: JSON.stringify({ pull_request_id: issueId })
    })
    .then((response) => response.clone())
    .then((res) => {
        return res.json();
    }).then((res) => {
        data.units = res.units || [];
        if (res.errors) {
            data.units = mergeErrorUnits(data.units, res.errors)
        }
    }).catch((err) => {
        console.warn('error get unit_links', err)
    }).finally(() => {
        loading.value = false;
        data.units = [ ...data.units].sort((a, b) => a.code > b.code ? 1 : -1)
    })
});

function handleConfirm() {
    console.log('confirm')
}

function handleCancel() {
    showAddTaskModal.value = false;
}

function mergeErrorUnits(units, errorUnits) {
    const _errorUntis = errorUnits.map(item => ({ ...EMPTY_TASK, ...item }))
    return [ ...units, ..._errorUntis];
}
</script>

<style scoped>
.tt-widget {
    width: 100%;
}
.tt-header {
    display: flex;
    align-items: center;
    column-gap: 4px;
    margin: 0;
}
.tt-button-add {
    background-color: transparent;
    border: none;
    outline: none;
    width: 24px;
    height: 24px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
}
.tt-button-add:hover {
    background-color: var(--sc-color-grey-light);
    border-radius: 4px;
}
.tt-list {
    padding: 12px 0;
    display: flex;
    flex-direction: column;
    row-gap: 4px;
}
.tt-empty {
    padding-top: 12px;
    color: var(--sc-color-text);
}
</style>


<i18n>
{
    "en-US": {
        "title": "Linked tasks",
        "addTaskTooltip": "Add linked task",
        "modalAddTaskButtonConfirm": "Add",
        "modalAddTaskButtonCancel": "Add",
        "modalAddTaskLabel": "Linked task",
        "modalAddTaskPlaceholder": "Enter linked task code",
        "modalAddTaskTitle": "Add linked task",
        "notLinkedTasksMessage": "No one task is linked"
    },

    "ru-RU": {
        "title": "Связанные задачи",
        "addTaskTooltip": "Добавить связанную задачу",
        "modalAddTaskButtonConfirm": "Добавить",
        "modalAddTaskButtonCancel": "Отменить",
        "modalAddTaskLabel": "Связанная задача",
        "modalAddTaskPlaceholder": "Введите связанную с этим запросом задачу",
        "modalAddTaskTitle": "Добавить связанную задачу",
        "notLinkedTasksMessage": "Нет связанных задач"
    }
}
</i18n>
