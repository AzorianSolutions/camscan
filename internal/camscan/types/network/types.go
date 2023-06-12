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
	IPv4NetworkAddressInt int
	IPv4NetworkMask       int
	Status                int
}
