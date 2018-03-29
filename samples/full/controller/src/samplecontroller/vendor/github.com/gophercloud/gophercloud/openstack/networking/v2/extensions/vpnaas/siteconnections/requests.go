package siteconnections

import (
	"github.com/gophercloud/gophercloud"
)

// CreateOptsBuilder allows extensions to add additional parameters to the
// Create request.
type CreateOptsBuilder interface {
	ToConnectionCreateMap() (map[string]interface{}, error)
}
type Action string
type Initiator string

const (
	ActionHold             Action    = "hold"
	ActionClear            Action    = "clear"
	ActionRestart          Action    = "restart"
	ActionDisabled         Action    = "disabled"
	ActionRestartByPeer    Action    = "restart-by-peer"
	InitiatorBiDirectional Initiator = "bi-directional"
	InitiatorResponseOnly  Initiator = "response-only"
)

// DPDCreateOpts contains all the values needed to create a valid configuration for Dead Peer detection protocols
type DPDCreateOpts struct {
	// The dead peer detection (DPD) action.
	// A valid value is clear, hold, restart, disabled, or restart-by-peer.
	// Default value is hold.
	Action Action `json:"action,omitempty"`

	// The dead peer detection (DPD) timeout in seconds.
	// A valid value is a positive integer that is greater than the DPD interval value.
	// Default is 120.
	Timeout int `json:"timeout,omitempty"`

	// The dead peer detection (DPD) interval, in seconds.
	// A valid value is a positive integer.
	// Default is 30.
	Interval int `json:"interval,omitempty"`
}

// CreateOpts contains all the values needed to create a new IPSec site connection
type CreateOpts struct {
	// The ID of the IKE policy
	IKEPolicyID string `json:"ikepolicy_id"`

	// The ID of the VPN Service
	VPNServiceID string `json:"vpnservice_id"`

	// The ID for the endpoint group that contains private subnets for the local side of the connection.
	// You must specify this parameter with the peer_ep_group_id parameter unless
	// in backward- compatible mode where peer_cidrs is provided with a subnet_id for the VPN service.
	LocalEPGroupID string `json:"local_ep_group_id,omitempty"`

	// The ID of the IPsec policy.
	IPSecPolicyID string `json:"ipsecpolicy_id"`

	// The peer router identity for authentication.
	// A valid value is an IPv4 address, IPv6 address, e-mail address, key ID, or FQDN.
	// Typically, this value matches the peer_address value.
	PeerID string `json:"peer_id"`

	// The ID of the project
	TenantID string `json:"tenant_id,omitempty"`

	// The ID for the endpoint group that contains private CIDRs in the form < net_address > / < prefix >
	// for the peer side of the connection.
	// You must specify this parameter with the local_ep_group_id parameter unless in backward-compatible mode
	// where peer_cidrs is provided with a subnet_id for the VPN service.
	PeerEPGroupID string `json:"peer_ep_group_id,omitempty"`

	// An ID to be used instead of the external IP address for a virtual router used in traffic between instances on different networks in east-west traffic.
	// Most often, local ID would be domain name, email address, etc.
	// If this is not configured then the external IP address will be used as the ID.
	LocalID string `json:"local_id,omitempty"`

	// The human readable name of the connection.
	// Does not have to be unique.
	// Default is an empty string
	Name string `json:"name,omitempty"`

	// The human readable description of the connection.
	// Does not have to be unique.
	// Default is an empty string
	Description string `json:"description,omitempty"`

	// The peer gateway public IPv4 or IPv6 address or FQDN.
	PeerAddress string `json:"peer_address"`

	// The pre-shared key.
	// A valid value is any string.
	PSK string `json:"psk"`

	// Indicates whether this VPN can only respond to connections or both respond to and initiate connections.
	// A valid value is response-only or bi-directional. Default is bi-directional.
	Initiator Initiator `json:"initiator,omitempty"`

	// Unique list of valid peer private CIDRs in the form < net_address > / < prefix > .
	PeerCIDRs []string `json:"peer_cidrs,omitempty"`

	// The administrative state of the resource, which is up (true) or down (false).
	// Default is false
	AdminStateUp *bool `json:"admin_state_up,omitempty"`

	// A dictionary with dead peer detection (DPD) protocol controls.
	DPD *DPDCreateOpts `json:"dpd,omitempty"`

	// The maximum transmission unit (MTU) value to address fragmentation.
	// Minimum value is 68 for IPv4, and 1280 for IPv6.
	MTU int `json:"mtu,omitempty"`
}

// ToServiceCreateMap casts a CreateOpts struct to a map.
func (opts CreateOpts) ToConnectionCreateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "ipsec_site_connection")
}

// Create accepts a CreateOpts struct and uses the values to create a new
// IPSec site connection.
func Create(c *gophercloud.ServiceClient, opts CreateOptsBuilder) (r CreateResult) {
	b, err := opts.ToConnectionCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = c.Post(rootURL(c), b, &r.Body, nil)
	return
}

// Delete will permanently delete a particular IPSec site connection based on its
// unique ID.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(resourceURL(c, id), nil)
	return
}
