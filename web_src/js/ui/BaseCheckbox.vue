<template>
    <div class="wrap" :class="{'disabled': disabled}">
        <input class="checkbox" type="checkbox" :id="generatedId" :name="name" v-model="model" :value="value" :disabled="disabled" @change="handleOnChange">
        <div v-if="hasDescription" class="label-wrap">
            <label v-if="label" class="label" :for="generatedId">{{ label }}</label>
            <p v-if="description" class="description">{{ description }}</p>
        </div>
    </div>
</template>

<script setup>
import { computed } from 'vue';
import { v4 as uuidv4 } from 'uuid';

const props = defineProps({
    id: {
        type: String
    },
    value: {
        type: [String, Number]
    },
    name: {
        type: String
    },
    label: {
        type: String
    },
    description: {
        type: String
    },
    disabled: {
        type: Boolean,
        default: false
    }
});

const emit = defineEmits(['change'])

const model = defineModel({
    modelValue: {
        required: true
    }
});

const generatedId = computed(() => {
    return props.id || `input-checkbox-id-${uuidv4()}`
});

const hasDescription = computed(() => {
    return !!props.label || !!props.description 
});

function handleOnChange(event) {
    emit('change', event);
}
</script>

<style scoped>
.wrap {
    --border-color-default: #D3DBDF;
    --bg-color-checked: #1976D2;
    --bg-color-hover: #2196F3;
    --bg-color-pressed: #1565C0;
    display: flex;
    align-items: flex-start;
    column-gap: 8px;
}

.label-wrap {
    display: flex;
    flex-direction: column;
    row-gap: 4px;
}

.label {
    font-size: 15px;
    color: var(--color-text-main);
    line-height: 20px;
    cursor: pointer;
}

.description {
    font-size: 13px;
    color: #78909C;
    line-height: 17px;
}

input {
    appearance: none;
    width: 20px;
    height: 20px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background-color: #fff;
    border: 1px solid var(--border-color-default);
    border-radius: 4px;
    cursor: pointer
}

/* checked state */
input:checked {
    background-image: url(../../svg/check.svg);
    background-color: var(--bg-color-checked);
    border-color: var(--bg-color-checked);
    background-position: center;
}
input:checked:active {
    background-color: var(--bg-color-pressed);
    border-color: var(--bg-color-pressed);
}


/* hover state */
input:hover:not(:checked):not(:focus):not(:focus-within):not(:disabled) {
    border-width: 2px;
    border-color: var(--border-color-default);
}


/* active state */
input:active:not(:checked):not(:disabled) {
    border-color: #B0BEC5;
}


/* focus state */
input:focus:not(:disabled),
input:focus-within:not(:disabled) {
    outline: none;
    border: 2px solid var(--bg-color-checked);
}


/* disabled state */
.wrap.disabled {
    opacity: .5;
}
.wrap.disabled input,
.wrap.disabled label,
input:disabled {
    cursor: not-allowed;
}
.wrap.disabled input:not(:checked) {
    background: #F3F5F6;
}

</style>