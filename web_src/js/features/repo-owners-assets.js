import {createApp} from 'vue';
import RepoOwnersAsset from '../components/RepoAssets/RepoOwnersAsset.vue';
import {i18nDict} from '../i18n/index.js';

export function initRepoOwnersAsset() {
  const el = document.getElementById('repo-owners-asset');
  if (!el) return;

  const view = createApp(RepoOwnersAsset);
  view.use(i18nDict);
  view.mount(el);
}
