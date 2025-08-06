package repo

import (
	"context"
	"fmt"
	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
)

func init() {
	db.RegisterModel(new(CodeOwnersSettings))
}

type CodeOwnersSettings struct {
	ID             int64 `xorm:"PK AUTOINCR"`
	RepositoryID   int64 `xorm:"BIGINT NOT NULL UNIQUE"`
	ApprovalStatus bool  `xorm:"BOOL NOT NULL"`
	AmountUsers    int64 `xorm:"BIGINT NOT NULL"`
}

func GetCodeOwnersSettings(ctx context.Context, repoID int64) (CodeOwnersSettings, error) {
	owners := &CodeOwnersSettings{}
	has, err := db.GetEngine(ctx).
		Where(builder.Eq{"repository_id": repoID}).
		Get(owners)
	if err != nil {
		return CodeOwnersSettings{}, fmt.Errorf("get code owners settings: %w", err)
	}
	if !has {
		return CodeOwnersSettings{}, nil
	}
	return *owners, nil
}

// InsertCodeOwners добавление codeowners
func InsertCodeOwners(ctx context.Context, codeOwners *CodeOwnersSettings) error {
	err := db.Insert(ctx, codeOwners)
	if err != nil {
		return fmt.Errorf("insert code owners: %w", err)
	}
	return nil
}

// UpdateCodeOwners обновление количества аппруверов
func UpdateCodeOwners(ctx context.Context, codeOwners *CodeOwnersSettings) error {
	_, err := db.GetEngine(ctx).Where(builder.Eq{"repository_id": codeOwners.RepositoryID}).Cols("amount_users").Update(codeOwners)
	if err != nil {
		return fmt.Errorf("update code owners: %w", err)
	}
	return nil
}

// DeleteCodeOwners удаление codeowners
func DeleteCodeOwners(ctx context.Context, codeOwners *CodeOwnersSettings) error {
	_, err := db.GetEngine(ctx).Delete(codeOwners)
	if err != nil {
		return fmt.Errorf("delete code owners: %w", err)
	}
	return nil
}
