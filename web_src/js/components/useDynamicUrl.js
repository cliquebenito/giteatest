import { computed } from 'vue';

export function useDynamicUrl(options) {
  const { link, appSubUrl } = options;

  const baseUrl = computed(() => {
    const cleanLink = link.replace(/^\/+/, '');

    if (appSubUrl) {
      const cleanAppSubUrl = appSubUrl.replace(/\/+$/, '');
      if (cleanLink.startsWith(cleanAppSubUrl)) {
        return cleanLink;
      }
      return `/${cleanLink}`;
    }

    return `/${cleanLink}`;
  });

  const getFullUrl = (path) => {
    const cleanPath = path.replace(/^\/+/, '');
    const cleanBase = baseUrl.value.replace(/\/+$/, '');

    return cleanPath.startsWith(cleanBase)
        ? cleanPath
        : `${cleanBase}/${cleanPath}`;
  };

  return {
    baseUrl,
    getFullUrl
  };
}
