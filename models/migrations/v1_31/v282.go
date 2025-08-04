package v1_31

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/external_metric_counter"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/models/migrations/base"
	"code.gitea.io/gitea/modules/log"
	"xorm.io/xorm"
)

// ChangeBranchPermissions функция для изменения таблицы protected_branch
func ChangeProtectedBranch(x *xorm.Engine) error {
	// проверяем наличие столбца can_push
	columnNamesToDelete := make([]string, 0, 4)

	canPushExists, err := x.Dialect().IsColumnExist(x.DB(), context.Background(), "protected_branch", "can_push")
	if err != nil {
		log.Error("Error has occured while checking columns is exist")
		return fmt.Errorf("Change protected branch: %w", err)
	}
	if canPushExists {
		columnNamesToDelete = append(columnNamesToDelete, "can_push")
	}

	// проверяем наличие столбца whitelist_team_i_ds
	whitelistTeamIDsExists, err := x.Dialect().IsColumnExist(x.DB(), context.Background(), "protected_branch", "whitelist_team_i_ds")
	if err != nil {
		log.Error("Error has occured while checking columns is exist")
		return fmt.Errorf("Change protected branch: %w", err)
	}
	if whitelistTeamIDsExists {
		columnNamesToDelete = append(columnNamesToDelete, "whitelist_team_i_ds")
	}

	// проверяем наличие столбца merge_whitelist_team_i_ds
	mergeWhitelistTeamIDsExists, err := x.Dialect().IsColumnExist(x.DB(), context.Background(), "protected_branch", "merge_whitelist_team_i_ds")
	if err != nil {
		log.Error("Error has occured while checking columns is exist")
		return fmt.Errorf("Change protected branch: %w", err)
	}
	if mergeWhitelistTeamIDsExists {
		columnNamesToDelete = append(columnNamesToDelete, "merge_whitelist_team_i_ds")
	}

	// проверяем наличие столбца approvals_whitelist_team_i_ds
	approvalsWhitelistTeamIDs, err := x.Dialect().IsColumnExist(x.DB(), context.Background(), "protected_branch", "approvals_whitelist_team_i_ds")
	if err != nil {
		log.Error("Error has occured while checking columns is exist")
		return fmt.Errorf("Change protected branch: %w", err)
	}
	if approvalsWhitelistTeamIDs {
		columnNamesToDelete = append(columnNamesToDelete, "approvals_whitelist_team_i_ds")
	}

	// Удаляем старые столбцы
	sess := x.NewSession()
	defer sess.Close()

	if err := base.DropTableColumns(sess, "protected_branch", columnNamesToDelete...); err != nil {
		return fmt.Errorf("unable to drop old credentialID column: %w", err)
	}

	query := "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS enable_deleter_whitelist BOOLEAN DEFAULT FALSE NOT NULL"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column enable_deleter_whitelist via constraints: %v", err)
	}

	query = "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS deleter_whitelist_user_i_ds TEXT"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column deleter_whitelist_user_i_ds via constraints: %v", err)
	}

	query = "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS deleter_whitelist_deploy_keys BOOLEAN DEFAULT FALSE NOT NULL"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column deleter_whitelist_deploy_keys via constraints: %v", err)
	}

	query = "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS enable_force_push_whitelist BOOLEAN DEFAULT FALSE NOT NULL"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column enable_force_push_whitelist via constraints: %v", err)
	}

	query = "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS force_push_whitelist_user_i_ds TEXT"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column force_push_whitelist_user_i_ds via constraints: %v", err)
	}

	query = "ALTER TABLE protected_branch ADD COLUMN IF NOT EXISTS force_push_whitelist_deploy_keys BOOLEAN DEFAULT FALSE NOT NULL"
	_, err = sess.Exec(query)
	if err != nil {
		return fmt.Errorf("Adding column force_push_whitelist_deploy_keys via constraints: %v", err)
	}

	return sess.Commit()
}
func ChangeCodeHubCounterTable(x *xorm.Engine) error {
	defaultKey := "unique_clones"
	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Begin(); err != nil {
		return fmt.Errorf("failed to begin session: %w", err)
	}

	if err := x.Sync(new(internal_metric_counter.InternalMetricCounter)); err != nil {
		return fmt.Errorf("failed to sync InternalMetricCounter model: %w", err)
	}

	if _, err := sess.Exec("ALTER TABLE `code_hub_counter` RENAME COLUMN num_uniq_usages TO metric_value"); err != nil {
		return fmt.Errorf("failed to rename column: %w", err)
	}
	if _, err := sess.Exec("ALTER TABLE `code_hub_counter` ADD COLUMN metric_key VARCHAR(255)"); err != nil {
		return fmt.Errorf("failed to add column metric_key: %w", err)
	}
	if _, err := sess.Exec(fmt.Sprintf("UPDATE `code_hub_counter` SET metric_key = '%s'", defaultKey)); err != nil {
		return fmt.Errorf("failed to update metric_key with default value: %w", err)
	}

	exists, err := x.Dialect().IsTableExist(x.DB(), context.Background(), "internal_metric_counter")
	if err != nil {
		return fmt.Errorf("failed to check if internal_metric_counter table exists: %w", err)
	}

	if exists {
		if _, err := sess.Exec("INSERT INTO internal_metric_counter (repo_id, metric_value, metric_key, updated_at, created_at) " +
			"SELECT repo_id, metric_value, metric_key, updated_at, created_at FROM code_hub_counter;"); err != nil {
			return fmt.Errorf("failed to duplicate data in internal_metric_counter: %w", err)
		}
	} else {
		sess.Rollback()
		return fmt.Errorf("internal_metric_counter table doesn't exist")
	}

	if err := sess.DropTable("code_hub_counter"); err != nil {
		return fmt.Errorf("failed to drop code_hub_counter table: %w", err)
	}

	if err := sess.Commit(); err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}

	return nil
}

func CreateExternalCounterTable(x *xorm.Engine) error {
	if err := x.Sync(new(external_metric_counter.ExternalMetricCounter)); err != nil {
		return fmt.Errorf("failed to sync ExternalMetricCounter model: %w", err)
	}
	return nil
}
