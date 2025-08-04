package audit

import "code.gitea.io/gitea/modules/json"

// Event тип для перечисления событий
type Event int

// перечисление событий
// https://dzo.sw.sbc.space/wiki/pages/viewpage.action?pageId=250505529
const (
	// Серверные события
	ServiceStartEvent Event = iota + 1 // Старт сервера
	ServiceStopEvent                   // Остановка сервера

	// События пользователей
	UserCreateEvent                // Пользователь создан
	UserProfileEditEvent           // Профиль пользователя изменен
	UserDeleteEvent                // Пользователь удален
	UserPasswordChangeEvent        // Пароль пользователя изменен
	UserNameChangeEvent            // Имя пользователя изменено
	UserTokenCreateEvent           // Персональный токен пользователя создан
	UserTokenDeleteEvent           // Персональный токен пользователя удален
	GPGKeyAddEvent                 // GPG ключ добавлен
	GPGKeyRemoveEvent              // GPG ключ удален
	SSHKeyAddEvent                 // SSH ключ добавлен
	SSHKeyRemoveEvent              // SSH ключ удален
	UserHookAddEvent               // Добавлен пользовательский hook
	UserHookRemoveEvent            // Удален пользовательский hook
	UserHookDisableEvent           // Отключен пользовательский hook
	UserHookEnableEvent            // Включен пользовательский hook
	UserHookSettingsChangeEvent    // Изменены настройки пользовательского hook
	UserAddToProjectTeamEvent      // Пользователь добавлен в команду
	UserRemoveFromProjectTeamEvent // Пользователь удален из команды
	UserAvatarChange               // Изменен аватар пользователя
	UserAvatarDelete               // Удален аватар пользователя
	ProjectAvatarChange            // Изменен аватар проекта
	ProjectAvatarDelete            // Удален аватар проекта

	// События прав/разрешений
	GlobalRightsGrantedEvent      // Выданы глобальные права
	GlobalRightsRemoveEvent       // Удалены глобальные права
	ProjectTeamRightsGrantedEvent // Пользователю выданы права команды в проекте
	ProjectTeamRightsChangeEvent  // Изменены права команды в проекте
	ProjectTeamRightsRemoveEvent  // У пользователя удалены права команды в проекте
	RepositoryRightsGrantedEvent  // Выданы права на репозиторий
	RepositoryRightsChangeEvent   // Изменены права на репозиторий
	RepositoryRightsRemoveEvent   // Удалены права на репозиторий
	UserTuzRightsGrantedEvent     // Пользователю выдана роль TUZ

	// События репозитория
	RepositoryOpenEvent // Репозиторий открыт пользователем
	BuildRunEvent       // Запущена сборка
	BranchCreateEvent   // Ветка создана
	BranchDeleteEvent   // Ветка удалена
	ChangesPushEvent    // Изменения отправлены в репозиторий
	// todo: Разделить на отдельные события: репозиторий извлечен(git pull) и репозиторий склонирован(git clone)
	RepositoryPullOrCloneEvent // Репозиторий извлечен или склонирован (git pull/git clone)

	// События pull requests
	PRMergeEvent  // Pull request объеденен
	PRCreateEvent // Pull request создан
	PRCloseEvent  // Pull request закрыт
	PRReopenEvent // Pull request переоткрыт
	PRDeleteEvent // Pull request удалён

	// События получения ссылок
	UnitLinksRequestCreateEvent // Создан запрос на получение ссылки
	PullRequestLinksAddEvent    // Отправлена ссылка на добавление pull request
	PullRequestLinksDeleteEvent // Отправлена ссылка на удаление pull request
	UnitTaskLockEvent           // Блокировка задачи
	UnitTaskUnlockEvent         // Разблокировка задачи
	PullRequestsUpdateEvent     // Отправлена ссылка об обновлении pull request

	// События аутентификации
	UserLoginEvent  // Пользователь вошел
	UserLogoutEvent // Пользователь вышел

	// События безопасности
	UnauthorizedRequestEvent // Неавторизованный доступ к ресурсу

	// События тенатов
	TenantCreateEvent     // Тенант создан
	TenantEditEvent       // Тенант изменен
	TenantActivateEvent   // Тенант активирован
	TenantDeactivateEvent // Тенант деактивирован
	TenantDeleteEvent     // Тенант удален

	// События настроек репозиториев
	RepositoryCreateEvent                     // Репозиторий создан
	RepositoryDeleteEvent                     // Репозиторий удален
	RepositoryAdoptEvent                      // Репозиторий принят
	RepositoryImportEvent                     // Репозиторий импортирован
	RepositoryForkEvent                       // Репозиторий форкнут
	HookInRepositoryAddEvent                  // Добавлен hook в репозитории
	HookInRepositoryRemoveEvent               // Удален hook в репозитории
	HookInRepositoryDisableEvent              // Отключен hook в репозитории
	HookInRepositoryEnableEvent               // Включен hook в репозитории
	HookSettingsInRepositoryChangeEvent       // Изменены настройки hook в репозитории
	RepositorySettingsChangeEvent             // Настройки репозитория изменены
	BranchProtectionAddToRepositoryEvent      // Защита на ветку добавлена в репозитории
	BranchProtectionDeleteFromRepositoryEvent // Защита на ветку удалена из репозитория
	BranchProtectionUpdateInRepositoryEvent   // Защита на ветку обновлена в репозитории
	BranchDeleteAfterMergeSettingEnableEvent  // Включено удаление ветки при слиянии
	BranchDeleteAfterMergeSettingDisableEvent // Отключено удаление ветки при слиянии
	PRMergeSettingDeleteEvent                 // Настройка слияния pull request удалена из репозитория
	PRMergeSettingUpdateEvent                 // Настройка слияния pull request обновлена в репозитории
	ReviewSettingCreateEvent                  // Настройка правил ревью добавлена
	ReviewSettingUpdateEvent                  // Настройка правил ревью обновлена
	ReviewSettingDeleteEvent                  // Настройка правил ревью удалена

	// События организаций
	ProjectCreateEvent               // Проект создан
	ProjectDeleteEvent               // Проект удален
	ProjectEditEvent                 // Проект отредактирован
	HookInProjectAddEvent            // Добавлен hook в проекте
	HookInProjectRemoveEvent         // Удален hook в проекте
	HookInProjectDisableEvent        // Отключен hook в проекте
	HookInProjectEnableEvent         // Включен hook в проекте
	HookSettingsInProjectChangeEvent // Изменены настройки hook в проекте
	ProjectSettingsChangeEvent       // Настройки проекта изменены
	TeamAddToProjectEvent            // Команда добавлена в проект
	TeamRemoveFromProjectEvent       // Команда удалена из проекта
	TeamUpdateInProjectEvent         // Команда обновлена в проекте

	// События настроек системы
	DefaultOrSystemHookAddEvent            // Добавлен стандартный или системный hook
	DefaultOrSystemHookRemoveEvent         // Удален стандартный или системный hook
	DefaultOrSystemHookDisableEvent        // Отключен стандартный или системный hook
	DefaultOrSystemHookEnableEvent         // Включен стандартный или системный hook
	DefaultOrSystemHookSettingsChangeEvent // Изменены настройки стандартного или системного hook
	GitHookAddEvent                        // Добавлен git hook
	GitHookEditEvent                       // Изменен git hook
	GitHookRemoveEvent                     // Удален git hook
	GitHookStartEvent                      // Запущен git hook
	GitHookFinishEvent                     // Завершен git hook

	// События cron задач
	CronTaskRegistered // Задача крон зарегистрирована
	CronTaskRun        // Задача крон запущена
	CronTaskFinished   // Задача крон завершена
	CronTaskCancel     // Задача крон отменена
	CronTaskLock       // Задача крон заблокирована
	CronTaskUnlock     // Задача крон разблокирована

	// События админских настроек
	AdminDashboardOpen                 // Открыта панель администратора
	AdminConfigurationOpen             // Открыта конфигурация
	AdminConfigurationChange           // Изменена конфигурация
	SystemNoticesOpen                  // Открыты системные уведомления
	SystemNoticesDelete                // Удалены системные уведомления
	SystemNoticesClear                 // Зачищены системные уведомления
	AuthenticationSourceManagementOpen // Открыто управление аутентификацией
	AuthenticationSourceAdd            // Добавлена аутентификация
	AuthenticationSourceEdit           // Изменена аутентификация
	AuthenticationSourceDelete         // Удалена аутентификация
	UserEmailManagementOpen            // Открыта панель управления эл. почтами пользователя
	UserEmailActivate                  // Активирована эл. почта пользователя
	UserEmailDeactivate                // Деактивирована эл. почта пользователя
	UserEmailDelete                    // Удалена эл. почта пользователя
	ApplicationsSettingsOpen           // Открыты настройки приложений
	ApplicationsSettingsAdd            // Добавлены настройки приложения
	ApplicationsSettingsEdit           // Изменены настройки приложения
	ApplicationsSettingsDelete         // Удалены настройки приложения
	ApplicationsSettingsGenerateSecret // Сгенерированы секреты для настроек приложения
	MonitoringStacktraceOpen           // Открыта мониторинговая трассировка стека
	StacktraceProcessCancel            // Отмена процесса стека
	DiagnosisReportDownload            // Скачен диагностический отчет
	MonitorQueuesPanelOpen             // Открыта мониторинговая панель очередей
	MonitorQueueOpen                   // Открыт мониторинг очереди
	QueueNumberOfWorkersChange         // Изменено количество воркеров
	QueueAllItemsRemove                // Удалены все элементы из очереди

	// События комментариев
	CommentDeleteEvent     // Комментарий удален
	CommentCreateCodeEvent // Комментарий создан
	CommentUpdateEvent     // Комментарий обновлен
	// События ролей
	PrivilegesRevokeEvent // Роль удалена
	PrivilegesGrantEvent  // Роль назначена
	// События Kafka
	CreateRepositorySendEvent // Отправлено событие о создании репозитория
	// События ТУЗа
	TuzCreateEvent // Создана ТУЗ

	// Событие получени creds из sec man
	// TODO выпилить после внесения VCS-1684 и заменить на SecManReadSecretEvent
	MTLSCredsGetFromSecMan
	// События создания tls config с mtls certs
	TLSConfigCreatingWithMTLS
	// События codeowners
	CodeOwnersSettingsChangeEvent // Изменены настройки codeowners
	CodeOwnersSettingsGrantEvent  // Назначены настройки codeowners
	CodeOwnersSettingsRevokeEvent // Удалены настройки codeowners
	CodeOwnersSettingsUpdateEvent // Обновлены настройки codeowners
	CodeOwnersAssignEvent         // Назначены права codeowners
	ReviewerAssignEvent           // Назначены права codeowners

	// События SecMan
	SecManReadSecretEvent  // Чтение секрета
	SecManApplySecretEvent // Обновление секрета или объектов с ним связанных

	// События CodeHub
	CodeHubMarkSetEvent              // Установка метки CodeHub
	CodeHubMarkDeleteEvent           // Удаление метки CodeHub
	ExternalMetricCounterSetEvent    // Установка внешнего счетчика
	ExternalMetricCounterDeleteEvent // Удаление внешнего счетчика

	// События CustomPrivileges
	AddCustomPrivilegesEvent    // Добавление пользовательских прав
	UpdateCustomPrivilegesEvent // Обновление пользовательских прав
	RemoveCustomPrivilegesEvent // Удаление пользовательских прав

	// События настроек Sonar
	SonarSettingsCreateEvent
	SonarSettingsUpdateEvent
	SonarSettingsDeleteEvent
)

// Описание событий
var events = map[Event]string{
	ServiceStartEvent:                         "Service start",
	ServiceStopEvent:                          "Service stop",
	UserCreateEvent:                           "Create user",
	UserProfileEditEvent:                      "Edit user profile",
	UserDeleteEvent:                           "Delete user",
	UserPasswordChangeEvent:                   "Change user password",
	UserNameChangeEvent:                       "Change user name",
	UserTokenCreateEvent:                      "Create user token",
	UserTokenDeleteEvent:                      "Delete user token",
	GPGKeyAddEvent:                            "Add GPG key",
	GPGKeyRemoveEvent:                         "Remove GPG key",
	SSHKeyAddEvent:                            "Add SSH key",
	SSHKeyRemoveEvent:                         "Remove SSH key",
	UserHookAddEvent:                          "Add user hook",
	UserHookRemoveEvent:                       "Remove user hook",
	UserHookDisableEvent:                      "Disable user hook",
	UserHookEnableEvent:                       "Enable user hook",
	UserHookSettingsChangeEvent:               "Change user hook settings",
	UserAddToProjectTeamEvent:                 "Add user to project team",
	UserRemoveFromProjectTeamEvent:            "Remove user from project team",
	UserAvatarChange:                          "Change user avatar",
	UserAvatarDelete:                          "Delete user avatar",
	UserTuzRightsGrantedEvent:                 "Grant user tuz rights",
	ProjectAvatarChange:                       "Change project avatar",
	ProjectAvatarDelete:                       "Delete project avatar",
	GlobalRightsGrantedEvent:                  "Granted global rights",
	GlobalRightsRemoveEvent:                   "Remove global rights",
	ProjectTeamRightsGrantedEvent:             "Granted project team rights to user",
	ProjectTeamRightsChangeEvent:              "Change project team rights",
	ProjectTeamRightsRemoveEvent:              "Remove project team rights at user",
	RepositoryRightsGrantedEvent:              "Granted repository rights",
	RepositoryRightsChangeEvent:               "Change repository rights",
	RepositoryRightsRemoveEvent:               "Remove repository rights",
	RepositoryOpenEvent:                       "Open repository",
	BuildRunEvent:                             "Run build",
	BranchCreateEvent:                         "Create branch",
	BranchDeleteEvent:                         "Delete branch",
	ChangesPushEvent:                          "Push changes",
	RepositoryPullOrCloneEvent:                "Pull or clone repository",
	PRMergeEvent:                              "Merge pull request",
	PRCreateEvent:                             "Create pull request",
	PRCloseEvent:                              "Close pull request",
	PRReopenEvent:                             "Reopen pull request",
	PRDeleteEvent:                             "Delete pull request",
	UnitLinksRequestCreateEvent:               "Create unit links request",
	PullRequestLinksAddEvent:                  "Send unit binding event to task tracker",
	PullRequestLinksDeleteEvent:               "Send unit binding removal event to task tracker",
	PullRequestsUpdateEvent:                   "Send an event of updating pull request status",
	UnitTaskLockEvent:                         "Lock unit task",
	UnitTaskUnlockEvent:                       "Unlock unit task",
	UserLoginEvent:                            "User login",
	UserLogoutEvent:                           "Logout user",
	UnauthorizedRequestEvent:                  "Unauthorized request",
	TenantEditEvent:                           "Edit tenant",
	TenantActivateEvent:                       "Activate tenant",
	TenantDeactivateEvent:                     "Deactivate tenant",
	TenantDeleteEvent:                         "Delete tenant",
	RepositoryCreateEvent:                     "Create repository",
	RepositoryDeleteEvent:                     "Delete repository",
	RepositoryAdoptEvent:                      "Adopt repository",
	RepositoryImportEvent:                     "Import repository",
	RepositoryForkEvent:                       "Fork repository",
	HookInRepositoryAddEvent:                  "Add hook in repository",
	HookInRepositoryRemoveEvent:               "Remove hook in repository",
	HookInRepositoryDisableEvent:              "Disable hook in repository",
	HookInRepositoryEnableEvent:               "Enable hook in repository",
	HookSettingsInRepositoryChangeEvent:       "Change hook settings in repository",
	RepositorySettingsChangeEvent:             "Change repository settings",
	BranchProtectionAddToRepositoryEvent:      "Branch protection add to repository",
	BranchProtectionDeleteFromRepositoryEvent: "Branch protection delete from repository",
	BranchProtectionUpdateInRepositoryEvent:   "Branch protection update in repository",
	BranchDeleteAfterMergeSettingEnableEvent:  "Enable branch delete after merge setting",
	BranchDeleteAfterMergeSettingDisableEvent: "Disable branch delete after merge setting",
	PRMergeSettingDeleteEvent:                 "Delete pull request merge setting",
	PRMergeSettingUpdateEvent:                 "Update pull request merge setting",
	ProjectCreateEvent:                        "Create project",
	ProjectDeleteEvent:                        "Delete project",
	ProjectEditEvent:                          "Edit project",
	HookInProjectAddEvent:                     "Add hook in project",
	HookInProjectRemoveEvent:                  "Remove hook in project",
	HookInProjectDisableEvent:                 "Disable hook in project",
	HookInProjectEnableEvent:                  "Enable hook in project",
	HookSettingsInProjectChangeEvent:          "Change hook settings in project",
	ProjectSettingsChangeEvent:                "Change project settings",
	TeamAddToProjectEvent:                     "Add team to project",
	TeamRemoveFromProjectEvent:                "Remove team from project",
	TeamUpdateInProjectEvent:                  "Update team in project",
	DefaultOrSystemHookAddEvent:               "Add default or system hook",
	DefaultOrSystemHookRemoveEvent:            "Remove default or system hook",
	DefaultOrSystemHookDisableEvent:           "Disable default or system hook",
	DefaultOrSystemHookEnableEvent:            "Enable default or system hook",
	DefaultOrSystemHookSettingsChangeEvent:    "Change default or system hook settings",
	GitHookAddEvent:                           "Add git hook",
	GitHookEditEvent:                          "Edit git hook",
	GitHookRemoveEvent:                        "Remove git hook",
	GitHookStartEvent:                         "Start git hook",
	GitHookFinishEvent:                        "Finish git hook",
	CronTaskRegistered:                        "Registered cron task",
	CronTaskRun:                               "Run cron task",
	CronTaskFinished:                          "Finish cron task",
	CronTaskCancel:                            "Cancel cron task",
	CronTaskLock:                              "Lock cron task",
	CronTaskUnlock:                            "Unlock cron task",
	AdminDashboardOpen:                        "Open admin dashboard",
	AdminConfigurationOpen:                    "Open admin configuration",
	AdminConfigurationChange:                  "Change admin configuration",
	SystemNoticesOpen:                         "Open system notices",
	SystemNoticesDelete:                       "Delete system notices",
	SystemNoticesClear:                        "Clear system notices",
	AuthenticationSourceManagementOpen:        "Open authentication source management",
	AuthenticationSourceAdd:                   "Add authentication source",
	AuthenticationSourceEdit:                  "Edit authentication source",
	AuthenticationSourceDelete:                "Delete authentication source",
	UserEmailManagementOpen:                   "Open user email management",
	UserEmailActivate:                         "Activate user email",
	UserEmailDeactivate:                       "Deactivate user email",
	UserEmailDelete:                           "Delete user email",
	ApplicationsSettingsOpen:                  "Open applications settings",
	ApplicationsSettingsAdd:                   "Add applications settings",
	ApplicationsSettingsEdit:                  "Edit applications settings",
	ApplicationsSettingsDelete:                "Delete applications settings",
	ApplicationsSettingsGenerateSecret:        "Generate secret for application settings",
	MonitoringStacktraceOpen:                  "Open monitoring stacktrace",
	StacktraceProcessCancel:                   "Cancel stacktrace process",
	DiagnosisReportDownload:                   "Download diagnosis report",
	MonitorQueuesPanelOpen:                    "Open monitor queues panel",
	MonitorQueueOpen:                          "Open monitor queue",
	QueueNumberOfWorkersChange:                "Change number of workers of queue",
	QueueAllItemsRemove:                       "Remove all items queue",
	CreateRepositorySendEvent:                 "Send create repository event",
	CommentDeleteEvent:                        "Delete comment",
	CommentCreateCodeEvent:                    "Create code comment",
	CommentUpdateEvent:                        "Update comment",
	TuzCreateEvent:                            "Create tuz",
	PrivilegesRevokeEvent:                     "Revoke privileges",
	PrivilegesGrantEvent:                      "Grant privileges to user",
	TenantCreateEvent:                         "Create tenant",
	MTLSCredsGetFromSecMan:                    "[READ_SECRET] Read mtls credentials",
	TLSConfigCreatingWithMTLS:                 "Create TLS configuration with the mtls certificates",
	CodeOwnersSettingsChangeEvent:             "Change code owners settings",
	CodeOwnersSettingsGrantEvent:              "Grant code owners settings",
	CodeOwnersSettingsRevokeEvent:             "Revoke code owners settings",
	CodeOwnersSettingsUpdateEvent:             "Update code owners settings",
	CodeOwnersAssignEvent:                     "Assign code owners",
	ReviewerAssignEvent:                       "Assign reviewers",
	SecManReadSecretEvent:                     "[READ_SECRET] Read secret from secret storage",
	SecManApplySecretEvent:                    "[APPLYING_SECRET] Update secret or related objects",
	CodeHubMarkSetEvent:                       "Set code hub mark",
	CodeHubMarkDeleteEvent:                    "Delete code hub mark",
	AddCustomPrivilegesEvent:                  "Add custom privileges",
	UpdateCustomPrivilegesEvent:               "Update custom privileges",
	RemoveCustomPrivilegesEvent:               "Remove custom privileges",
	ExternalMetricCounterSetEvent:             "Set external metric counter",
	ExternalMetricCounterDeleteEvent:          "Delete external metric counter",
	ReviewSettingCreateEvent:                  "Review setting to repository",
	ReviewSettingUpdateEvent:                  "Review setting update in repository",
	ReviewSettingDeleteEvent:                  "Review setting delete from repository",
	SonarSettingsCreateEvent:                  "Create sonar settings",
	SonarSettingsUpdateEvent:                  "Update sonar settings",
	SonarSettingsDeleteEvent:                  "Delete sonar settings",
}

// String возвращает описание событий
func (e Event) String() string {
	return events[e]
}

// MarshalJSON функция для преобразования Event в json
func (e *Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}
