import {createApp} from "vue";
import Tenants from "../../components/TenantAdministration/AppTenant.vue";
import {SvgIcon} from "../../svg.js";
import {i18nDict} from "../../i18n/index.js";

export function initTenantsAdministration() {
  const el = document.getElementById('tenants-administration');
  if (!el) return;

  const view = createApp(Tenants);
  view.component('SvgIcon',  SvgIcon)
  view.use(i18nDict)
  view.mount(el);
}
