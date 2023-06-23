package tasks

import (
	"as/camscan/internal/camscan/config"
	"as/camscan/internal/camscan/database"
	dbAp "as/camscan/internal/camscan/database/device/ap"
	dbSm "as/camscan/internal/camscan/database/device/sm"
	dbSubnet "as/camscan/internal/camscan/database/network/subnet"
	dbOm "as/camscan/internal/camscan/database/snmp/om"
	"as/camscan/internal/camscan/device/ap"
	"as/camscan/internal/camscan/device/sm"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types/device"
	"as/camscan/internal/camscan/types/network"
	"as/camscan/internal/camscan/types/snmp"
	"as/camscan/internal/camscan/workers"
	"context"
	"fmt"
)

// Setup device maps
var accessPointOidMaps []snmp.OidMap
var accessPointOids map[string]string
var accessPoints []device.AccessPoint
var subscriberModuleOidMaps []snmp.OidMap
var subscriberModuleOids map[string]string
var subscriberModules []device.SubscriberModule
var subnets []network.Subnet

// Define worker pool variables
var jobs = make([]workers.Job, 0)
var ctx context.Context
var wp workers.WorkerPool

var accessPointCSV = ""
var subscriberModuleCSV = ""

func ManageTasks() bool {
	select {
	case <-ctx.Done():
		// Handles the case where the context is canceled before the first task is executed
		//logging.Error("Context canceled before the first task was executed; error: %s;", ctx.Err().Error())
	case r, ok := <-wp.Results():
		// Handles the case where a task has finished executing
		if !ok {
			break
		}

		if r.Err != nil {
			logging.Error("Task failed to execute; id: %s; error: %s;", r.Descriptor.ID, r.Err.Error())
		}

		//logging.Debug("Task finished executing; id: %s;", r.Descriptor.ID)

		results := r.Value.(map[string]interface{})

		if r.Descriptor.JType == "ap" {
			for _, om := range accessPointOidMaps {
				result, ok := results[om.KeyName]
				if !ok {
					accessPointCSV += ","
					continue
				}
				accessPointCSV += fmt.Sprintf("%v,", result)
			}
			accessPointCSV = accessPointCSV[:len(accessPointCSV)-1] + "\n"
		}

		if r.Descriptor.JType == "sm" {
			for _, om := range subscriberModuleOidMaps {
				result, ok := results[om.KeyName]
				if !ok {
					subscriberModuleCSV += ","
					continue
				}
				subscriberModuleCSV += fmt.Sprintf("%v,", result)
			}
			subscriberModuleCSV = subscriberModuleCSV[:len(subscriberModuleCSV)-1] + "\n"
		}
	case <-wp.Done:
		// Handles the case where the worker pool has finished executing all tasks
		//logging.Warning("Worker pool has finished executing all tasks.")
		logging.Warning("Access Point CSV: %s;", accessPointCSV)
		logging.Warning("Subscriber Module CSV: %s;", subscriberModuleCSV)
		return false
	default:
		// Handles the case where the worker pool is still executing tasks
		//logging.Debug("Worker pool is still executing tasks.")
	}

	return true
}

func SetupTaskManager() bool {
	SetupWorkerPool()

	// Creates jobs in the queue
	SetupJobs()

	// Loads the job queue into the worker pool
	LoadJobs()

	// Signals the worker pool to begin execution of the job queue
	StartJobs()

	return true
}

func SetupWorkerPool() context.CancelFunc {
	logging.Debug("Setting up new worker pool with %v workers.", config.AppConfig.Workers)

	var cancel context.CancelFunc

	// Create worker pool
	wp = workers.New(config.AppConfig.Workers)

	ctx, cancel = context.WithCancel(context.TODO())

	return cancel
}

func SetupJobs() {

	logging.Info("Setting up jobs for workers...")

	logging.Debug("Loading existing inventory records from database...")

	// Synchronize changes from the database
	syncDatabase()

	db := database.GetConnection(database.ConnectionMap.CamScan)

	// dbSubnet.PopulateSubnets(db)

	// Build jobs queue for device ICMP checks
	jobId := 1
	// _, jobId, jobs = networkApi.BuildDeviceCheckJobs(db, appConfig, 1)

	_, accessPointOidMaps = dbOm.GetRecords(db, snmp.DeviceTypeAccessPoint)
	_, subscriberModuleOidMaps = dbOm.GetRecords(db, snmp.DeviceTypeSubscriberModule)

	accessPointOids = make(map[string]string)
	subscriberModuleOids = make(map[string]string)

	for _, el := range accessPointOidMaps {
		accessPointOids[el.KeyName] = el.Oid
	}

	for _, el := range subscriberModuleOidMaps {
		subscriberModuleOids[el.KeyName] = el.Oid
	}

	for key, _ := range accessPointOids {
		accessPointCSV += key + ","
	}

	for key, _ := range subscriberModuleOids {
		subscriberModuleCSV += key + ","
	}

	accessPointCSV = accessPointCSV[:len(accessPointCSV)-1] + "\n"
	subscriberModuleCSV = subscriberModuleCSV[:len(subscriberModuleCSV)-1] + "\n"

	// Load jobs queue with access points
	for _, el := range accessPoints {
		if el.Status < 1 {
			continue
		}

		metadata := make(map[string]interface{})
		metadata["record"] = el
		metadata["oids"] = accessPointOids

		job := workers.Job{
			Descriptor: workers.JobDescriptor{
				ID:        workers.JobID(fmt.Sprintf("%v", jobId)),
				JType:     "ap",
				AppConfig: config.AppConfig,
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
		metadata["oids"] = subscriberModuleOids

		job := workers.Job{
			Descriptor: workers.JobDescriptor{
				ID:        workers.JobID(fmt.Sprintf("%v", jobId)),
				JType:     "sm",
				AppConfig: config.AppConfig,
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
	logging.Debug("Starting %v workers to process SNMP queries.", config.AppConfig.Workers)
	go wp.Run(ctx)
}

func syncDatabase() bool {
	// Open a fresh database connection to ensure a smooth execution
	success, db := database.CreateConnection(database.ConnectionMap.CamScan, config.AppConfig.DbConfig)

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
