package database

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strconv"
	"strings"
	"time"
)

var configs = make(map[string]types.DbConfig)
var connections = make(map[string]*sql.DB)
var db *sql.DB = nil
var dbConnectionString string

func CreateConfig(host string, port string, user string, password string, name string, retries int, delay int) types.DbConfig {
	var config = GetDefaultConfig()
	config.Host = host
	config.Port = port
	config.User = user
	config.Password = password
	config.Name = name
	config.ConnectRetries = retries
	config.ConnectRetryDelay = delay
	return config
}

func CreateConfigFromEnvironment() types.DbConfig {
	var defaultConfig = GetDefaultConfig()
	var host = strings.Trim(os.Getenv("CAMS_DB_HOST"), " ")
	var port = strings.Trim(os.Getenv("CAMS_DB_PORT"), " ")
	var user = strings.Trim(os.Getenv("CAMS_DB_USER"), " ")
	var password = strings.Trim(os.Getenv("CAMS_DB_PASSWORD"), " ")
	var name = strings.Trim(os.Getenv("CAMS_DB_NAME"), " ")
	var retries, _ = strconv.Atoi(strings.Trim(os.Getenv("CAMS_DB_CONNECT_RETRIEES"), " "))
	var delay, _ = strconv.Atoi(strings.Trim(os.Getenv("CAMS_DB_CONNECT_RETRY_DELAY"), " "))

	if len(host) == 0 {
		host = defaultConfig.Host
	}

	if len(port) == 0 {
		port = defaultConfig.Port
	}

	if len(user) == 0 {
		user = defaultConfig.User
	}

	if len(password) == 0 {
		password = defaultConfig.Password
	}

	if len(name) == 0 {
		name = defaultConfig.Name
	}

	return CreateConfig(host, port, user, password, name, retries, delay)
}

func GetDefaultConfig() types.DbConfig {
	// Create a DbConfig instance populated with the program's default values
	return types.DbConfig{
		Host:     "localhost",
		Name:     "camscan",
		Password: "camscan",
		Port:     "3306",
		User:     "camscan",
	}
}

func HasConnection(name string) bool {
	// Check if the connections map contains the given key and return true if found, false otherwise
	if _, ok := connections[name]; ok {
		return true
	}
	return false
}

func CreateConnection(name string, config types.DbConfig) (bool, *sql.DB) {

	// Cache a reference to the connection configuration
	configs[name] = config

	// Build the database connection string
	dbConnectionString = config.User + ":" + config.Password +
		"@(" + config.Host + ":" + config.Port + ")/" + config.Name

	// Declare connection opening error variable
	var dbOpenError error = nil

	logging.Debug("Connecting to MySQL server; server: %s; port: %s; user: %s; name: %s;",
		config.Host, config.Port, config.User, config.Name)

	// Attempt to open connection to database
	db, dbOpenError = sql.Open("mysql", dbConnectionString)

	// Check for errors when opening database connection and handle accordingly
	if dbOpenError != nil {
		logging.Error("Error connecting to MySQL server; server: %s; port: %s; user: %s; name: %s; error: %s;",
			config.Host, config.Port, config.User, config.Name, dbOpenError.Error())
		return false, db
	}

	// Cache a reference to the connection
	connections[name] = db

	return true, db
}

func OpenConnection(name string, config types.DbConfig) (bool, *sql.DB) {
	var db *sql.DB
	opened := false
	attempts := 0

	for opened == false {
		if attempts > 0 {
			logging.Debug("Failed to connect to database server. Waiting %v seconds to try again...",
				config.ConnectRetryDelay)
			time.Sleep(time.Second * time.Duration(config.ConnectRetryDelay))
		}

		attempts++
		opened, db = CreateConnection(name, config)

		if opened == false && attempts >= config.ConnectRetries {
			logging.Debug("The threshold for database connection attempts of %v has been reached.")
			return false, db
		}
	}

	return true, db
}

func CloseConnection(name string) bool {
	if HasConnection(name) == false {
		return false
	}

	logging.Debug("Closing database connection; name: %s;", name)

	// Retrieve a reference to the given connection
	db := GetConnection(name)
	dbConfig := configs[name]
	err := db.Close()

	if err != nil {
		logging.Error("Error closing connection to MySQL database server; "+
			"server: %s; port: %s; user: %s; name: %s; error: %s;",
			dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Name, err.Error())
		return false
	} else {
		logging.Debug("Closed connection to MySQL database server; "+
			"server: %s; port: %s; user: %s; name: %s;",
			dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Name)
	}

	return true
}

func GetConnection(name string) *sql.DB {
	return connections[name]
}

func RemoveConnection(name string) bool {
	// Verify that the given connection name is already registered or return false otherwise
	if HasConnection(name) == false {
		return false
	}
	// Delete the connection reference from the map
	delete(connections, name)
	return true
}
