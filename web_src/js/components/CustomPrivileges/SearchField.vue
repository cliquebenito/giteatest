<template>
    <div class="search">
        <input
            type="text"
            class="search__field"
            :placeholder="t('placeholder')"
            v-model="filter"
            @input="handleInput($event)"
        >
        <svg-icon name="octicon-search" class="search__icon"></svg-icon>
    </div>
</template>

<script setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { SvgIcon } from '../../svg';

const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const emit = defineEmits(['filterChange']);
const filter = ref('')

const handleInput = (event) => {
    emit('filterChange', event.target.value);
};

</script>

<style scoped>
.search {
    display: flex;
    align-items: center;
    height: 40px;
    border: 1px solid #D5D9DD;
    padding: 10px 16px;
    border-radius: 8px;
    background: transparent;
    column-gap: 12px;
    flex-grow: 1;
    width: 100%;
  }

.search:focus-within {
    border-width: 2px;
    border-color:  #1A5EE6;
}

.search:focus-within .search__icon {
    color: #1A5EE6;
}

.search__field {
    flex-grow: 1;
    border: none;
    background: transparent;
    outline: none;
}

.search__icon {
    flex-shrink: 0;
    color: #2E3038;
}

.theme-dark  .search__icon {
    color: var(--color-caret)
}
</style>

<i18n>
{
    "en-US": {
        "placeholder": "Search",
    },

    "ru-RU": {
        "placeholder": "Поиск",
    }
}
</i18n>