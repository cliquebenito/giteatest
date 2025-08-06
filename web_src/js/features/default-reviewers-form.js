import { createApp } from "vue";
import { i18nDict } from "../i18n/index.js";
import DefaultReviewersSettingsForm from "../components/DefaultReviewers/DefaultReviewersSettingsForm.vue";

export function initRepoDefaultReviewersSettingsForm() {
  const el = document.getElementById("v-default-reviewers-form");
  if (!el) {
    return;
  }

  const view = createApp(DefaultReviewersSettingsForm);
  view.use(i18nDict);
  view.mount(el);
}
