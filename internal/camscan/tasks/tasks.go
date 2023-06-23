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
	"encoding/csv"
	"fmt"
	"os"
)

var accessPointOidMaps []snmp.OidMap
var accessPointOids map[string]string
var accessPointResults []map[string]interface{}
var accessPoints []device.AccessPoint
var subscriberModuleOidMaps []snmp.OidMap
var subscriberModuleOids map[string]string
var subscriberModuleResults []map[string]interface{}
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

		// Process the task result data
		deviceResult := r.Value.(map[string]interface{})

		// If the task was for an access point device
		if r.Descriptor.JType == "ap" {
			accessPointResults = append(accessPointResults, deviceResult)
		}

		// If the task was for a subscriber module
		if r.Descriptor.JType == "sm" {
			subscriberModuleResults = append(subscriberModuleResults, deviceResult)
		}
	case <-wp.Done:
		// Handles the case where the worker pool has finished executing all tasks
		//logging.Warning("Worker pool has finished executing all tasks.")
		if !createCSVExport() {
			logging.Error("Failed to create CSV exports.")
		}
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

func createCSVExport() bool {
	// Setup CSV headers for each device type

	// Access Point Headers
	apHeader := make([]string, 0)
	apRows := make([][]string, 0)
	for _, om := range accessPointOidMaps {
		apHeader = append(apHeader, om.KeyName)
	}
	apRows = append(apRows, apHeader)

	// Subscriber Module Headers
	smHeader := make([]string, 0)
	smRows := make([][]string, 0)
	for _, om := range subscriberModuleOidMaps {
		smHeader = append(smHeader, om.KeyName)
	}
	smRows = append(smRows, smHeader)

	// Process Access Point Results
	for _, accessPointResult := range accessPointResults {
		apRow := make([]string, 0)
		for _, om := range accessPointOidMaps {
			result, ok := accessPointResult[om.KeyName]
			if !ok {
				apRow = append(apRow, "")
				continue
			}
			apRow = append(apRow, fmt.Sprintf("%v", result))
		}
		apRows = append(apRows, apRow)
	}

	// Process Subscriber Module Results
	for _, subscriberModuleResult := range subscriberModuleResults {
		smRow := make([]string, 0)
		for _, om := range subscriberModuleOidMaps {
			result, ok := subscriberModuleResult[om.KeyName]
			if !ok {
				smRow = append(smRow, "")
				continue
			}
			smRow = append(smRow, fmt.Sprintf("%v", result))
		}
		smRows = append(smRows, smRow)
	}

	failed := false
	apFilePath := "/tmp/ap.csv"
	smFilePath := "/tmp/sm.csv"

	apFile, err := os.Create(apFilePath)

	if err != nil {
		failed = true
		logging.Error("Failed to create Access Point CSV file; path: %s; error: %s;", apFilePath, err.Error())
	} else {
		defer func(apFile *os.File) {
			err := apFile.Close()
			if err != nil {

			}
		}(apFile)
	}

	smFile, err := os.Create(smFilePath)

	if err != nil {
		failed = true
		logging.Error("Failed to create Subscriber Module CSV file; path: %s; error: %s;",
			smFilePath, err.Error())
	} else {
		defer func(smFile *os.File) {
			err := smFile.Close()
			if err != nil {

			}
		}(smFile)
	}

	apWriter := csv.NewWriter(apFile)
	smWriter := csv.NewWriter(smFile)

	for _, row := range apRows {
		err := apWriter.Write(row)
		if err != nil {
			failed = true
			logging.Error("Failed to write Access Point CSV row; error: %s;", err.Error())
		}
	}

	for _, row := range smRows {
		err := smWriter.Write(row)
		if err != nil {
			failed = true
			logging.Error("Failed to write Subscriber Module CSV row; error: %s;", err.Error())
		}
	}

	apWriter.Flush()
	smWriter.Flush()

	return !failed
}
