/*
Package usage provides information and interaction with the
SimpleTenantUsage extension for the OpenStack Compute service.

Example to Retrieve Usage for a Single Tenant:
	start := time.Date(2017, 01, 21, 10, 4, 20, 0, time.UTC)
	end := time.Date(2017, 01, 21, 10, 4, 20, 0, time.UTC)

    singleTenantOpts := usage.SingleTenantOpts{
        Start: &start,
        End: &end,
    }

    page, err := usage.SingleTenant(computeClient, tenantID, singleTenantOpts).AllPages()
    if err != nil {
        panic(err)
    }

    tenantUsage, err := usage.ExtractSingleTenant(page)
    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", tenantUsage)

*/
package usage
