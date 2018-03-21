/*
Package portsecurity provides information and interaction with the port
security extension for the OpenStack Networking service.

Example to List Networks with Port Security Information

	type NetworkWithPortSecurityExt struct {
		networks.Network
		portsecurity.PortSecurityExt
	}

	var allNetworks []NetworkWithPortSecurityExt

	listOpts := networks.ListOpts{
		Name: "network_1",
	}

	allPages, err := networks.List(networkClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	err = networks.ExtractNetworksInto(allPages, &allNetworks)
	if err != nil {
		panic(err)
	}

	for _, network := range allNetworks {
		fmt.Println("%+v\n", network)
	}

Example to Get a Port with Port Security Information

	var portWithExtensions struct {
		ports.Port
		portsecurity.PortSecurityExt
	}

	portID := "46d4bfb9-b26e-41f3-bd2e-e6dcc1ccedb2"

	err := ports.Get(networkingClient, portID).ExtractInto(&portWithExtensions)
	if err != nil {
		panic(err)
	}

	fmt.Println("%+v\n", portWithExtensions)
*/
package portsecurity
