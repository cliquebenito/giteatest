<template>
  <Transition name="aside-dialog">
    <div v-if="show" class="dialog-wrap">
      <div @click.self="onClose('overlay')" class="aside-dialog-overlay"></div>

      <div class="aside-dialog">
        <div class="aside-dialog__header">
          <slot name="header">
            <h2 class="aside-dialog__title">
              {{ title }}
            </h2>
            <base-button type="transparent" icon @click.stop="onClose('close')" class="aside-dialog__close-btn">
              <svg-icon name="octicon-x"></svg-icon>
            </base-button>
          </slot>
        </div>

        <div class="aside-dialog__body">
          <slot></slot>
        </div>

        <div v-if="$slots.footer" class="aside-dialog__footer">
          <slot name="footer"></slot>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { watch, onUnmounted } from 'vue';
import { SvgIcon } from '../svg';
import BaseButton from './BaseButton.vue'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  show: {
    type: Boolean,
    required: true
  }
});

const emit = defineEmits(['cancel']);

watch(() => props.show, (newShow) => {
  if (newShow) {
    document.body.classList.add('modal-view');
    document.addEventListener('keydown', handleKeyDown);
  } else {
    document.body.classList.remove('modal-view');
    document.removeEventListener('keydown', handleKeyDown);
  }
});

onUnmounted(() => {
  document.body.classList.remove('modal-view');
  document.removeEventListener('keydown', handleKeyDown);
});

function handleKeyDown(e) {
  if (e.key === 'Escape') {
    onClose('escape');
  }
}

function onClose(target) {
    emit('cancel', target);
}

</script>

<style scoped>
.dialog-wrap {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
}

.aside-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 1000;
}

.aside-dialog {
  position: fixed;
  width: 443px;
  height: 100vh;
  background: var(--color-box-body);
  right: 0;
  top: 0;
  overflow: hidden;
  z-index: 1001;
  padding: 32px;
  display: flex;
  flex-direction: column;
}

.aside-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.aside-dialog__body {
  flex-grow: 1;
  overflow-y: auto;
}

.aside-dialog__title {
  font-size: 28px;
  font-weight: 600;
  color: var(--color-text-main);
}

.aside-dialog__close-btn svg {
  width: 24px;
  height: 24px;
}

.aside-dialog-enter-active,
.aside-dialog-leave-active {
  transition: all 0.3s ease;
}

.aside-dialog-enter-from,
.aside-dialog-leave-to {
  opacity: 0;
}

.aside-dialog-enter-from .aside-dialog,
.aside-dialog-leave-to .aside-dialog {
  transform: translateX(100%);
}

.aside-dialog-enter-to .aside-dialog,
.aside-dialog-leave-from .aside-dialog {
  transform: translateX(0);
}
</style>
