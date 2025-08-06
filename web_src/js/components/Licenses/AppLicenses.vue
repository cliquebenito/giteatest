<template>
  <div
    v-if="hasLicenseInfo"
    class="flex border-[1px] rounded-[4px] border-gray-20 p-4"
  >
    <div class="license-container">
      <div class="license-info">
        <div class="license-info__header">
          <svg-icon name="license" />
          <h4 class="license__title">
            {{ licenseInfo.name }}
          </h4>
        </div>
        <p class="license-info__desc" v-html="licenseInfo.description" />
      </div>
      <div class="license-meta">
        <div class="column-license__meta">
          <h5 class="license-meta__title">
            {{ t("license.permission") }}
          </h5>
          <ul class="license-meta__list license-meta__list_permissions">
            <li
              v-for="item in licenseInfo?.permissions"
              :key="item"
            >
                {{ item }}
            </li>
          </ul>
        </div>
        <div class="column-license__meta">
          <h5 class="license-meta__title">
            {{ t("license.conditions") }}
          </h5 >
          <ul class="license-meta__list license-meta__list_conditions">
            <li
              v-for="item in licenseInfo.conditions"
              :key="item"
            >
                {{ item }}
            </li>
          </ul>
        </div>
        <div class="column-license__meta">
          <h5 class="license-meta__title">
            {{ t("license.limitations") }}
          </h5>
          <ul class="license-meta__list license-meta__list_limitations">
            <li
              v-for="item in licenseInfo.limitations"
              :key="item"
            >
                {{ item }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { SvgIcon } from "../../svg";
import { useI18n } from "vue-i18n";

const { t } = useI18n({
  inheritLocale: true,
  useScope: "local",
});

const { filename, username, reponame, repoId, branch } = window.config.pageData;

let isLoading = ref(true);
let licenseInfo = ref({});
let alert = ref(null);

const licenseFileNames = [
  "li[cs]en[cs]e(s?)",
  "legal",
  "copy(left|right|ing)",
  "unlicense",
  "l?gpl([-_ v]?)(\\d\\.?\\d)?",
  "bsd",
  "mit",
  "apache",
]
  .map((licName) => `(${licName})`)
  .join("|");

const fileExtensions = [".md", ".rst", ".html", ".txt"];

const licenseRegexp = new RegExp(
  `${licenseFileNames}(\..*(${fileExtensions.join("|")}))?$`,
  "gim"
);

const isFileLicense = (filename) => {
  return licenseRegexp.test(filename);
};


const fetchData = async () => {
  const { appUrl = "/" } = window.config;
  const formData = new FormData();
  const path = `${username}/${reponame}/${filename}`;
  formData.append('repository_id', repoId);
  formData.append('branch', branch);
  formData.append('path_file', path);

  const URL = `${appUrl}${username}/${reponame}/license`;
  try {
    const response = await fetch(URL, { method: 'post', body: formData});
    if (!response.ok) {
      throw new Error(response.statusText);
    }
    const data = await response.json();
    licenseInfo.value = data;
  } catch (error) {
    console.warn(error)
  } finally {
    isLoading.value = false;  }
};

const hasLicenseInfo = computed(() => {
  return licenseInfo.value.name || licenseInfo.value.description;
});

onMounted(() => {
  if (isFileLicense(filename)) {
    fetchData();
  }
});
</script>

<style scoped>
  .license-container {
    display: flex;
    align-items: flex-start;
    padding: 16px;
    border: 1px solid var(--sc-color-stroke);
    border-radius: 8px;
    margin-bottom: 24px;
  }
  .license-info__header {
    display: flex;
    align-items: center;
    column-gap: 4px;
    margin-bottom: 8px;
    min-height: 24px;
  }

  .license__title {
    color: var(--sc-color-text);
    font-weight: bold;
    font-size: 15px;
    margin: 0;
  }

  .license-info__desc {
    color: var(--sc-color-text-secondary);
    font-size: 13px;
    line-height: 17px;
  }

  .license-meta {
    display: flex;
    justify-content: space-between;
  }

  .license-meta__title {
    font-size: 13px;
    color: var(--sc-color-text);
    line-height: 17px;
    margin-bottom: 8px;
    font-weight: bold;
  }

  .column-license__meta {
    width: 187px;
  }

  .license__title,
  .column-license__meta > p {
    color: var(--sc-color-text);
  }

  .license-meta__list {
    margin: 0;
    padding: 0;
    color: var(--sc-color-text-secondary);
    font-size: 13px;
    line-height: 17px;
    list-style: none;
  }

  .license-meta__list li::before {
      content: '';
      display: inline-block;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      margin-right: 4px;
  }

  .license-meta__list_permissions li::before {
    background-color: var(--sc-color-green-60);
  }

  .license-meta__list_conditions li::before {
    background-color: var(--sc-color-blue-60);
  }

  .license-meta__list_limitations li::before {
    background-color: var(--sc-color-red-60);
  }
</style>
