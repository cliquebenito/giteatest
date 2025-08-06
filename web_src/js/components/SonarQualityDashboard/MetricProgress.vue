<template>
  <div class="body" :class="getClassByPercent()">
    <svg width="60" height="60" viewBox="0 0 60 60">
      <circle :r="INNER_RADIUS" :cx="OUTER_RADIUS" :cy="OUTER_RADIUS" fill="transparent" stroke="#F1F2F4" :stroke-width="STROKE_WIDTH"></circle>
      <circle class="progress" :r="INNER_RADIUS" :cx="OUTER_RADIUS" :cy="OUTER_RADIUS" fill="transparent" stroke="#60e6a8" :stroke-width="STROKE_WIDTH" :stroke-dasharray="strokeDashArray" :stroke-dashoffset="strokeDashOffset"></circle>
    </svg>
    <div class="percent">
      {{ percentLabel }}
    </div>
  </div>
</template>

<script setup>
  import { convertToPercent } from './helpers.js';

  const props = defineProps({
    value: {
      type: String,
      required: true
    },
    isInvert: {
      type: Boolean,
      required: true
    }
  });

  const TOTAL_PERCENT = 100;
  const INNER_RADIUS = 28;
  const OUTER_RADIUS = 30;
  const STROKE_WIDTH = 4;

  const LIMIT_SUCCESS = 90;
  const LIMIT_WARN = 51;
  const LIMIT_D = 50;

  const percentLabel = convertToPercent(props.value);
  // for specific metrics, duplication for example, need invert progress value. 0% is good, 100% is bad
  const percentNumber = Number(percentLabel.slice(0, -1));
  const percent = props.isInvert ? TOTAL_PERCENT - percentNumber : percentNumber;

  const strokeDashArray = 2 * Math.PI * INNER_RADIUS;
  const strokeDashOffset = strokeDashArray * ((TOTAL_PERCENT - percent) / TOTAL_PERCENT);

  const getClassByPercent = () => {
    if (percent >= LIMIT_SUCCESS) {
      return 'rating_a'
    } else if (percent >= LIMIT_WARN) {
      return 'rating_c'
    } else {
      return 'rating_e'
    }
  }
</script>

<style>
  .category {
    width: 120px;
    display: flex;
    flex-direction: column;
    align-items: center;
    row-gap: 8px;
  }
  .body {
    width: 60px;
    height: 60px;
    position: relative;
  }

  .body svg {
    transform: rotate(-90deg);
  }

  .percent {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: var(--color-text);
    font-size: 16px;
    display: flex;
  }

  .rating_a .progress {
    color: #fff;
    stroke: var(--category-a-color);
  }

  .rating_b .progress {
    color: #fff;
    stroke: var(--category-b-color);
  }

  .rating_c .progress {
    color: #2E3038;
    stroke: var(--category-c-color);
  }

  .rating_d .progress {
    color: #fff;
    stroke: var(--category-d-color);
  }

  .rating_e .progress {
    color: #fff;
    stroke: var(--category-e-color);
  }

</style>
