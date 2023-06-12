package types

type AppConfig struct {
	Community       string
	DbConfig        DbConfig
	Debug           bool
	DryRun          bool
	LogLevel        int
	SnmpApCommunity string
	SnmpSmCommunity string
	SnmpTimeoutAp   float64
	SnmpTimeoutSm   float64
	Workers         int
}

type DbConfig struct {
	Host              string
	Name              string
	Password          string
	Port              string
	User              string
	ConnectRetries    int
	ConnectRetryDelay int
}
