package tasks

import (
	"as/camscan/internal/camscan/database"
	dbAp "as/camscan/internal/camscan/database/device/ap"
	dbSm "as/camscan/internal/camscan/database/device/sm"
	dbSubnet "as/camscan/internal/camscan/database/network/subnet"
	"as/camscan/internal/camscan/device/ap"
	"as/camscan/internal/camscan/device/sm"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types"
	"as/camscan/internal/camscan/types/device"
	"as/camscan/internal/camscan/types/network"
	"as/camscan/internal/camscan/workers"
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

var appConfig types.AppConfig
var db *sql.DB

// Setup device maps
var accessPoints []device.AccessPoint
var subscriberModules []device.SubscriberModule
var subnets []network.Subnet

// Define worker pool variables
var jobs = make([]workers.Job, 0)
var ctx context.Context
var wp workers.WorkerPool

func SetAppConfig(config types.AppConfig) {
	appConfig = config
}

func SetDb(connection *sql.DB) {
	db = connection
}

func SetupWorkerPool() context.CancelFunc {
	logging.Debug("Setting up new worker pool with %v workers.", appConfig.Workers)

	var cancel context.CancelFunc

	// Create worker pool
	wp = workers.New(appConfig.Workers)

	ctx, cancel = context.WithCancel(context.TODO())

	return cancel
}

func SetupJobs() {

	logging.Info("Setting up jobs for workers...")

	logging.Debug("Loading existing inventory records from database...")

	// Synchronize changes from the database
	syncDatabase(db, appConfig)

	// Build jobs queue for device ICMP checks
	jobId := 1
	// _, jobId, jobs = networkApi.BuildDeviceCheckJobs(db, appConfig, 1)

	// Load jobs queue with access points
	for _, el := range accessPoints {
		if el.Status < 1 {
			continue
		}

		metadata := make(map[string]interface{})
		metadata["record"] = el

		job := workers.Job{
			Descriptor: workers.JobDescriptor{
				ID:        workers.JobID(fmt.Sprintf("%v", jobId)),
				JType:     "ap",
				AppConfig: appConfig,
				Metadata:  metadata,
				Db:        db,
			},
			ExecFn: ap.ScanDevice,
			Args:   jobId,
		}

		logging.Debug("Queueing job for ap (%v); id: %v; nid: %v; mac: %s; ip: %s; status: %v;",
			job.Descriptor.ID, el.Id, el.NetworkId, el.MacAddress, el.IPv4Address, el.Status)

		jobs = append(jobs, job)
		jobId++
	}

	// Load jobs queue with subscriber modules
	for _, el := range subscriberModules {
		if el.Status < 1 {
			continue
		}

		metadata := make(map[string]interface{})
		metadata["record"] = el

		job := workers.Job{
			Descriptor: workers.JobDescriptor{
				ID:        workers.JobID(fmt.Sprintf("%v", jobId)),
				JType:     "sm",
				AppConfig: appConfig,
				Metadata:  metadata,
				Db:        db,
			},
			ExecFn: sm.ScanDevice,
			Args:   jobId,
		}

		logging.Debug("Queueing job for sm (%v); id: %v; nid: %v; mac: %s; ip: %s; status: %v;",
			job.Descriptor.ID, el.Id, el.NetworkId, el.MacAddress, el.IPv4Address, el.Status)

		jobs = append(jobs, job)
		jobId++
	}
}

func LoadJobs() {
	logging.Debug("Loading %v jobs into the worker pool task queue.", len(jobs))
	go wp.GenerateFrom(jobs)
}

func StartJobs() {
	logging.Debug("Starting %v workers to process SNMP queries.", appConfig.Workers)
	go wp.Run(ctx)
}

func MonitorJobs() {
	for {
		stop := false
		select {
		case r, ok := <-wp.Results():
			if !ok {
				continue
			}

			i, err := strconv.ParseInt(string(r.Descriptor.ID), 10, 64)
			if err != nil {
				logging.Error("unexpected error: %v", err)
			}

			val := r.Value.(int)
			if val != int(i)*2 {
				logging.Error("wrong value %v; expected %v", val, int(i)*2)
			}
		case <-wp.Done:
			stop = true
		default:
		}

		if stop == true {
			break
		}
	}
}

func syncDatabase(db *sql.DB, appConfig types.AppConfig) bool {
	var success = false

	// Open a fresh database connection to ensure a smooth execution
	success, db = database.CreateConnection("main", appConfig.DbConfig)

	// Handle any exceptions that may have occurred when attempting to open the database connection
	if success != true {
		return false
	}

	success, subnets = dbSubnet.GetRecords(db)

	if success != true {
		return false
	}

	success, accessPoints = dbAp.GetRecords(db)

	if success != true {
		return false
	}

	success, subscriberModules = dbSm.GetRecords(db)

	if success != true {
		return false
	}

	return true
}
