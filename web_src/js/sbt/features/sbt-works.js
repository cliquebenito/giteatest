export const initSbtWorks = () => {
  const menu = document.getElementById('menu');
  //  если меню one-work не проинициализировано, то завершаем выпонение скрипта
  if (!menu) {
    sessionStorage.removeItem('side-menu-state');
    return;
  }

  const setActiveKey = () => {
    menu?.setAttribute(
      'selected-keys',
      `["${location.pathname}${location.search}"]`
    );
  };

  const OWProjectName = window.localStorage.getItem('OWProjectName');
  const { oneWorkMenuConfig = {} } = window.config;

  const sideMenuRoot = document.querySelector('side-menu').shadowRoot;
  const widthSideMenu = document.querySelector('.with-side-menu');
  const sessionSideMenuState =  sessionStorage.getItem('side-menu-state');

  const toggleMenuVisible = () => {
    if (widthSideMenu.classList.contains('side-menu-active')) {
      sessionStorage.setItem('side-menu-state', 'close');
      widthSideMenu.classList.remove('side-menu-active');
    } else {
      sessionStorage.setItem('side-menu-state', 'open');
      widthSideMenu.classList.add('side-menu-active');
    }
  };

  observeSideMenu(sideMenuRoot);

  // Отображаем последнее сохраненное состяние меню при загрузке страницы
  if (sessionSideMenuState === 'close') {
    widthSideMenu.classList.remove('side-menu-active');
  } else {
    widthSideMenu.classList.add('side-menu-active');
  }

  const appSubUrl = window.config.appSubUrl;

  const logout = (e) => {
    e.preventDefault();
    const { csrfToken } = window.config;
    const formData = new FormData();
    formData.append('_csrf', csrfToken)

    fetch(`${appSubUrl}/user/logout`, {
      method: 'POST',
      body: formData
    }).then((res) => {
      if (res.ok) {
        window.location.href = `${appSubUrl || '/'}`;
        sessionStorage.removeItem('side-menu-state');
      } else {
        throw res;
      }
    }).catch((err) => console.warn('cant logout', err));
  };

  const addLogout = () => {
    const logoutLink = menu?.shadowRoot?.querySelector(
      '.title[href$="/user/logout"]'
    );
    if (logoutLink && !logoutLink.getAttribute('data-listened')) {
      logoutLink.setAttribute('data-listened', true);
      logoutLink?.addEventListener('click', logout);
    }
  };
  setActiveKey();
  document.querySelector('.close-menu-button').addEventListener('click', toggleMenuVisible)
  addLogout();

  if (OWProjectName && oneWorkMenuConfig.Works) {
    menu.setAttribute('mode', 'works');
    menu.setAttribute('project-uri', `${appSubUrl}/${OWProjectName}`);
  } else {
    menu.setAttribute('mode', 'separate');
  }


  // Добавляем файл стилей  веб-компоненту <side-menu />
  // для поддержки темной темы
  const style = document.createElement('style')
  style.innerHTML = `
      .ant-menu-sub.ant-menu-inline,
      .sidemenu__menu,
      .sidemenu {
          background-color: var(--color-box-body)
       }

      .ant-menu-submenu-arrow {
        visibility: var(--sidemenu-chevron-visibility)
      }

      .ant-menu-submenu-title,
      .ant-menu-submenu-arrow,
      .ant-menu-submenu:not(.ant-menu-submenu-active) .ant-menu-title-content,
      .ant-menu-item:not(.ant-menu-item-active) .anticon,
      .ant-menu-item:not(.ant-menu-item-active) a {
        color: var(--color-text)
      }

      #root .ant-menu-item-selected,
      #root .ant-menu-submenu-title:active,
      #root .ant-menu-item:active,
      #root .ant-menu-item-selected:active,
      #root .ant-menu-item-selected:hover {
         background-color: var(--sidemenu-selected-color);
         color: var(--color-text)
      }

      .ant-menu-item:not(.ant-menu-item-active) a {
        display: var(--display-menu-item-text);
      }

      .ant-menu-sub,
      .error-info__title.text-h3,
      .error-info__content.text-p {
        display: var(--display-submenu);
      }

      .logo,
      .sidemenu__project-info {
        visibility: var(--sidemenu-works-title-visibility)
      }

      .error-info svg {
        width: var(--sidemenu-error-svg-size);
        height: var(--sidemenu-error-svg-size);
      }

      .ant-menu-hidden {
          --display-submenu: none;
          --sidemenu-works-title-visibility: hidden;
      }
  `;


  sideMenuRoot.appendChild(style);
  document.querySelector('.side-menu-container').classList.remove('loading');

  // Когда меню свернуто и пользователь пытается прокликать на групповые пункты меню - раскрываем основное меню
  sideMenuRoot.querySelectorAll('.ant-menu-submenu-inline').forEach((submenu) => {
    submenu.addEventListener('click', (event) => {
        if (!widthSideMenu.classList.contains('side-menu-active')) {
          sessionStorage.setItem('side-menu-state', 'open');
          widthSideMenu.classList.add('side-menu-active');
          event.stopPropagation();
        }
      })
  })

  // DOM узлы в меню могут обновляться, удаляться и добавляться заново
  // в связи с этим необходимо поддерживать метод logout, 
  // так как на новом dom элементе обработчика события на ссылке logout уже не будет
  function observeSideMenu(menuNode) {
    const observer = new MutationObserver(() => {
      addLogout();
    });

    observer.observe(menuNode, {
      childList: true,
      subtree: true
    })
  }

};


