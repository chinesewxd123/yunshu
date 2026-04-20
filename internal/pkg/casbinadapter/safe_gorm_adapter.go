package casbinadapter

import (
	"strings"

	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

// CasbinRule mirrors the schema used by casbin gorm-adapter.
type CasbinRule struct {
	ID    uint   `gorm:"primaryKey"`
	PType string `gorm:"size:100;column:ptype"`
	V0    string `gorm:"size:1000;column:v0"`
	V1    string `gorm:"size:1000;column:v1"`
	V2    string `gorm:"size:1000;column:v2"`
	V3    string `gorm:"size:1000;column:v3"`
	V4    string `gorm:"size:1000;column:v4"`
	V5    string `gorm:"size:1000;column:v5"`
}

// SafeGormAdapter is a small Casbin adapter that loads policies from the
// casbin_rule table but avoids panicking on malformed rows.
type SafeGormAdapter struct {
	db        *gorm.DB
	tableName string
}

func NewSafeGormAdapter(db *gorm.DB, tableName string) *SafeGormAdapter {
	if strings.TrimSpace(tableName) == "" {
		tableName = "casbin_rule"
	}
	return &SafeGormAdapter{db: db, tableName: tableName}
}

func (a *SafeGormAdapter) LoadPolicy(m model.Model) error {
	var lines []CasbinRule
	if err := a.db.Table(a.tableName).Find(&lines).Error; err != nil {
		return err
	}

	for _, line := range lines {
		ptype := strings.TrimSpace(line.PType)
		if ptype == "" {
			continue
		}
		// v0..v5: include up to last non-empty value.
		vs := []string{line.V0, line.V1, line.V2, line.V3, line.V4, line.V5}
		last := -1
		for i := len(vs) - 1; i >= 0; i-- {
			if strings.TrimSpace(vs[i]) != "" {
				last = i
				break
			}
		}
		if last < 0 {
			continue
		}

		sec := ptype[:1]
		ast, ok := m[sec][ptype]
		if !ok {
			// Model doesn't define this policy type; skip silently.
			continue
		}
		rule := make([]string, 0, last+1)
		for i := 0; i <= last; i++ {
			rule = append(rule, strings.TrimSpace(vs[i]))
		}

		ast.Policy = append(ast.Policy, rule)
		ast.PolicyMap[strings.Join(rule, model.DefaultSep)] = len(ast.Policy) - 1
	}
	return nil
}

func (a *SafeGormAdapter) SavePolicy(m model.Model) error {
	// simplest: wipe and re-insert
	if err := a.db.Table(a.tableName).Where("1 = 1").Delete(&CasbinRule{}).Error; err != nil {
		return err
	}
	var rules []CasbinRule
	for ptype, ast := range m["p"] {
		for _, rule := range ast.Policy {
			rules = append(rules, toCasbinRule(ptype, rule))
		}
	}
	for ptype, ast := range m["g"] {
		for _, rule := range ast.Policy {
			rules = append(rules, toCasbinRule(ptype, rule))
		}
	}
	if len(rules) == 0 {
		return nil
	}
	return a.db.Table(a.tableName).Create(&rules).Error
}

func (a *SafeGormAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	row := toCasbinRule(ptype, rule)
	return a.db.Table(a.tableName).Create(&row).Error
}

func (a *SafeGormAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	q := a.db.Table(a.tableName).Where("ptype = ?", ptype)
	q = applyRuleWhere(q, 0, rule)
	return q.Delete(&CasbinRule{}).Error
}

func (a *SafeGormAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	q := a.db.Table(a.tableName).Where("ptype = ?", ptype)
	q = applyRuleWhere(q, fieldIndex, fieldValues)
	return q.Delete(&CasbinRule{}).Error
}

func toCasbinRule(ptype string, rule []string) CasbinRule {
	out := CasbinRule{PType: ptype}
	vs := []string{"", "", "", "", "", ""}
	for i := 0; i < len(rule) && i < 6; i++ {
		vs[i] = rule[i]
	}
	out.V0, out.V1, out.V2, out.V3, out.V4, out.V5 = vs[0], vs[1], vs[2], vs[3], vs[4], vs[5]
	return out
}

func applyRuleWhere(q *gorm.DB, fieldIndex int, fieldValues []string) *gorm.DB {
	cols := []string{"v0", "v1", "v2", "v3", "v4", "v5"}
	for i, v := range fieldValues {
		colIdx := fieldIndex + i
		if colIdx < 0 || colIdx >= len(cols) {
			continue
		}
		if strings.TrimSpace(v) == "" {
			continue
		}
		q = q.Where(cols[colIdx]+" = ?", v)
	}
	return q
}

