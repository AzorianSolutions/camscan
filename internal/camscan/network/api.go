package network

import (
	dbAp "as/camscan/internal/camscan/database/device/ap"
	dbSm "as/camscan/internal/camscan/database/device/sm"
	dbSubnet "as/camscan/internal/camscan/database/network/subnet"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types"
	"as/camscan/internal/camscan/types/device"
	"as/camscan/internal/camscan/types/network"
	"as/camscan/internal/camscan/workers"
	"context"
	"database/sql"
	"fmt"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/gosnmp/gosnmp"
	"github.com/praserx/ipconv"
	"github.com/prometheus-community/pro-bing"
	"net"
	"strconv"
	"strings"
	"time"
)

const FirmwareModeOid = "1.3.6.1.2.1.1.1.0"

func PingHost(appConfig types.AppConfig, host string) bool {
	alive := false
	pinger, err := probing.NewPinger(host)

	if err != nil {
		return false
	}

	pinger.Count = 1
	pinger.Timeout = time.Duration(1000000000 * appConfig.ICMPTimeout)
	pinger.Size = 24

	err = pinger.Run()

	if err != nil {
		logging.Error("ICMP Test Failed; host: %s; error: %s;", host, err.Error())
		return false
	}

	stats := pinger.Statistics()

	if stats.PacketsRecv > 0 {
		alive = true
	}

	msg := ""
	if alive == true {
		msg = "alive"
	} else {
		msg = "dead"
	}

	logging.Trace1("ICMP Test; host: %s; sent: %v; received: %v; lost: %v; status: %s;",
		host, stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss, msg)

	return alive
}

func QueryHost(appConfig types.AppConfig, host string, oid string) (bool, interface{}) {
	timeout := time.Duration(1000000000 * appConfig.SnmpTimeoutSm)

	snmp := &gosnmp.GoSNMP{
		Target:    host,
		Port:      161,
		Community: appConfig.SnmpSmCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
	}

	snmpError := snmp.Connect()

	if snmpError != nil {
		logging.Warning("Failed to open SNMP connection for device; ip: %s;", host)
		return false, nil
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			logging.Warning("Failed to close SNMP connection for device; ip: %s;", host)
		}
	}(snmp.Conn)

	oids := []string{oid}

	logging.Trace1("Querying SNMP service for device; ip: %s;", host)

	snmpResult, snmpError := snmp.Get(oids)

	if snmpError != nil {
		logging.Warning("Failed to query SNMP service for device; ip: %s; error: %s;", host, snmpError.Error())
		return false, nil
	}

	for _, variable := range snmpResult.Variables {
		// Cache a reference to the OID without the leading "."
		resultOid := variable.Name[1:]

		// Process OID value based on type
		if variable.Type == gosnmp.OctetString {
			value := strings.Trim(string(variable.Value.([]byte)), " ")

			// Firmware Version & Mode
			if resultOid == FirmwareModeOid {
				logging.Trace1("Loaded firmware mode; ip: %s; value: %s;",
					host, value)
				return true, value
			}
		} else {
			logging.Error("SNMP Exception - Unexpected value type for oid; ip: %s; oid: %s; type: %s;",
				host, resultOid, variable.Type)
		}
	}

	return false, nil
}

func CheckDevice(ctx context.Context, args interface{}, descriptor workers.JobDescriptor) (interface{}, error) {
	var record = descriptor.Metadata["record"].(network.Device)
	argVal := args.(int)
	mode := "unknown"
	returnVal := argVal * 2
	timeout := time.Duration(1000000000 * descriptor.AppConfig.ICMPTimeout)

	logging.Trace("Testing ICMP for device; "+
		"id: %v; nid: %v; sid: %v; ipv4: %s; ipv4int: %v; status: %v; timeout: %s;",
		record.Id, record.NetworkId, record.SubnetId, record.IPv4Address, record.IPv4AddressInt, record.Status,
		timeout)

	alive := PingHost(descriptor.AppConfig, record.IPv4Address)

	if alive == true {
		success, result := QueryHost(descriptor.AppConfig, record.IPv4Address, FirmwareModeOid)

		if success == true && result != nil {
			mode = result.(string)

			if strings.HasSuffix(mode, " AP") == true {
				mode = "ap"
			} else if strings.HasSuffix(mode, " SM") == true {
				mode = "sm"
			}
		}

		if success == true {
			logging.Trace1("SNMP Query Complete; host: %s; mode: %s;", record.IPv4Address, mode)
		} else {
			logging.Error("SNMP Query Failed; host: %s;", record.IPv4Address)
		}
	}

	if mode == "ap" {
		record := device.AccessPoint{
			NetworkId:      record.NetworkId,
			MacAddress:     "000000000000",
			IPv4Address:    record.IPv4Address,
			IPv4AddressInt: record.IPv4AddressInt,
			Status:         2,
		}

		dbAp.UpsertRecord(descriptor.Db, record)
	}

	if mode == "sm" {
		record := device.SubscriberModule{
			NetworkId:      record.NetworkId,
			MacAddress:     "000000000000",
			IPv4Address:    record.IPv4Address,
			IPv4AddressInt: record.IPv4AddressInt,
			Status:         2,
		}

		dbSm.UpsertRecord(descriptor.Db, record)
	}

	return returnVal, nil
}

func BuildDeviceCheckJobs(db *sql.DB, appConfig types.AppConfig, jobId int) (bool, int, []workers.Job) {
	jobs := make([]workers.Job, 0)
	success, subnets := dbSubnet.GetRecords(db)

	if jobId < 1 {
		jobId = 1
	}

	if success != true {
		return false, jobId, jobs
	}

	for _, el := range subnets {
		if el.Status < 1 {
			continue
		}

		_, networkIpv4, err := net.ParseCIDR(el.IPv4NetworkAddress + "/" + strconv.Itoa(el.IPv4NetworkMask))

		if err == nil {
			ipv4Start, ipv4End := cidr.AddressRange(networkIpv4)
			ipv4StartInt, _ := ipconv.IPv4ToInt(ipv4Start)
			ipv4EndInt, _ := ipconv.IPv4ToInt(ipv4End)

			for i := ipv4StartInt; i <= ipv4EndInt; i++ {
				ipv4Address := ipconv.IntToIPv4(i).String()

				networkDevice := network.Device{
					NetworkId:      el.NetworkId,
					SubnetId:       el.Id,
					IPv4Address:    ipv4Address,
					IPv4AddressInt: i,
					Status:         0,
				}

				metadata := make(map[string]interface{})
				metadata["record"] = networkDevice

				job := workers.Job{
					Descriptor: workers.JobDescriptor{
						ID:        workers.JobID(fmt.Sprintf("%v", jobId)),
						JType:     "icmp",
						AppConfig: appConfig,
						Metadata:  metadata,
						Db:        db,
					},
					ExecFn: CheckDevice,
					Args:   jobId,
				}

				logging.Trace("Building ICMP job for device (%v); nid: %v; sid: %v; ip: %s; ipInt: %v; status: %v;",
					job.Descriptor.ID, networkDevice.NetworkId, networkDevice.SubnetId, networkDevice.IPv4Address,
					networkDevice.IPv4AddressInt, el.Status)

				jobs = append(jobs, job)
				jobId++
			}
		}
	}

	return true, jobId, jobs
}
