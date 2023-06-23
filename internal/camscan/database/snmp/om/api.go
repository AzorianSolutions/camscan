package om

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types/snmp"
	"database/sql"
)

func GetRecords(db *sql.DB, deviceType int) (bool, []snmp.OidMap) {
	var records []snmp.OidMap
	var sqlQuery = `SELECT som.id, som.device_type, som.key_name, som.oid, som.order
					FROM snmp_oid_map som
					WHERE som.device_type = ?
					ORDER BY som.order`

	sqlResults, sqlError := db.Query(sqlQuery, deviceType)

	if sqlError != nil {
		logging.Error("Error retrieving SNMP OID map records from database; error: %s;",
			sqlError.Error())
		return false, records
	} else {
		for sqlResults.Next() {
			var record snmp.OidMap
			_ = sqlResults.Scan(&record.Id, &record.DeviceType, &record.KeyName, &record.Oid, &record.Order)

			records = append(records, record)

			logging.Trace1("SNMP OID map record loaded; "+
				"id: %v; type: %s; key: %s; oid: %s; order: %v;",
				record.Id, record.DeviceType, record.KeyName, record.Oid, record.Order)
		}
	}

	return true, records
}
