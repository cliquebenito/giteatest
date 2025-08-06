import {createApp} from "vue";
import SonarConnectForm from "../components/SonarConnectForm.vue";

export function initRepoSonarConnection() {
  const el = document.getElementById('form-sonar-connect');
  if (!el) return;

  const view = createApp(SonarConnectForm);
  view.mount(el);
}
