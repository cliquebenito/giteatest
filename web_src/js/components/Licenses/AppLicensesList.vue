<template>
  <div v-if="licenses.length > 0" class="licenses-list__container">
    <h4 class="licenses-list__title">
        <svg-icon name="license"></svg-icon>
        {{ t('title') }}
        <span v-if="isMoreThanLimit" class="sc-badge sc-badge_info">{{ licenses.length }}</span>
      </h4>
    <ul class="licenses-list__items">
      <li class="licenses-list__item" v-for="(license, index) in licenses.slice(0, limit)" :key="index">
        <a class="licenses-list__item-link" :href="license.Path">{{ license.Name }}</a>
      </li>
    </ul>
    <base-button v-if="isMoreThanLimit" class="licenses-list__button" @click="onClick">
      {{ isCollapsed ? t('buttonMore') : t('buttonLess') }}
    </base-button>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { SvgIcon } from "../../svg";
import { useI18n } from "vue-i18n";
import BaseButton from '../../ui/BaseButton.vue'

const { licenses } = window.config;
const LIMIT_COUNT = 5;
const isCollapsed = ref(true);
const limit = ref(LIMIT_COUNT);

const onClick = () => {
  if (limit.value === LIMIT_COUNT) {
    limit.value = licenses.length;
    isCollapsed.value = false;
  } else {
    limit.value = LIMIT_COUNT;
    isCollapsed.value = true;
  }
};

const isMoreThanLimit = computed(() => {
  return licenses.length > LIMIT_COUNT;
});

const { t } = useI18n({
  inheritLocale: true,
  useScope: "local",
});

</script>

<style scoped>
.theme-dark .licenses-list__container{
  --text-color: #9a9ea9;
  --bg-color: #2a2e3a;
}

.licenses-list__container {
  --text-color: #263238;
  --bg-color: #F3F5F6;

  padding: 16px;
  border: 1px solid var(--sc-color-stroke);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  row-gap: 16px;
  align-items: flex-start;
  margin-bottom: 16px;
}

.licenses-list__title {
  font-weight: 500;
  font-size: 15px;
  color: var(--text-color);
  display: flex;
  align-items: center;
  column-gap: 6px;
}

.licenses-list__badge {
  display: inline-flex;
  height: 16px;
  padding: 0 8px;
  background-color: var(--bg-color);
  color: #78909C;
  font-family: var(--font-monospace);
  font-size: 12px;
  line-height: 16px;
  border-radius: 16px;
}

.licenses-list__items {
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  row-gap: 4px;
  max-height: 206px;
  overflow-y: auto;
}


.licenses-list__item {
  font-size: 13px;
}


.licenses-list__item-link {
  color: #1976D2;
  cursor: pointer;
}

.licenses-list__item-link:hover {
  text-decoration: underline;
}


.licenses-list__button {
  height: 32px;
  font-size: 13px;
}
</style>


<i18n>
   {
    "en-US": {
      "title": "The repository contains licenses",
      "buttonMore": "Show all licenses",
      "buttonLess": "Hide"
    },
    "ru-RU": {
      "title": "Репозиторий содержит лицензии",
      "buttonMore": "Показать все лицензии",
      "buttonLess": "Скрыть"
    }
  }
</i18n>