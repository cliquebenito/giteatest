import {createApp} from "vue";
import AppSonarQualityDashboard from "../components/SonarQualityDashboard/AppSonarQualityDashboard.vue";
import { i18nDict } from '../i18n/index.js';

export function initRepoSonarQualityDashboard() {
  const el = document.getElementById('sonar-quality-dashboard');
  if (!el) return;

  const view = createApp(AppSonarQualityDashboard);
  view.use(i18nDict)
  view.mount(el);
}
