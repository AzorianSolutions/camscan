package ap

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types/device"
	"as/camscan/internal/camscan/workers"
	"context"
	"github.com/gosnmp/gosnmp"
	"net"
	"strings"
	"time"
)

func ScanDevice(ctx context.Context, args interface{}, descriptor workers.JobDescriptor) (interface{}, error) {
	var record = descriptor.Metadata["record"].(device.AccessPoint)
	results := make(map[string]interface{})
	timeout := time.Duration(1000000000 * descriptor.AppConfig.SnmpTimeoutSm)

	logging.Trace("Opening SNMP connection for access point; "+
		"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v; timeout: %s;",
		record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt, record.Status,
		timeout)

	snmp := &gosnmp.GoSNMP{
		Target:    record.IPv4Address,
		Port:      161,
		Community: descriptor.AppConfig.SnmpApCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
	}

	snmpError := snmp.Connect()

	if snmpError != nil {
		logging.Warning("Failed to open SNMP connection for access point; ip: %s;", record.IPv4Address)
		return results, nil
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			logging.Warning("Failed to close SNMP connection for access point; ip: %s;", record.IPv4Address)
		}
	}(snmp.Conn)

	oids := make([]string, 0)
	oidMap := make(map[string]string)

	for key, oid := range descriptor.Metadata["oids"].(map[string]string) {
		oids = append(oids, oid)
		oidMap[oid] = key
	}

	logging.Trace1("Querying SNMP service for access point; ip: %s;", record.IPv4Address)

	snmpResult, snmpError := snmp.Get(oids)

	if snmpError != nil {
		logging.Warning("Failed to query SNMP service for access point; ip: %s;", record.IPv4Address)
		return results, nil
	}

	for _, variable := range snmpResult.Variables {
		// Cache a reference to the OID without the leading "."
		oid := variable.Name[1:]

		key, ok := oidMap[oid]

		if !ok {
			logging.Warning("Failed to find OID key in map; ip: %s; oid: %s;", record.IPv4Address, oid)
			continue
		}

		// Process OID value based on type
		if variable.Type == gosnmp.OctetString {
			value := strings.Trim(string(variable.Value.([]byte)), " ")
			results[key] = value

			logging.Trace2("Loaded string value; ip: %s; oid: %s; value: %s;", record.IPv4Address, oid, value)
		} else if variable.Type == gosnmp.Counter32 || variable.Type == gosnmp.Counter64 {
			value := variable.Value
			results[key] = value

			logging.Trace2("Loaded counter value; ip: %s; oid: %s; value: %v;", record.IPv4Address, oid, value)
		} else if variable.Type == gosnmp.Null {
			logging.Trace1("Received SNMP value is nil for access point; ip: %s; oid: %s;",
				record.IPv4Address, oid)
		} else {
			logging.Error("SNMP Exception - Unexpected value type for oid; ip: %s; oid: %s; type: %s;",
				record.IPv4Address, oid, variable.Type)
			logging.Trace(variable.Value.(string))
		}
	}

	return results, nil
}
