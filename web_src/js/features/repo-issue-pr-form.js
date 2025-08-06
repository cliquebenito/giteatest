import {createApp} from 'vue';
import PullRequestMergeForm from '../components/PullRequestMergeForm.vue';
import {i18nDict} from "../i18n/index.js";

export function initRepoPullRequestMergeForm() {
  const el = document.getElementById('pull-request-merge-form');
  if (!el) return;

  const view = createApp(PullRequestMergeForm);
  view.use(i18nDict)
  view.mount(el);
}
