<template>
  <div class="body" :class="classByValue">
    {{ bool }}
  </div>
</template>

<script setup>
  import { convertToBool } from './helpers.js';

  const props = defineProps({
    value: {
      type: String,
      required: true
    }
  });

  const bool = convertToBool(props.value);
  const classByValue = props.value.toLowerCase();

</script>

<style scoped>
  .body {
    width: 60px;
    height: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
    border-radius: 50%;
    text-transform: uppercase;
    background-color: blue;
    color: #fff;
  }
  .true {
    background-color: var(--category-a-color);
  }
  .false {
    background-color: var(--category-e-color);
  }
</style>

