package network

type Network struct {
	Id     int
	Name   string
	Alias  string
	Status int
}

type Subnet struct {
	Id                    int
	NetworkId             int
	Cidr                  string
	IPv4NetworkAddress    string
	IPv4NetworkAddressInt uint32
	IPv4NetworkMask       int
	Status                int
}

type Device struct {
	Id             int
	NetworkId      int
	SubnetId       int
	IPv4Address    string
	IPv4AddressInt uint32
	Status         int
}
