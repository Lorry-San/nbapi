package model

import (
	"sync"
	"testing"

	"github.com/Lorry-San/nbapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

// SQLite upgrades subscription_plans from a hand-written column list, so a field added to
// SubscriptionPlan but not to that list works in tests (AutoMigrate) and fails in production
// with "no such column". Every stored field must be added to an older table.
func TestSQLitePlanTableUpgradeAddsEveryModelColumn(t *testing.T) {
	if !common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		t.Skip("the hand-written column list only serves SQLite")
	}
	require.NoError(t, DB.Migrator().DropTable(&SubscriptionPlan{}))
	require.NoError(t, DB.Exec("CREATE TABLE `subscription_plans` (`id` integer, `title` varchar(128) NOT NULL, `price_amount` decimal(10,6) NOT NULL, PRIMARY KEY (`id`))").Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Migrator().DropTable("subscription_plans"))
		require.NoError(t, DB.AutoMigrate(&SubscriptionPlan{}))
	})

	require.NoError(t, ensureSubscriptionPlanTableSQLite())

	parsed, err := schema.Parse(&SubscriptionPlan{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)
	var missing []string
	for _, field := range parsed.Fields {
		if field.DBName != "" && !DB.Migrator().HasColumn("subscription_plans", field.DBName) {
			missing = append(missing, field.DBName)
		}
	}
	assert.Empty(t, missing, "add these columns to ensureSubscriptionPlanTableSQLite")
}
