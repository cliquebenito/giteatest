<template>
  <div class="tenant-search">
    <input
      type="text"
      class="tenant-search__field"
      :placeholder="$t('tenant.placeholder.search')"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    >
    <svg-icon name="octicon-search" class="tenant-search__icon"></svg-icon>
  </div>
</template>

<script>

export default  {
  name: 'tenant-search',
  props: {
    modelValue: {
      type: String,
      required: true
    },
  },
  emits: ['update:modelValue']
}

</script>

<style scoped>
  .tenant-search {
    display: flex;
    align-items: center;
    margin: 24px 0;
    width: 100%;
    height: 40px;
    border: 1px solid #D5D9DD;
    padding: 10px 16px;
    border-radius: 8px;
    background: transparent;
    column-gap: 12px;
  }

  .tenant-search:focus-within {
    border-width: 2px;
    border-color:  #1A5EE6;
  }

  .tenant-search:focus-within .tenant-search__icon {
    color: #1A5EE6;
  }

  .tenant-search__field {
    flex-grow: 1;
    border: none;
    background: transparent;
    outline: none;
  }

  .tenant-search__icon {
    flex-shrink: 0;
    color: #2E3038;
  }

  .theme-dark  .tenant-search__icon {
    color: var(--color-caret)
  }
</style>

<!--<i18n>-->
<!--{-->
<!--  "en-US": {-->
<!--    "placeholder": "Search tenants..."-->
<!--  },-->
<!--  "ru-RU": {-->
<!--    "placeholder": "Поиск по тенантам"-->
<!--  }-->
<!--}-->
<!--</i18n>-->
