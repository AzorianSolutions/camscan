package sm

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

const FirmwareModeOid = "1.3.6.1.2.1.1.1.0"
const MacAddressOid = "1.3.6.1.4.1.161.19.3.3.1.3.0"
const SNMPSiteNameOid = "1.3.6.1.2.1.1.5.0"
const SNMPSiteLocationOid = "1.3.6.1.2.1.1.6.0"
const SNMPSiteContactOid = "1.3.6.1.2.1.1.4.0"
const IfInErrorsOid = "1.3.6.1.2.1.2.2.1.14.1"
const IfOutErrorsOid = "1.3.6.1.2.1.2.2.1.20.1"
const FECCRCErrorOid = "1.3.6.1.4.1.161.19.3.3.1.223.0"
const FECRxFifoNoBufOid = "1.3.6.1.4.1.161.19.3.3.1.224.0"
const FECCarrierSenseLostOid = "1.3.6.1.4.1.161.19.3.3.1.229.0"
const FECNoCarrierOid = "1.3.6.1.4.1.161.19.3.3.1.230.0"

func ScanDevice(ctx context.Context, args interface{}, descriptor workers.JobDescriptor) (interface{}, error) {
	var record = descriptor.Metadata["record"].(device.SubscriberModule)
	argVal := args.(int)
	returnVal := argVal * 2
	timeout := time.Duration(1000000000 * descriptor.AppConfig.SnmpTimeoutSm)

	logging.Trace("Opening SNMP connection for subscriber module; "+
		"id: %v; nid: %v; mac: %s; ipv4: %s; ipv4int: %v; status: %v; timeout: %s;",
		record.Id, record.NetworkId, record.MacAddress, record.IPv4Address, record.IPv4AddressInt, record.Status,
		timeout)

	snmp := &gosnmp.GoSNMP{
		Target:    record.IPv4Address,
		Port:      161,
		Community: descriptor.AppConfig.SnmpSmCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
	}

	snmpError := snmp.Connect()

	if snmpError != nil {
		logging.Warning("Failed to open SNMP connection for subscriber module; ip: %s;", record.IPv4Address)
		return returnVal, nil
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			logging.Warning("Failed to close SNMP connection for subscriber module; ip: %s;", record.IPv4Address)
		}
	}(snmp.Conn)

	oids := []string{FirmwareModeOid, MacAddressOid, SNMPSiteNameOid, SNMPSiteLocationOid, SNMPSiteContactOid,
		IfInErrorsOid, IfOutErrorsOid, FECCRCErrorOid, FECRxFifoNoBufOid, FECCarrierSenseLostOid, FECNoCarrierOid}

	logging.Trace1("Querying SNMP service for subscriber module; ip: %s;", record.IPv4Address)

	snmpResult, snmpError := snmp.Get(oids)

	if snmpError != nil {
		logging.Warning("Failed to query SNMP service for subscriber module; ip: %s;", record.IPv4Address)
		return returnVal, nil
	}

	for _, variable := range snmpResult.Variables {
		// Cache a reference to the OID without the leading "."
		oid := variable.Name[1:]

		// Process OID value based on type
		if variable.Type == gosnmp.OctetString {
			value := strings.Trim(string(variable.Value.([]byte)), " ")

			// Firmware Version & Mode
			if oid == FirmwareModeOid {
				logging.Debug("Loaded firmware mode; ip: %s; value: %s;",
					record.IPv4Address, value)
			}

			// Primary MAC Address
			if oid == MacAddressOid {
				logging.Debug("Loaded primary MAC address; ip: %s; value: %s;",
					record.IPv4Address, value)
			}

			// SNMP Site Name
			if oid == SNMPSiteNameOid {
				logging.Debug("Loaded site name; ip: %s; value: %s;",
					record.IPv4Address, value)
			}

			// SNMP Site Location
			if oid == SNMPSiteLocationOid {
				logging.Debug("Loaded site location; ip: %s; value: %s;",
					record.IPv4Address, value)
			}

			// SNMP Site Contact
			if oid == SNMPSiteContactOid {
				logging.Debug("Loaded site contact; ip: %s; value: %s;",
					record.IPv4Address, value)
			}
		} else if variable.Type == gosnmp.Counter32 || variable.Type == gosnmp.Counter64 {
			value := variable.Value

			// Ethernet Interface In Errors
			if oid == IfInErrorsOid {
				logging.Debug("Loaded ethernet interface in errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}

			// Ethernet Interface Out Errors
			if oid == IfOutErrorsOid {
				logging.Debug("Loaded ethernet interface out errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}

			// FEC CRC Errors
			if oid == FECCRCErrorOid {
				logging.Debug("Loaded FEC CRC errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}

			// FEC RX Fifo No Buf Errors
			if oid == FECRxFifoNoBufOid {
				logging.Debug("Loaded FEC Receive Fifo No Buf errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}

			// FEC Carrier Sense Lost Errors
			if oid == FECCarrierSenseLostOid {
				logging.Debug("Loaded FEC carrier sense lost errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}

			// FEC No Carrier Errors
			if oid == FECNoCarrierOid {
				logging.Debug("Loaded FEC no carrier errors; ip: %s; value: %v;",
					record.IPv4Address, value)
			}
		} else if variable.Type == gosnmp.Null {
			logging.Trace1("Received SNMP value is nil for subscriber module; ip: %s; oid: %s;",
				record.IPv4Address, oid)
		} else {
			logging.Error("SNMP Exception - Unexpected value type for oid; ip: %s; oid: %s; type: %s;",
				record.IPv4Address, oid, variable.Type)
			logging.Trace(variable.Value.(string))
		}
	}

	return returnVal, nil
}
