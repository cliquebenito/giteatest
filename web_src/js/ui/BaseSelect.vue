<template>
    <div class="select-wrap" :class="{'select-wrap_disabled': disabled}" ref="nodeRef">
        <label :for="generatedId" class="select__label">
            {{ label }}
        </label>
        <div class="select__field-wrap" @click="onSelectClick">
            <input class="select__field" type="text" :id="generatedId" :name="name" v-model="value" readonly :placeholder="placeholder" ref="inputNodeRef">
        </div>
        <ul v-if="show" class="select__dropdown" tabindex="0">
            <li class="select__dropdown-item" :class="{'select__dropdown-item_selected': isSelected(option)}" tabindex="0" v-for="(option, index) in options" :key="index" @click="onClickOption(option)">
                {{ getOption(option) }}
                <svg-icon name="octicon-check" v-if="isSelected(option)"></svg-icon>
            </li>
        </ul>
    </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { SvgIcon } from '../svg';
import { v4 as uuidv4 } from 'uuid';

const props = defineProps({
    options: {
        type: Array,
        required: true
    },
    name: {
        type: String
    },
    label: {
        type: String
    },
    disabled: {
        type: Boolean
    },
    modelValue: {
        required: true
    },
    placeholder: {
        type: String
    },
    id: {
        type: String
    }
});

const emit = defineEmits(['update:modelValue']);

onMounted(() => {
    document.body.addEventListener('click', outsideClickHandle);
});

onBeforeUnmount(() => {
    document.body.removeEventListener('click', outsideClickHandle);
});

const show = ref(false);
const nodeRef = ref(null);
const inputNodeRef = ref(null);

const value = computed(() => {
    if (props.modelValue === null) {
        return props.modelValue;
    } else if (typeof props.modelValue === 'object') {
        return props.modelValue.label
    } else {
        return props.modelValue
    }
});

const generatedId = computed(() => {
    return props.id || `select-id-${uuidv4()}`
});

function getOption(option)  {
    return option.label || option
}

function isSelected(option) {
    if (props.modelValue === null) {
        return false
    } else if (typeof option === 'object') {
        return props.modelValue.value === option.value;
    } else {
        return props.modelValue === option
    }
}

function onSelectClick() {
    if (!props.disabled) {
        if (show.value) {
            show.value = false;
            inputNodeRef.value.blur();
        } else {
            show.value = true;
            inputNodeRef.value.focus();
        }
        
    }
}

function onClickOption(option) {
    emit('update:modelValue', option);
    show.value = false;
}

function outsideClickHandle(event) {
    const target = event.target;
    if (!nodeRef.value.contains(target)) {
        show.value = false;
    }
}
</script>

<style scoped>
.select-wrap {
    display: flex;
    flex-direction: column;
    row-gap: 8px;
    position: relative;
    width: 100%;
    min-width: 300px;
}

.select__label {
    font-size: 13px;
    line-height: 17px;
    color: #78909C;
}

.select__field-wrap {
    height: 40px;
    width: 100%;
    border: 1px solid #D3DBDF;
    padding: 10px 16px;
    font-size: 15px;
    color: var(--color-text-main);
    border-radius: 8px;
    position: relative;
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
}

.select__field-wrap::after {
    content: '';
    flex-shrink: 0;
    display: block;
    width: 10px;
    border-style: solid;
    border-width: 5px 5px 0 5px;
    border-color: #263238 transparent transparent transparent;
}

.select-wrap:not(.select-wrap_disabled) .select__field-wrap:focus-within {
    border: 2px solid #1976D2;
    outline: none;
}

.select__field {
    border: none;
    outline: none;
    background: transparent;
    width: 100%;
    flex-grow: 1;
    cursor: pointer;
}

.select__dropdown {
    position: absolute;
    top: 100%;
    transform: translateY(8px);
    left: 0;
    width: 100%;
    display: flex;
    flex-direction: column;
    background: #fff;
    border: 1px solid #D3DBDF;
    box-shadow: 0 8px 16px rgba(0,0,0, .2);
    border-radius: 8px;
    overflow-y: auto;
    z-index: 10;
    max-height: 400px;
    padding: 0;
    margin: 0;
    list-style: none;
}

.select__dropdown-item {
    height: 40px;
    font-size: 15px;
    color: var(--color-text-main);
    padding: 10px 16px;
    cursor: pointer;
}
.select-wrap:not(.select-wrap_disabled) .select__dropdown-item:hover {
    color: #fff;
    background-color: #1976D2;
}
.select__dropdown-item_selected {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background-color: #F3F5F6;
}
.select__dropdown-item_selected svg {
    color: #1976D2;
}

.select__dropdown-item_selected:hover svg {
    color: #fff;
}

.select-wrap_disabled .select__label {
    pointer-events: none;
}
.select-wrap_disabled .select__field-wrap {
    cursor: not-allowed;
    opacity: .4;
}
.select-wrap_disabled .select__field {
    cursor: not-allowed;
}
</style>