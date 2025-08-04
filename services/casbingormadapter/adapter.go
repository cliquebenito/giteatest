package casbingormadapter

import (
	"database/sql"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"

	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// TODO: add recover to casbin adapter methods https://sberworks.ru/jira/browse/VCS-1351

// Adapter allows to store enforcer entities in postgres.
// Adapter is a wrapper over the "github.com/casbin/gorm-adapter/v3"
// for adding logging and panic handling.
type Adapter struct {
	innerAdapter *gormadapter.Adapter
}

// New allows to create new adapter.
func New(innerAdapter *gormadapter.Adapter) *Adapter {
	return &Adapter{innerAdapter: innerAdapter}
}

// LoadPolicy allows to load casbin policy.
func (a *Adapter) LoadPolicy(model model.Model) error {
	return a.innerAdapter.LoadPolicy(model)
}

// SavePolicy allows to save casbin policy.
func (a *Adapter) SavePolicy(model model.Model) error {
	return a.innerAdapter.SavePolicy(model)
}

// AddPolicy allows to add casbin policy.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return a.innerAdapter.AddPolicy(sec, ptype, rule)
}

// RemovePolicy allows to remove casbin policy.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return a.innerAdapter.RemovePolicy(sec, ptype, rule)
}

// RemoveFilteredPolicy allows to remove filtered casbin policy.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return a.innerAdapter.RemoveFilteredPolicy(sec, ptype, fieldIndex, fieldValues...)
}

// Transaction allows to run enforcer commands in one transaction.
func (a *Adapter) Transaction(e casbin.IEnforcer, fc func(casbin.IEnforcer) error, opts ...*sql.TxOptions) error {
	return a.innerAdapter.Transaction(e, fc, opts...)
}
