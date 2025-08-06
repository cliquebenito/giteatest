<template>
    <span class="badge" :class="type">{{ text }}</span>
</template>

<script setup>
defineProps({
    type: {
        type: String,
        default: 'default'
    },
    text: {
        type: String,
        required: true
    }
})
</script>

<style scoped>
.badge {
    min-height: 18px;
    padding: 0 8px;
    border-radius: 16px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    text-transform: none;
    font-size: 12px;
    color: var(--sc-color-text-primary);
    font-family: var(--fonts-monospace);
}


.default {
    color: var(--sc-color-white);
    background-color: var(--sc-color-blue-lighten);
}


.base {
    color: var(--sc-color-white);
    background-color: var(--sc-color-grey);
}


.outlined {
    border: 1px solid var(--sc-color-stroke);
    color: var(--sc-color-grey-dark);
    background-color: transparent;
}


.info {
    color: var(--sc-color-grey-dark);
    background-color: var(--sc-color-grey);
}


.positive {
    color: var(--sc-color-white);
    background-color: var(--sc-color-green);
}


.negative {
    color: var(--sc-color-white);
    background-color: var(--sc-color-red);
}


.warning {
    color: var(--sc-color-white);
    background-color: var(--sc-color-orange);
}


.transparent {
    color: var(--sc-color-grey-dark);
    background-color: var(--sc-color-bg);
}
</style>