/*
Package siteconnections allows management and retrieval of IPSec site connections in the
OpenStack Networking Service.


Example to create an IPSec site connection

createOpts := siteconnections.CreateOpts{
		Name:           "Connection1",
		PSK:            "secret",
		Initiator:      siteconnections.InitiatorBiDirectional,
		AdminStateUp:   gophercloud.Enabled,
		IPSecPolicyID:  "4ab0a72e-64ef-4809-be43-c3f7e0e5239b",
		PeerEPGroupID:  "5f5801b1-b383-4cf0-bf61-9e85d4044b2d",
		IKEPolicyID:    "47a880f9-1da9-468c-b289-219c9eca78f0",
		VPNServiceID:   "692c1ec8-a7cd-44d9-972b-8ed3fe4cc476",
		LocalEPGroupID: "498bb96a-1517-47ea-b1eb-c4a53db46a16",
		PeerAddress:    "172.24.4.233",
		PeerID:         "172.24.4.233",
		MTU:            1500,
	}
	connection, err := siteconnections.Create(client, createOpts).Extract()
	if err != nil {
		panic(err)
	}
*/
package siteconnections
