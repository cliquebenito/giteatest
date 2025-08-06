<template>
    <component :is="variant" class="title" :class="`title_${type}`,`title_${variant}`">
        <slot></slot>
    </component>
</template>

<script setup>
defineProps({
    variant: {
        type: String,
        default: 'h1',
        validator: (val) => {
            return ['h1', 'h2', 'h3', 'h4', 'h5'].includes(val)
        }
    },
    type: {
        type: String,
        default: 'base',
        validator: (val) => {
            return ['base', 'monospace'].includes(val);
        }
    }
})
</script>

<style scoped>
.title {
    color: var(--color-text-main);
    font-weight: bold;
}

.title_base {
    font-family: var(--fonts-proportional);
}

.title_monospace {
    font-family: var(--fonts-monospace);
}

.title_h1 {
    font-size: 48px;
    line-height: 56px;
    margin-bottom: 32px;
}

.title_h2 {
    font-size: 36px;
    line-height: 48px;
    margin-bottom: 24px;
}

.title_h3 {
    font-size: 28px;
    line-height: 34px;
    margin-bottom: 16px;
}

.title_h4 {
    font-size: 15px;
    line-height: 20px;
    margin-bottom: 8px;
}

.title_h5 {
    font-size: 13px;
    line-height: 16px;
    margin-bottom: 4px;
}
</style>