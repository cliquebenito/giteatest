import {createApp} from "vue";
import PreReceiveGitHooks from "../../components/PreReceiveGitHooksForm.vue";
import {i18nDict} from "../../i18n/index.js";

export function initPreReceiveGitHooksForm() {
  const el = document.getElementById('pre-receive-git-hooks-form');
  if (!el) return;

  const view = createApp(PreReceiveGitHooks);
  view.use(i18nDict)
  view.mount(el);
}
