<template>
    <div class="loader-wrapper">
        <span class="loader"></span>
    </div>
</template>

<style scoped>
.loader-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
}
.loader {
    width: 48px;
    height: 48px;
    border: 5px solid var(--sc-color-fill);
    border-bottom-color: transparent;
    border-radius: 50%;
    display: inline-block;
    box-sizing: border-box;
    animation: rotation 1s linear infinite;
}
</style>