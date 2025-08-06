<template>
    <component
      :is="href ? 'a' : 'button'"
      :href="href"
      class="button"
      :class="{
          'button_loading': loading,
          'button_outlined': outlined,
          'button_icon': icon,
          'button_small': small,
          'button_fluid': fluid
          },
          `button_${type}`"
      :disabled="isDisabled">
        <slot></slot>
        <div v-if="loading" class="button__spinner"></div>
    </component>
</template>

<script setup>
import { computed } from 'vue';


const props = defineProps({
    loading: {
        type: Boolean
    },
    type: {
        type: String,
        default: 'base',
        validator: (val) => {
            const BUTTON_TYPES = ['primary', 'secondary', 'base', 'warn', 'danger', 'transparent'];
            return BUTTON_TYPES.includes(val)
        }
    },
    outlined: {
        type: Boolean
    },
    disabled: {
        type: Boolean
    },
    icon: {
        type: Boolean
    },
    fluid: {
        type: Boolean,
        default: false
    },
    href: {
        type: String
    },
    small: {
        type: Boolean
    }
});

const isDisabled = computed(() => props.outlined || props.disabled);
</script>

<style scoped>

.button {
    --color-primary: #1976D2;
    --color-primary-hover: #2196F3;
    --color-primary-active: #1565C0;

    --color-base: #F3F5F6;
    --color-base-hover: #f9fbfc;
    --color-base-active: #D3DBDF;

    --color-transparent: transparent;
    --color-transparent-hover: #F3F5F6;
    --color-transparent-active: #D3DBDF;

    --color-danger: #F44336;
    --color-danger-active: #E57373;
    --color-danger-hover: #C62828;



    height: 40px;
    min-width: 160px;
    padding: 0 16px;
    display: inline-flex;
    column-gap: 4px;
    justify-content: center;
    align-items: center;
    color: #fff;
    border: 1px solid;
    cursor: pointer;
    border-radius: 8px;
    overflow: hidden;
}

.button:hover {
    text-decoration: none;
}

.button_small {
    height: 32px;
    min-width: 32px;
}

.button svg {
    color: currentColor
}

.button_fluid {
    width: 100%;
    flex-grow: 1;
}

/* button icon */
.button_icon {
    width: 40px;
    height: 40px;
    padding: 0;
    min-width: 0;
}

.button_icon.button_small {
    height: 32px;
    min-width: 32px;
    width: 32px;
}


/* primary */
.button_primary {
    color: #fff;
    background-color:  var(--color-primary);
    border-color: var(--color-primary);
}
.button_primary .button__spinner {
    border-color: #fff;
}
.button_primary:not(:disabled):hover {
    background-color: var(--color-primary-hover);
    border-color: var(--color-primary-hover);
}
.button_primary:active {
    background-color: var(--color-primary-active);
    border-color: var(--color-primary-active);
}


/* base */
.button_base {
    color: #263238;
    background-color:  var(--color-base);
    border-color: var(--color-base-active);
}
.button_base .button__spinner {
    border-color: #263238;
}
.button_base:not(:disabled):hover {
    background-color: var(--color-base-hover);
}
.button_base:active {
    background-color: var(--color-base-active);
}

/* transparent */
.button_transparent {
    color: #263238;
    background-color:  var(--color-transparent);
    border-color: var(--color-transparent);
}
.button_transparent .button__spinner {
    border-color: #263238;
}
.button_transparent:not(:disabled):hover {
    background-color: var(--color-transparent-hover);
}
.button_transparent:active {
    background-color: var(--color-transparent-active);
}


/* warn */
.button_danger {
    color: #fff;
    background-color:  var(--color-danger);
    border-color: var(--color-danger);
}
.button_danger .button__spinner {
    border-color: #fff;
}
.button_danger:not(:disabled):hover {
    background-color: var(--color-danger-hover);
}
.button_danger:active {
    background-color: var(--color-danger-active);
}


/* disabled */
.button:disabled {
    opacity: .4;
    cursor: not-allowed;
}


/* loading */
.button_loading {
    color: transparent;
    position: relative;
}
.button_loading *:not(.button__spinner) {
    visibility: hidden;
}


.button__spinner {
    width: 20px;
    height: 20px;
    border: 2px solid #FFF;
    border-bottom-color: transparent!important;
    border-radius: 50%;
    display: inline-block;
    box-sizing: border-box;
    animation: rotation 1s linear infinite;
    position: absolute;
    top: 50%;
    left: 50%;
    margin-top: -10px;
    margin-left: -10px;
}

@keyframes rotation {
    0% {
        transform: rotate(0deg);
    }
    100% {
        transform: rotate(360deg);
    }
}
</style>
