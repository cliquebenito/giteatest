package role_model

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/casbinlogger"
)

const InnerSource = "InnerSource"

// securityEnforcer основной интерфейс для управления настройками ролевой моделью
var securityEnforcer *casbin.SyncedEnforcer

// configureRoleModel возвращает за конфигурированную модель контроля доступа
/*
	Конфигурация модели содержит разделы:
		r - request_definition - Раздел определяет аргументы в функции e.Enforce(...).
		p - policy_definition - Раздел определяет значение политики.
		g - role_definition - Данный раздел используется для определения отношений наследования ролей RBAC.
Так же при использовании в разделе matchers служит построителем графических отношений,
который использует график для сравнения объекта запроса с объектом политики.
		e - policy_effect - это определение для политического эффекта. Он определяет, следует ли одобрять запрос на доступ,
если запросу соответствуют несколько правил политики. Например, одно правило разрешает и другое отрицает.
		m - matchers - Раздел является определением для политических соответствий. matchers - это выражения,
которые определяют, как правила политики вычисляются в соответствии с запросом.

	Текущая настройка(описана в формате: (раздел)/(запись в разделе) - (значение записи) - (описание):
		r/r - sub, tenant, project, act - Поля имеют следующий смысл:
																				sub - объект доступа.
																				tenant - тенант к которому запрашивается доступ.
																				project - проект к которому запрашивается доступ.
																				act - действие, которое необходимо совершить.
		p/p - sub, tenant, project, act - Поля имеют следующий смысл:
																				sub - объект доступа.
																				tenant - тенант к которому предоставляется доступ.
																				project - проект к которому предоставляется доступ.
																				act - действие, которое разрешено совершать.
		g/g - _, _ - означает, что в отношениях наследования участвуют две стороны. Например, owner, own - означает,
что роль owner содержит в себе доступ к действию own
		e/e - some(where (p.eft == allow)) - если есть какое-либо согласованное правило политики allow, конечным эффектом является allow
		m/m - r.sub == p.sub && r.tenant == p.tenant && r.project == p.project && g(p.act, r.act) - проверяет что:
																																																- мы нашли политику для пользователя из запроса
																																																- тенант в политике и в запросе совпадают
																																																- проект в политике и в запросе совпадают
																																																- содержит ли политика запрашиваемое действие в списке разрешенных действий
*/

// InitRoleModelDB инициализирует БД для работы с ролевой моделью
func InitRoleModelDB() error {
	a, err := db.RoleModelAdapter()
	if err != nil {
		log.Error("Error has occurred while getting role model adapter. Error: %v", err)
		return err
	}
	securityEnforcer, err = casbin.NewSyncedEnforcer(configureRoleModel(), a)
	if err != nil {
		log.Error("Error has occurred while creating security enforcer. Error: %v", err)
		return err
	}
	return nil
}

// InitRoleModel инициализирует ролевую модель, если она включена
func InitRoleModel() error {
	if !setting.SourceControl.TenantWithRoleModeEnabled {
		return nil
	}

	log.Info("Start initializing role model")

	dbAdapter, err := db.RoleModelAdapter()
	if err != nil {
		log.Error("Error has occurred while getting role model adapter. Error: %v", err)
		return err
	}

	securityEnforcer, err = casbin.NewSyncedEnforcer(configureRoleModel(), dbAdapter)
	if err != nil {
		log.Error("Error has occurred while creating security enforcer. Error: %v", err)
		return err
	}

	logger := casbinlogger.New()
	securityEnforcer.SetLogger(logger)
	ctx := context.Background()

	if err = securityEnforcer.LoadPolicy(); err != nil {
		log.Error("Error has occurred while loading policy. Error: %v", err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), OWN.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", OWN.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), CREATE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", CREATE.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), EDIT.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", EDIT.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), EDIT_PROJECT.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", EDIT_PROJECT.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), READ.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), READ_PRIVATE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ_PRIVATE.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), WRITE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", WRITE.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), DELETE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", DELETE.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), MERGE_WITHOUT_CHECK.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", MERGE_WITHOUT_CHECK.String(), OWNER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(OWNER.String(), MANAGE_COMMENTS.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", DELETE.String(), MANAGE_COMMENTS.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), READ_PRIVATE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ_PRIVATE.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), CREATE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", CREATE.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), EDIT.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", EDIT.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), DELETE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", DELETE.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), WRITE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", WRITE.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(MANAGER.String(), READ.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ.String(), MANAGER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(WRITER.String(), READ.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ.String(), WRITER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(WRITER.String(), WRITE.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", WRITE.String(), WRITER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(READER.String(), READ.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", READ.String(), READER.String(), err)
		return err
	}

	if _, err = securityEnforcer.AddNamedGroupingPolicy("g2", InnerSource, READ.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for %v projects. Error: %v", READ.String(), InnerSource, err)
		return err
	}

	if _, err = securityEnforcer.AddGroupingPolicy(TUZ.String(), OWNER.String()); err != nil {
		log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", OWNER.String(), TUZ.String(), err)
		return fmt.Errorf("add grouping policy: %w", err)
	}

	if setting.SourceControlCustomGroups.Enabled {
		err = syncPrivilegesFromConfig(ctx)
		if err != nil {
			log.Fatal("Error has occurred while %v", err)
		}
	}

	if err = securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving policy. Error: %v", err)
		return err
	}
	log.Info("Role model successful initialized")

	return nil
}
