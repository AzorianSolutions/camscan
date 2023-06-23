package main

import (
	"as/camscan/internal/camscan/config"
	"as/camscan/internal/camscan/database"
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/tasks"
	"flag"
)

var debug = false
var dryRun = false
var initialized = false
var workers = 0

func main() {
	// Initialize program on first cycle execution
	if initialized == false {
		logging.Info("Initializing CamScan...")
		initialize()
	}

	for {
		// Execute management process for task manager
		if !tasks.ManageTasks() {
			logging.Info("CamScan has finished executing.")
			break
		}
	}
}

func initialize() {
	// Define application arguments and allow for override of database environment settings
	flag.BoolVar(&debug, "debug", debug, "Determines whether debug mode is enabled.")
	flag.BoolVar(&dryRun, "dry-run", dryRun, "Determines whether dry-run mode is enabled.")
	flag.IntVar(&workers, "workers", workers, "Defines the number of workers to create.")
	flag.Parse()

	// Load application settings from environment into structured configuration
	appConfig := config.CreateAppConfig(workers, dryRun, debug)
	appConfig.DbConfig = database.CreateConfigFromEnvironment()
	config.AppConfig = appConfig

	// Configure the logging API
	logging.SetLogLevel(appConfig.LogLevel)

	// Set up the task manager
	tasks.SetupTaskManager()

	initialized = true
}
