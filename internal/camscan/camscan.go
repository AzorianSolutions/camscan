package main

import (
	"as/camscan/internal/camscan/config"
	"as/camscan/internal/camscan/database"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/tasks"
	"flag"
	"fmt"
	"os"
)

const DbConnectionName = "main"

func main() {
	debug := false
	dryRun := false
	workers := 0

	// Define application arguments and allow for override of database environment settings
	flag.BoolVar(&debug, "debug", debug, "Determines whether debug mode is enabled.")
	flag.BoolVar(&dryRun, "dry-run", dryRun, "Determines whether dry-run mode is enabled.")
	flag.IntVar(&workers, "workers", workers, "Defines the number of workers to create.")
	flag.Parse()

	// Load application settings from environment into structured configuration
	appConfig := config.CreateAppConfig(database.CreateConfigFromEnvironment(), workers, dryRun, debug)

	// Configure the logging API
	logging.SetLogLevel(appConfig.LogLevel)

	fmt.Printf("Log Level: %v;", logging.GetLogLevel())

	logging.Info("Starting the CamScan main process...")

	// Open a fresh database connection to ensure a smooth execution
	opened, db := database.OpenConnection(DbConnectionName, appConfig.DbConfig)

	if opened == false {
		logging.Critical("Could not open a connection to the database server!")
		os.Exit(1)
	}

	// Set up the task management system
	tasks.SetAppConfig(appConfig)
	tasks.SetDb(db)

	cancel := tasks.SetupWorkerPool()
	defer cancel()

	// Creates jobs in the queue
	tasks.SetupJobs()

	// Loads the job queue into the worker pool
	tasks.LoadJobs()

	// Signals the worker pool to begin execution of the job queue
	tasks.StartJobs()

	// Wait for all jobs to be completed while monitoring the workers for communications
	tasks.MonitorJobs()

	// Close & remove the database connection
	if database.CloseConnection(DbConnectionName) == true {
		database.RemoveConnection(DbConnectionName)
	}

	logging.Info("CamScan has completed the execution cycle.")
}
