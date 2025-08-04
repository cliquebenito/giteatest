import { createApp } from "vue";
import AppLicense from "../components/Licenses/AppLicenses.vue";
import { i18nDict } from "../i18n/index.js";

export function initLicenses() {
  const el = document.getElementById("license");
  if (!el) {
    return;
  }

  const view = createApp(AppLicense);
  view.use(i18nDict);
  view.mount(el);
}
