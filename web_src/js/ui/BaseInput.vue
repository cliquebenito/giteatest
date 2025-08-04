<template>
    <div class="input-container" :class="{'input-container_required': required, 'input-container_error': errorMessage, 'input-container_disabled': disabled }">
        <label v-if="label" :for="generatedId" class="input__label">{{ label }}</label>
        <div class="input__field-wrap">
            <input class="input__field" :id="generatedId" type="text" :placeholder="placeholder" v-model="model" :disabled="disabled"/>
            <svg-icon v-if="icon" :name="icon" class="input__icon"></svg-icon>
        </div>
        <div v-if="hint" class="input__hint">
            {{ hint }}
        </div>
        <div class="input__error-message" v-if="errorMessage">
            {{ errorMessage }}
        </div>
    </div>
</template>

<script setup>
import { computed } from 'vue';
import { v4 as uuidv4 } from 'uuid';
import { SvgIcon } from '../svg';

const props = defineProps({
    icon: {
        type: String
    },
    required: {
        type: Boolean
    },
    id: {
        type: String
    },
    label: {
        type: String
    },
    placeholder: {
        type: String
    },
    hint: {
        type: String
    },
    errorMessage: {
        type: String
    },
    disabled: {
        type: Boolean
    }
});

const emit = defineEmits('change');

const model = defineModel();

const generatedId = computed(() => {
    return props.id || `input-id-${uuidv4()}`
});
</script>

<style scoped>
.input-container {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    row-gap: 8px;
    width: 100%;
    min-width: 300px;
}

.input__hint,
.input__label {
    font-size: 13px;
    color: #78909C;
}

.input__field-wrap {
    width: 100%;
    height: 40px;
    border-radius: 8px;
    border: 1px solid #D5D9DD;
    overflow: hidden;
    padding: 10px 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}
.input__field-wrap:focus-within {
    border-color: #1976D2;
    border-width: 2px;
}
.input__field-wrap:focus-within .input__icon {
    color: #1976D2
}
.input__field {
    width: 100%;
    flex-grow: 1;
    font-size: 15px;
    outline: none;
    border: none;
}
.input__icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
}
.input__icon svg {
    color: currentColor
}


/* required */
.input-container_required .input__label::after {
    content: '*';
    margin-left: 4px;
    color: var(--color-red);
}


/* disabled */
.input-container_disabled .input__field-wrap {
    cursor: not-allowed;
    opacity: .4;
}
.input-container_disabled .input__field {
    cursor: not-allowed;
}

/* error */
.input-container_error .input__field-wrap {
    border-color: #F44336
}
.input__error-message {
    font-size: 13px;
    color: #F44336;
}
</style>
