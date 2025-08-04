import {createApp} from 'vue';
import AppPrivileges from '../components/OrgPrivileges/AppPrivileges.vue';
import {i18nDict} from "../i18n/index.js";

export function initOrgPrivileges() {
  const el = document.getElementById('privileges');
  if (!el) {
    return
  };

  const view = createApp(AppPrivileges);
  view.use(i18nDict)
  view.mount(el);
}
