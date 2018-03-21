/*
Package endpointgroups allows management of endpoint groups in the Openstack Network Service

Example to create an Endpoint Group

	createOpts := endpointgroups.CreateOpts{
		Name: groupName,
		Type: endpointgroups.TypeCIDR,
		Endpoints: []string{
			"10.2.0.0/24",
			"10.3.0.0/24",
		},
	}
	group, err := endpointgroups.Create(client, createOpts).Extract()
	if err != nil {
		return group, err
	}

Example to retrieve an Endpoint Group

	group, err := endpointgroups.Get(client, "6ecd9cf3-ca64-46c7-863f-f2eb1b9e838a").Extract()
	if err != nil {
		panic(err)
	}

Example to Delete an Endpoint Group

	err := endpointgroups.Delete(client, "5291b189-fd84-46e5-84bd-78f40c05d69c").ExtractErr()
	if err != nil {
		panic(err)
	}

Example to List Endpoint groups

	allPages, err := endpointgroups.List(client, nil).AllPages()
	if err != nil {
		panic(err)
	}

	allGroups, err := endpointgroups.ExtractEndpointGroups(allPages)
	if err != nil {
		panic(err)
	}
*/
package endpointgroups
