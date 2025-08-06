import {createApp} from 'vue';
import AppTaskTracker from '../components/TaskTracker/AppTaskTracker.vue';
import {i18nDict} from "../i18n/index.js";

export function initTaskTracker() {
  const el = document.getElementById('tt-widget');
  if (!el) {
    return
  };

  const view = createApp(AppTaskTracker);
  view.use(i18nDict)
  view.mount(el);
}
