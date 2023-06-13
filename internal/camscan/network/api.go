package network

import (
	dbSubnet "as/camscan/internal/camscan/database/network/subnet"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types"
	"as/camscan/internal/camscan/types/network"
	"as/camscan/internal/camscan/workers"
	"context"
	"database/sql"
	"fmt"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/praserx/ipconv"
	"github.com/prometheus-community/pro-bing"
	"net"
	"strconv"
	"time"
)

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

func CheckDevice(ctx context.Context, args interface{}, descriptor workers.JobDescriptor) (interface{}, error) {
	var record = descriptor.Metadata["record"].(network.Device)
	argVal := args.(int)
	returnVal := argVal * 2
	timeout := time.Duration(1000000000 * descriptor.AppConfig.ICMPTimeout)

	logging.Trace("Testing ICMP for device; "+
		"id: %v; nid: %v; sid: %v; ipv4: %s; ipv4int: %v; status: %v; timeout: %s;",
		record.Id, record.NetworkId, record.SubnetId, record.IPv4Address, record.IPv4AddressInt, record.Status,
		timeout)

	PingHost(descriptor.AppConfig, record.IPv4Address)

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
