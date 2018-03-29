/*
Package services allows management and retrieval of VPN services in the
OpenStack Networking Service.

Example to List Services

	listOpts := services.ListOpts{
		TenantID: "966b3c7d36a24facaf20b7e458bf2192",
	}

	allPages, err := services.List(networkClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allPolicies, err := services.ExtractServices(allPages)
	if err != nil {
		panic(err)
	}

	for _, service := range allServices {
		fmt.Printf("%+v\n", service)
	}

Example to Create a Service

	createOpts := services.CreateOpts{
		Name:        "vpnservice1",
		Description: "A service",
		RouterID:	 "2512e759-e8d7-4eea-a0af-4a85927a2e59",
		AdminStateUp: gophercloud.Enabled,
	}

	service, err := services.Create(networkClient, createOpts).Extract()
	if err != nil {
		panic(err)
	}

Example to Delete a Service

	serviceID := "38aee955-6283-4279-b091-8b9c828000ec"
	err := policies.Delete(networkClient, serviceID).ExtractErr()
	if err != nil {
		panic(err)
	}

*/
package services
