package config

import (
	"as/camscan/internal/camscan/logging"
	"as/camscan/internal/camscan/types"
	"os"
	"strconv"
	"strings"
)

const DefaultSnmpTimeout = 3
const DefaultWorkers = 10
const MinSnmpTimeout = 0.1
const MinWorkers = 1

var AppConfig types.AppConfig

func CreateAppConfig(workers int, dryRun bool, debug bool) types.AppConfig {
	community := strings.Trim(os.Getenv("CAMS_COMMUNITY"), " ")
	debugEnv := strings.Trim(os.Getenv("CAMS_DEBUG"), " ")
	dryRunEnv := strings.Trim(os.Getenv("CAMS_DRY_RUN"), " ")
	icmpRetries, _ := strconv.Atoi(strings.Trim(os.Getenv("CAMS_ICMP_RETRIES"), " "))
	icmpTimeout, _ := strconv.ParseFloat(strings.Trim(os.Getenv("CAMS_ICMP_TIMEOUT"), " "), 64)
	logLevel, _ := strconv.Atoi(strings.Trim(os.Getenv("CAMS_LOG_LEVEL"), " "))
	snmpApCommunity := strings.Trim(os.Getenv("CAMS_SNMP_AP_COMMUNITY"), " ")
	snmpSmCommunity := strings.Trim(os.Getenv("CAMS_SNMP_SM_COMMUNITY"), " ")
	snmpTimeoutAp, _ := strconv.ParseFloat(strings.Trim(os.Getenv("CAMS_SNMP_TIMEOUT_AP"), " "), 64)
	snmpTimeoutSm, _ := strconv.ParseFloat(strings.Trim(os.Getenv("CAMS_SNMP_TIMEOUT_SM"), " "), 64)
	workersEnv, _ := strconv.Atoi(strings.Trim(os.Getenv("CAMS_WORKERS"), " "))

	// Enforce minimum worker policy as well as assign default values
	if workers < MinWorkers {
		workers = DefaultWorkers
	}

	if community == "" {
		community = "public"
	}

	if len(debugEnv) > 0 {
		debug, _ = strconv.ParseBool(debugEnv)
	}

	if len(dryRunEnv) > 0 {
		dryRun, _ = strconv.ParseBool(dryRunEnv)
	}

	if logLevel <= 0 {
		logLevel = logging.DefaultLogLevel
	}

	if snmpTimeoutAp == 0 {
		snmpTimeoutAp = DefaultSnmpTimeout
	} else if snmpTimeoutAp < MinSnmpTimeout {
		logging.Debug("Changing value for the 'SNMP_TIMEOUT_AP' setting from '%v' to '%v'", snmpTimeoutAp, MinSnmpTimeout)
		snmpTimeoutAp = MinSnmpTimeout
	}

	if snmpTimeoutSm == 0 {
		snmpTimeoutSm = DefaultSnmpTimeout
	} else if snmpTimeoutSm < MinSnmpTimeout {
		logging.Debug("Changing value for the 'SNMP_TIMEOUT_SM' setting from '%v' to '%v'", snmpTimeoutSm, MinSnmpTimeout)
		snmpTimeoutSm = MinSnmpTimeout
	}

	if workersEnv > 0 {
		workers = workersEnv
	}

	if workers < MinWorkers {
		logging.Debug("Changing value for the 'workers' parameter from '%v' to '%v'", workers, MinWorkers)
		workers = MinWorkers
	}

	config := types.AppConfig{
		Community:       community,
		Debug:           debug,
		DryRun:          dryRun,
		ICMPRetries:     icmpRetries,
		ICMPTimeout:     icmpTimeout,
		LogLevel:        logLevel,
		SnmpApCommunity: snmpApCommunity,
		SnmpSmCommunity: snmpSmCommunity,
		SnmpTimeoutSm:   snmpTimeoutSm,
		SnmpTimeoutAp:   snmpTimeoutAp,
		Workers:         workers,
	}

	return config
}
