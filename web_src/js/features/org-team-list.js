import { createApp } from "vue";
import AppCustomPrivileges from "../components/CustomPrivileges/App.vue";
import { i18nDict } from "../i18n/index.js";

export function initOrgTeamList() {
  const el = document.getElementById("v-app-team-list");
  if (!el) {
    return;
  }

  const view = createApp(AppCustomPrivileges);
  view.use(i18nDict);
  view.mount(el);
}
