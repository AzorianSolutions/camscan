package ap

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types/device"
	"database/sql"
)

func GetRecords(db *sql.DB) (bool, []device.AccessPoint) {
	var records []device.AccessPoint
	var sqlQuery = `SELECT id, network_id, mac_address, ipv4_address, ipv4_address_int, status
					FROM device_access_point`

	sqlResults, sqlError := db.Query(sqlQuery)

	if sqlError != nil {
		logging.Error("Error retrieving access point records from database; error: %s;",
			sqlError.Error())
		return false, records
	} else {
		for sqlResults.Next() {
			var record device.AccessPoint
			_ = sqlResults.Scan(&record.Id, &record.NetworkId, &record.MacAddress, &record.IPv4Address,
				&record.IPv4AddressInt, &record.Status)

			records = append(records, record)

			logging.Trace1("Access point record loaded; "+
				"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v;",
				record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt,
				record.Status)
		}
	}

	return true, records
}

func UpsertRecord(db *sql.DB, record device.AccessPoint) (bool, device.AccessPoint) {
	sqlQuery := `INSERT INTO device_access_point(network_id, mac_address, ipv4_address, ipv4_address_int, status)
			     VALUES (?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE network_id=?, mac_address=?, ipv4_address=?, ipv4_address_int=?, status=?`

	insertStmt, sqlError := db.Prepare(sqlQuery)

	if sqlError != nil {
		logging.Error("Failed to create access point record; "+
			"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt,
			record.Status, sqlError.Error())
		return false, record
	}

	_, sqlError = insertStmt.Exec(
		record.NetworkId,
		record.MacAddress,
		record.IPv4Address,
		record.IPv4AddressInt,
		record.Status,
	)

	if sqlError != nil {
		logging.Error("Failed to create access point record; "+
			"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt,
			record.Status, sqlError.Error())
		return false, record
	}

	sqlError = insertStmt.Close()
	if sqlError != nil {
		logging.Warning("Failed to close MySQL prepared statement for access point; "+
			"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt,
			record.Status, sqlError.Error())
		return false, record
	}

	return true, record
}
