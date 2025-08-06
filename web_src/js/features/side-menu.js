import '@sds-eng/fonts/index.css';
import '@sbt-works/side-menu';
import { handleEvent } from '@sbt-works/side-menu';

const MENU_TYPE_STANDALONE = 'standalone';
const MENU_TYPE_ONEWORK = 'works';

const { oneWorkMenuConfig, userInfo, appSubUrl, OWProjectName, OWProjectReposCount, notificationCount } = window.config;

const userName = userInfo.lowerName;

const XSSDMode  = window.config.XSSDMode || MENU_TYPE_STANDALONE;

const MODE_PROJECT = 'project';
const MODE_ORGANIZATION = 'organization';
const CONTEXT_URI_TYPE_TAIL = 'url-tail';
const CONTEXT_URI_TYPE_ID = 'ssd-id';

const PROPS_MENU_LAYOUT_OPTIONS =  {
  targets: [
    {
      selector: 'body',
      cssPropertyName: 'paddingLeft',
    },
  ],
};

const PROPS_MENU_LOGO = {
  type: 'text',
  title: 'SourceControl'
};

// convert nanosecdons to seconds
const PROPS_MENU_FALLBACK_TIMEOUT_SECONDS = oneWorkMenuConfig.FallbackTimeout / Math.pow(10, 9);

const MENU_PROPS_STANDALONE = {
  logo: PROPS_MENU_LOGO,
  selectedKey: setActiveMenuItem(userName, OWProjectName, oneWorkMenuConfig.MenuKey),
  mainMenu: {
    items: [
      { key: 'source_control__dashboard',
        label: getLabelByKey('dashboard'),
        link: getLinkWithSubPath('/'),
        icon: {
          type: 'svg',
          url: getLinkWithSubPath('/assets/img/svg/sc-grid.svg')
        }
      },
      { key: 'source_control__pulls',
        label: getLabelByKey('pulls'),
        link: getLinkWithSubPath('/pulls'),
        icon: {
          type: 'svg',
          url: getLinkWithSubPath('/assets/img/svg/sc-pull-request.svg')
        }
      },
      { key: 'source_control__explore',
        label: getLabelByKey('explore'),
        link: getLinkWithSubPath('/explore/repos'),
        icon: {
          type: 'svg',
          url: getLinkWithSubPath('/assets/img/svg/sc-bullet-list.svg')
        }
      },

      ...( userInfo.isAdmin ?
          [{ key: 'source_control__admin',
            label: getLabelByKey('control_panel'),
            link: getLinkWithSubPath(`/admin`),
            icon: {
              type: 'svg',
              url: getLinkWithSubPath('/assets/img/svg/sc-gear.svg')
            }
        }] : [])
    ]
  },
  user: {
    avatar: {
      source: userInfo.avatar,
      placeholder: getUserNamePlacehodler(userInfo)
    },
    name: getUserName(userInfo),
    email: userInfo.email
  },
  layoutOptions: PROPS_MENU_LAYOUT_OPTIONS,
  functionalMenu: {
    groups: [
      {
        key: 'common',
        children: [
          { key: 'source_control__favorite', label: getLabelByKey('favourites'), link: getLinkWithSubPath(`/${userName}?tab=stars`) },
          { key: 'source_control__notifications', label: getLabelByKey('notifications'), link: getLinkWithSubPath('/notifications'), notifications: notificationCount  || undefined},
          { key: 'source_control__subscriptions', label: getLabelByKey('subscriptions'), link: getLinkWithSubPath('/notifications/subscriptions') },
          { key: 'source_control__settings', label: getLabelByKey('settings'), link: getLinkWithSubPath('/user/settings') },
        ]
      },
      {
        key: 'user',
        children: [
          { key: 'source_control__profile',
            label: getLabelByKey('profile'),
            link: getLinkWithSubPath(`/${userName}`),
            icon: {
              type: 'svg',
              url: getLinkWithSubPath('/assets/img/svg/sc-profile.svg')
            }
          },
        ]
      },
      {
        key: 'footer',
        children: [
          { key: 'source_control__logout',
            label: getLabelByKey('logout'),
            navigation: 'soft'
          },
        ]
      }
    ]
  }
}

const isProjectManager = window.config.isProjectManager || false

const TOOL_ITEMS = {
  [oneWorkMenuConfig.MenuKey]: [
    {
      key: `source_control__org_repos`,
      label: getLabelByKey('repos'),
      link: getLinkWithSubPath(`/${OWProjectName}`),
      notifications: OWProjectReposCount || undefined,
      navigation: 'hard'
    },
    ...(
      isProjectManager ? [{
      key: `source_control__org_teams`,
      label: getLabelByKey('teams'),
      link: getLinkWithSubPath(`/org/${OWProjectName}/teams`),
      navigation: 'hard'
    }] : []),
    {
      key: `source_control__org_settings`,
      label: getLabelByKey('settings'),
      link: getLinkWithSubPath(`/org/${OWProjectName}/settings`),
      navigation: 'hard'
    }
  ]
};

const FALLBACK_OPTIONS = {
  timeout: PROPS_MENU_FALLBACK_TIMEOUT_SECONDS,
  logo: PROPS_MENU_LOGO,
  menuItems: [
    {
      key: oneWorkMenuConfig.MenuKey,
      label: "SourceControl",
      icon: {
        type: 'svg',
        url: getLinkWithSubPath('/assets/img/svg/sc-ow-logo.svg')
      }
    }
  ]
};

const MENU_PROPS = {
  serviceUrl: oneWorkMenuConfig.ServiceURL,
  mode: getOneWorkMode(),
  toolKey: oneWorkMenuConfig.ToolKey,
  contextUriType: getContextUriType(),
  contextUri: getContextUri(),
  selectedKey: setActiveMenuItem(userName, OWProjectName),
  layoutOptions: PROPS_MENU_LAYOUT_OPTIONS
};

if (getOneWorkMode() === 'project') {
  MENU_PROPS.toolItems = TOOL_ITEMS;
  MENU_PROPS.fallbackOptions = FALLBACK_OPTIONS;
}

// init menu
let menuComponent = null;

if (isStandaloneType(XSSDMode)) {
  menuComponent = document.createElement('side-menu-standalone-v4');
  menuComponent.setAttribute('props', JSON.stringify(MENU_PROPS_STANDALONE));
} else {
  menuComponent = document.createElement('side-menu-v4');
  menuComponent.setAttribute('props', JSON.stringify(MENU_PROPS));
}

document.body.append(menuComponent);


// зарегистрировать обработчик события заданного типа
handleEvent('sidemenu.functionalMenu.click', (event) => {
  const { key } = event.payload;
  if (key === 'source_control__logout') {
    handleLogout();
  }
});

// в зависимости от того, находимся мы в контексте проекта или нет определяется режим работы меню
function getOneWorkMode() {
  return OWProjectName ? MODE_PROJECT : MODE_ORGANIZATION;
}

// в зависимости от режима работы меню используются различные индентификаторы контекста
function getContextUriType() {
  return OWProjectName ? CONTEXT_URI_TYPE_TAIL : CONTEXT_URI_TYPE_ID;
}

// Формируем данные для параметра contextURI. В режиме проекта передаем ссылку на проект,
// в режиме организации используем ключ органиации заведенный в OneWork
function getContextUri() {
  return OWProjectName ? `${appSubUrl}/${OWProjectName}` : `${oneWorkMenuConfig.ContextURI}`.replaceAll('//', '/');
}

// Формируем ссылки на пункты меню с учетом возможного наличия перфикса в URL
function getLinkWithSubPath(link) {
  return `${appSubUrl}${link}`.replaceAll('//', '/');
}

// поддержка языковой версии для пунктов меню
// работает в режиме standalone
function getLabelByKey(key) {
  const lang = window.config.lang || 'ru-RU';
  const labelsDict = {
    'ru-RU': {
      'pulls': 'Запросы на слияние',
      'explore': 'Обзор',
      'teams': 'Команды',
      'repos': 'Репозитории',
      'settings': 'Настройки',
      'notifications': 'Уведомления',
      'profile': 'Мой профиль',
      'favourites': 'Избранное',
      'subscriptions': 'Подписки',
      'control_panel': 'Панель управления',
      'logout': 'Выход',
      'org_create': 'Создать проект',
      'dashboard': 'Активность'
    },
    'en-US': {
      'pulls': 'Pulls',
      'explore': 'Explore',
      'teams': 'Teams',
      'repos': 'Repositories',
      'settings': 'Settings',
      'notifications': 'Notifications',
      'profile': 'My profile',
      'favourites': 'Favourites',
      'subscriptions': 'Subscriptions',
      'control_panel': 'Control panel',
      'logout': 'Logout',
      'org_create': 'Create project',
      'dashboard': 'Activity'
    }
  }

  return labelsDict[lang][key] || key;
}

// Для выбора ключа активного пункта меню в зависимости от текущего URL
function setActiveMenuItem(userName, OWProjectName, fallbackKey) {
  const pathName = window.location.pathname;
  if (pathName.includes('/user/settings')) {
      return 'source_control__settings'
  }
  if (pathName.includes(`/org/create`)) {
    return 'source_control__org_create';
  }
  if (pathName.includes(`/org/${OWProjectName}/teams`)) {
      return 'source_control__org_teams';
  }
  if (pathName.includes(`/org/${OWProjectName}/settings`)) {
      return 'source_control__org_settings'
  }
  if (pathName.includes(`/${OWProjectName}`) && (!pathName.includes(`/pulls`))) {
      return 'source_control__org_repos';
  }
  if (pathName.includes(`/${userName}?tab=stars`)) {
      return 'source_control__favorite';
  }
  if (pathName.includes(`/${userName}`)) {
      return 'source_control__profile';
  }
  if (pathName.includes('/explore')) {
      return 'source_control__explore';
  }
  if (pathName.includes('/pulls')) {
      return 'source_control__pulls';
  }
  if (pathName.includes('/notifications')) {
    return 'source_control__notifications'
  }
  if (pathName.includes('/admin')) {
    return 'source_control__admin'
  }

  if (pathName.includes('/')) {
    return 'source_control__dashboard'
  }
  return fallbackKey
}

function getUserName(user) {
  return user.fullName || user.lowerName;
}

function getUserNamePlacehodler(user) {
  const name = getUserName(user);
  const [firstName, lastName] = name.split(' ');
  if (lastName) {
    return `${firstName[0]}${lastName[0]}`.toUpperCase();
  } else {
    return name.slice(0, 2).toUpperCase();
  }
}

// обработка выхода. При авторизации через IAM "выходим" с помощью перехода по прямой ссылке
// При авторизации через SourceControl выходим через POST на /user/logout
function handleLogout() {
  const { csrfToken, oneWorkMenuConfig: { LogoutLink } } = window.config;
  if (LogoutLink) {
    window.location.href = LogoutLink;
  } else {
    const formData = new FormData();
    formData.append('_csrf', csrfToken)

    fetch(`${appSubUrl}/user/logout`, {
      method: 'POST',
      body: formData
    }).then((res) => {
      if (res.ok) {
        window.location.href = `${appSubUrl || '/'}`;
      } else {
        throw res;
      }
    }).catch((err) => console.warn('can\'t logout', err));
  }
}

function isStandaloneType(type) {
  return type.toLowerCase().includes(MENU_TYPE_STANDALONE);
}

function isOneWorkType(type) {
  return type.toLowerCase().includes(MENU_TYPE_ONEWORK);
}
