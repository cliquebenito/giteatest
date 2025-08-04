<template>
 <div class="body" :class="classByRating">
   {{ rating }}
 </div>
</template>

<script setup>
  import { convertToRating } from './helpers.js';

  const props = defineProps({
    value: {
      type: String,
      required: true
    }
  });


  const rating = convertToRating(props.value);
  const classByRating = `rating_${rating.toLowerCase()}`;

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

  .rating_a {
    color: #fff;
    background-color: var(--category-a-color);
  }

  .rating_b  {
    color: #fff;
    background-color: var(--category-b-color);
  }

  .rating_c  {
    color: #2E3038;
    background-color: var(--category-c-color);
  }

  .rating_d  {
    color: #fff;
    background-color: var(--category-d-color);
  }

  .rating_e {
    color: #fff;
    background-color: var(--category-e-color);
  }

</style>
