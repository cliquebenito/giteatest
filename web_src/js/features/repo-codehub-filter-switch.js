import {createApp} from 'vue';
import CodeHubFilterSwitch from '../components/CodeHub/AppCodehubFilterSwitch.vue';


export function initRepoCodeHubFilterSwitch() {
  const el = document.getElementById('v-app-codehub-filter-switch');
  if (!el) return;

  const fileTreeView = createApp(CodeHubFilterSwitch, {...el.dataset});
  fileTreeView.mount(el);
}
