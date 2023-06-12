package device

import (
	"as/camscan/internal/camscan/logging"
	"database/sql"
)

func HasRecordByField(db *sql.DB, table string, field string, value interface{}) bool {
	found := false
	total := 0

	sqlQuery := `SELECT count(*) AS total
				 FROM ?
				 WHERE ? = ?;`

	// Execute the query against the database
	row := db.QueryRow(sqlQuery, table, field, value)

	// Attempt to load the resulting row into the LLDPInventory record object
	sqlError := row.Scan(&total)

	// Handle any SQL errors and respond accordingly
	switch sqlError {
	case nil:
		logging.Trace1("Executed record count query; table: %s; field: %s; value: %s;", table, field, value)
		if total > 0 {
			found = true
		}
	case sql.ErrNoRows:
		logging.Error("Failed to execute record count query; table: %s; field: %s; value: %s;",
			table, field, value)
		return found
	default:
		logging.Error("Failed to execute record count query; table: %s; field: %s; value: %s; error: %s;",
			table, field, value, sqlError.Error())
		return found
	}

	return found
}
