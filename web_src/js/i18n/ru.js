export const ruDict = {
  buttons: {
    cancel: "Отменить",
    clear: "Очистить",
    reset: "Сбросить",
    save: "Сохранить",
    submit: "Отправить",
    add: "Добавить",
    delete: "Удалить",
  },
  actions: "Действия",
  license: {
    permission: "Permission",
    limitations: "Limitations",
    conditions: "Conditions",
  },

  yes: "Да",
  no: "Нет",

  privileges: {
    delete: {
      title: "Удаление",
      message: "Вы уверены, что хотите удалить пользователя из проекта?",
    },
    addButton: "Добавить участника",
    addingMember: "Добавление участника",
    editingMember: "Редактирование",
    select: "Выберите группу привилегий",
    table: {
      header: {
        name: "ФИО и логин",
        role: "Группа привилегий в проекте",
        edit: "Редактирование",
      },
      emptyMessage: "Нет данных",
    },
    searchPlaceholder: "Поиск...",
    owner: "Владелец проекта",
    reader: "Пользователь с правами на чтение",
    writer: "Пользователь с правами на запись",
    manager: "Менеджер",
  },

  tenant: {
    placeholder: {
      search: "Поиск по тенантам",
      create: "Название тенанта",
      edit: "Название тенатна",
    },

    isActiveLabel: "Тенант активен",

    createButton: "Создать тенант",

    title: {
      page: "Тенанты",
      create: "Создание",
      edit: "Редактирование",
      projects: "Привязанные проекты",
      delete: "Удаление",
    },

    tooltip: {
      edit: "Редактировать тенант",
      delete: "Удалить тенант",
    },

    tableTenantsHeaders: {
      name: "Название",
      projects: "Проекты",
      created: "Создано",
      edit: "Редактирование",
    },

    emptyTable: "Нет данных по вашему запросу",

    tableProjectsHeaders: {
      name: "Название",
      create: "Создано",
    },

    hint: "Если тенант не активен, то все связанные с ним проекты и репозитории перестанут отображаться на платформе",

    emptyProjects: "К данному тенанту не привязан пока ни один проект",

    deleteWarning:
      "Вы уверены, что хотите удалить {name}? Все связанные с ним проекты и репозитории будут также удалены",

    message: {
      success: {
        delete: "Тенант - {name} удален",
        create: "Тенант - {name} создан",
        update: "Тенант - {name} обновлен",
      },
      failure: {
        fetch: "Не удалось получить данные. {error}",
        delete: "Не удалось удалить тенант. {error}",
        create: "Не удалось создать тенант. {error}",
        update: "Не удалось обновить тенант. {error}",
      },
    },
  },
};
