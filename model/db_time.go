package model

import (
	"github.com/Lorry-San/nbapi/common"
	"gorm.io/gorm"
)

// GetDBTimestamp returns a UNIX timestamp from database time.
// Falls back to application time on error.
func GetDBTimestamp() int64 { return getDBTimestampTx(DB) }

// getDBTimestampTx reads time on the caller's connection, required inside a transaction.
func getDBTimestampTx(db *gorm.DB) int64 {
	var ts int64
	var err error
	switch {
	case common.UsingMainDatabase(common.DatabaseTypePostgreSQL):
		err = db.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingMainDatabase(common.DatabaseTypeSQLite):
		err = db.Raw("SELECT strftime('%s','now')").Scan(&ts).Error
	default:
		err = db.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
