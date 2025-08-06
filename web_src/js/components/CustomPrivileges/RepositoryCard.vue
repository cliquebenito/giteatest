<template>
    <div class="card">
        <div class="card__body">
            <a :href="url" class="card__title">
                {{ repository.Name }}
            </a>
            <div class="card__info">
                <div class="card__icons-group">
                    <div class="card__icon">
                        <svg-icon name="octicon-star"></svg-icon>
                        {{ repository.NumStars  }}
                    </div> 
                    <div class="card__icon">
                        <svg-icon name="octicon-git-branch"></svg-icon>
                        {{ repository.NumForks  }}
                    </div> 
                </div>
                <p class="card__updated">{{ updated }}</p>    
            </div>
        </div>
    </div>
</template>

<script setup>
import { formatDistanceToNow } from 'date-fns';
import { SvgIcon } from '../../svg';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import ruLocale from 'date-fns/locale/ru';

const { lang } = window.config;

const props = defineProps({
    isLoading: {
        type: Boolean
    },
    repository: {
        type: Object,
        required: true
    }
});


const { t } = useI18n({
    inheritLocale: true,
    useScope: 'local'
});

const { appUrl } = window.config;

const updated = computed(() => {
    const options = { addSuffix: true };
    if (lang.match('ru') !== null) {
        options.locale = ruLocale;
    }
    return `${t('updatedLabel')} ${formatDistanceToNow(new Date(props.repository.UpdatedUnix * 1000), options)}`;
});

const url = computed(() => {
    return  `${appUrl}${props.repository.OwnerName}/${props.repository.LowerName}`
});


</script>

<style scoped>
    .card {
        display: flex;
        flex-direction: row;
        align-items: center;
        column-gap: 16px;
        border-bottom: 1px solid #D3DBDF;
        min-height: 76px;
        width: 100%;
        padding: 14px 8px;
        justify-content: space-between;
    }
    .card__body {
        display: flex;
        flex-direction: column;
        row-gap: 4px;
        flex-grow: 1;
    }
    .card:first-child {
        border-top: 1px solid #D3DBDF;
    }
    .card:nth-child(even) {
        background-color: #F3F5F6;;
    }
    .card__title {
        font-size: 15px;
        font-weight: 500;
        color: #1976D2;
    }
    .card__info {
        display: flex;
        align-items: center;
        column-gap: 16px;
        font-size: 13px;
        color: #78909C;
    }
    .card__icons-group {
        display: flex;
        align-items: center;
        column-gap: 10px;
    }
    .card__icon {
        color: #78909C;
        display: inline-flex;
        align-items: center;
        column-gap: 2px;
    }
    .card__button {
        width: 40px;
        height: 40px;
        color: #263238;
    }
    .card__button:hover {
        border-radius: 8px;
        background-color: #F3F5F6;
    }
</style>

<i18n>
{
  "en-US": {
    "updatedLabel": "Updated"
  },

  "ru-RU": {
    "updatedLabel": "Обновлено"
  }
}  

</i18n>