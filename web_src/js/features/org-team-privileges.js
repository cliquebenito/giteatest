import { createApp } from "vue";
import { i18nDict } from "../i18n/index.js";
import TeamPrivileges from "../components/CustomPrivileges/TeamPrivileges.vue";

export function initOrgTeamPrivileges() {
  const el = document.getElementById("v-app-team-privileges");
  if (!el) {
    return;
  }

  const view = createApp(TeamPrivileges);
  view.use(i18nDict);
  view.mount(el);
}
