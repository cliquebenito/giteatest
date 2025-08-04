import { createApp } from "vue";
import AppLicensesList from "../components/Licenses/AppLicensesList.vue";
import { i18nDict } from "../i18n/index.js";

export function initLicensesList() {
  const el = document.getElementById("licenses-list-container");
  if (!el) {
    return;
  }

  const view = createApp(AppLicensesList);
  view.use(i18nDict);
  view.mount(el);
}
