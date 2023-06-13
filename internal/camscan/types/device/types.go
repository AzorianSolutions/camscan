package device

type AccessPoint struct {
	Id             int
	NetworkId      int
	MacAddress     string
	IPv4Address    string
	IPv4AddressInt uint32
	Status         int
}

type SubscriberModule struct {
	Id             int
	NetworkId      int
	MacAddress     string
	IPv4Address    string
	IPv4AddressInt uint32
	Status         int
}
