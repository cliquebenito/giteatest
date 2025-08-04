<template>
    <div v-if="show" class="overlay">
        <div class="modal">
            <div class="modal__header">
                <h3 class="modal__title">{{ title }}</h3>
                <base-button type="transparent" icon :disabled="isLoading" class="modal__close-btn" @click.prevent="handleCancel">
                    <svg-icon name="octicon-x"></svg-icon>
                </base-button>
            </div>
            <div class="modal__body">
                <slot></slot>
            </div>
            <div v-if="$slots.footer" class="modal__footer">
                <slot name="footer"></slot>
            </div>
        </div>
    </div>
</template>

<script setup>
import { watch } from 'vue';
import { SvgIcon } from '../svg'

import BaseButton from './BaseButton.vue';


const props = defineProps({
    isLoading: {
        type: Boolean
    },
    show: {
        required: true,
        type: Boolean
    },
    title: {
        required: true,
        type: String,
    },
});

const emit = defineEmits(['modalCancel']);


watch(() => props.show, (newShow) => {
    if (newShow) {
        document.body.classList.add('modal-view');
    } else {
        document.body.classList.remove('modal-view');
    }
});


function handleCancel() {
    emit('modalCancel');
}
</script>

<style scoped>
.overlay {
    width: 100vw;
    height: 100vh;
    background-color: rgba(0,0,0, .5);
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 100;
}

.modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    min-width: 440px;
    z-index: 101;
    padding: 24px;
    background: #fff;
    border-radius: 24px;
    display: flex;
    flex-direction: column;
    row-gap: 24px;
    font-family: var(--fonts-proportional);
}
.modal__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
}
.modal__close-btn svg {
    width: 24px;
    height: 24px;
}
.modal__title {
    font-weight: bold;
    margin-bottom: 0;
    font-size: 22px;
}
.modal__footer {
    display: flex;
    justify-content: center;
    column-gap: 16px;
}

</style>
