package subnet

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types/network"
	"database/sql"
	"fmt"
	"github.com/praserx/ipconv"
	"net"
	"strconv"
)

func PopulateSubnets(db *sql.DB) {
	for i := 1; i <= 52; i++ {
		mask := 24
		ipv4 := fmt.Sprintf("10.%v.2.0", i)
		ip, _, _ := ipconv.ParseIP(ipv4)
		ipv4Int, _ := ipconv.IPv4ToInt(ip)
		cidr := fmt.Sprintf("%s/%v", ipv4, mask)

		record := network.Subnet{
			NetworkId:             1,
			Cidr:                  cidr,
			IPv4NetworkAddress:    ipv4,
			IPv4NetworkAddressInt: ipv4Int,
			IPv4NetworkMask:       mask,
			Status:                1,
		}

		UpsertRecord(db, record)
	}

	for i := 1; i <= 52; i++ {
		mask := 22
		ipv4 := fmt.Sprintf("10.%v.148.0", i)
		var ipv4Int uint32 = 0

		ip, _, err := net.ParseCIDR(ipv4 + "/" + strconv.Itoa(mask))

		if err == nil {
			ipInt, err := ipconv.IPv4ToInt(ip)
			if err == nil {
				ipv4Int = ipInt
			}
		}

		record := network.Subnet{
			NetworkId:             1,
			Cidr:                  ipv4 + "/" + strconv.Itoa(mask),
			IPv4NetworkAddress:    ipv4,
			IPv4NetworkAddressInt: ipv4Int,
			IPv4NetworkMask:       mask,
			Status:                1,
		}

		UpsertRecord(db, record)
	}
}

func GetRecords(db *sql.DB) (bool, []network.Subnet) {
	var records []network.Subnet
	var sqlQuery = `SELECT id, network_id, cidr, ipv4_network_address, ipv4_network_address_int, ipv4_network_mask,
       				status
					FROM network_subnet`

	sqlResults, sqlError := db.Query(sqlQuery)

	if sqlError != nil {
		logging.Error("Error retrieving network subnet records from database; error: %s;",
			sqlError.Error())
		return false, records
	} else {
		for sqlResults.Next() {
			var record network.Subnet
			_ = sqlResults.Scan(&record.Id, &record.NetworkId, &record.Cidr, &record.IPv4NetworkAddress,
				&record.IPv4NetworkAddressInt, &record.IPv4NetworkMask, &record.Status)

			records = append(records, record)

			logging.Trace1("Network subnet record loaded; "+
				"id: %v; nid: %v; cidr: %s; ipv4: %s; ipv4int: %v; mask: %v; status: %v;",
				record.Id, record.NetworkId, record.Cidr, record.IPv4NetworkAddress, record.IPv4NetworkAddressInt,
				record.IPv4NetworkMask, record.Status)
		}
	}

	return true, records
}

func UpsertRecord(db *sql.DB, record network.Subnet) (bool, network.Subnet) {
	sqlQuery := `INSERT INTO network_subnet(network_id, cidr, ipv4_network_address, ipv4_network_address_int,
                           ipv4_network_mask, status)
			     VALUES (?, ?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE network_id=?, cidr=?, ipv4_network_address=?, ipv4_network_address_int=?,
				     ipv4_network_mask=?, status=?`

	insertStmt, sqlError := db.Prepare(sqlQuery)

	if sqlError != nil {
		logging.Error("Failed to create network subnet record; "+
			"id: %v; nid: %v; cidr: %s; ipv4: %s; ipv4int: %v; mask: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.Cidr, record.IPv4NetworkAddress, record.IPv4NetworkAddressInt,
			record.IPv4NetworkMask, record.Status, sqlError.Error())
		return false, record
	}

	_, sqlError = insertStmt.Exec(
		record.NetworkId,
		record.Cidr,
		record.IPv4NetworkAddress,
		record.IPv4NetworkAddressInt,
		record.IPv4NetworkMask,
		record.Status,
		record.NetworkId,
		record.Cidr,
		record.IPv4NetworkAddress,
		record.IPv4NetworkAddressInt,
		record.IPv4NetworkMask,
		record.Status,
	)

	if sqlError != nil {
		logging.Error("Failed to create network subnet record; "+
			"id: %v; nid: %v; cidr: %s; ipv4: %s; ipv4int: %v; mask: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.Cidr, record.IPv4NetworkAddress, record.IPv4NetworkAddressInt,
			record.IPv4NetworkMask, record.Status, sqlError.Error())
		return false, record
	}

	sqlError = insertStmt.Close()
	if sqlError != nil {
		logging.Warning("Failed to close MySQL prepared statement for network subnet; "+
			"id: %v; nid: %v; cidr: %s; ipv4: %s; ipv4int: %v; mask: %v; status: %v; error: %s;",
			record.Id, record.NetworkId, record.Cidr, record.IPv4NetworkAddress, record.IPv4NetworkAddressInt,
			record.IPv4NetworkMask, record.Status, sqlError.Error())
		return false, record
	}

	return true, record
}
