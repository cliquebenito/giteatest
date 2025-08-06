<template>
  <div class="dialog-overlay" @click="$emit('onClose')"></div>
  <div class="dialog-content">
    <div class="dialog-title">
      {{ t("privileges.delete.title") }}
      <button class="icon ui button small x-button" @click="$emit('onClose')">
        <svg-icon name="octicon-x" />
      </button>
    </div>
    <div class="dialog-message">
      {{ t("privileges.delete.message") }}
    </div>
    <div class="dialog-actions">
      <button class="button ui primary" @click="$emit('onConfirm')">
        {{ t("buttons.delete") }}
      </button>
      <button class="button ui" @click="$emit('onClose')">
        {{ t("buttons.cancel") }}
      </button>
    </div>
  </div>
</template>

<script>
import { useI18n } from "vue-i18n";
import { SvgIcon } from "../../svg";

export default {
  name: "PrivilegesRevokeDialog",
  data() {
    return {};
  },
  emits: ["onConfirm", "onClose"],
  setup() {
    const { t, locale } = useI18n({
      inheritLocale: true,
      useScope: "local",
    });
    return { t, locale };
  },
  components: { SvgIcon },
};
</script>

<style scoped>
.dialog-overlay {
  position: fixed;
  z-index: 20;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  background-color: rgba(0, 0, 0, 0.4);
  overflow: hidden;
}

.dialog-content {
  position: fixed;
  z-index: 21;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  overflow: hidden;
  width: 443px;
  min-height: 200px;
  padding: 24px;
  background-color: white;
  border-radius: 8px;
  overflow: hidden;
}

.dialog-content > :not(:last-child) {
  margin-bottom: 24px;
}

.dialog-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 20px;
  font-weight: 700;
}

.dialog-actions {
  display: flex;
  gap: 16px;
}

.dialog-actions > * {
  flex: 1;
}

.x-button {
  background-color: transparent !important;
  border: none !important;
  margin: 0 !important;
}
</style>
