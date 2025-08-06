import { createI18n } from 'vue-i18n'
import {ruDict} from "./ru.js";
import {enDict} from "./en.js";

export const i18nDict = createI18n({
  legacy: false,
  locale: window.config.lang || 'ru-RU',
  messages: {
    'ru-RU': ruDict,
    'en-US': enDict
  }
})
