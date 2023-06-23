package snmp

const DeviceTypeAccessPoint = 1
const DeviceTypeSubscriberModule = 2

type OidMap struct {
	Id         int
	DeviceType int
	KeyName    string
	Oid        string
	Order      int
}

type Value struct {
	Id            int
	DeviceType    int
	DeviceId      int
	OidMapId      int
	SnmpType      int
	SnmpValueChar string
	SnmpValueNum  float64
	SnmpValueText string
	Captured      int
}
