import { createApp } from "vue";
import { i18nDict } from "../i18n/index.js";
import DefaultReviewers from "../components/DefaultReviewers/DefaultReviewers.vue";

export function initRepoDefaultReviewersSetting() {
  const el = document.getElementById("v-default-reviewers");
  if (!el) {
    return;
  }

  const view = createApp(DefaultReviewers);
  view.use(i18nDict);
  view.mount(el);
}
