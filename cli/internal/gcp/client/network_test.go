package client

import (
	"context"
	"errors"
	"testing"

	"github.com/edgelesssys/constellation/cli/cloud/cloudtypes"
	"github.com/stretchr/testify/assert"
)

func TestCreateVPCs(t *testing.T) {
	someErr := errors.New("failed")

	testCases := map[string]struct {
		operationGlobalAPI operationGlobalAPI
		operationRegionAPI operationRegionAPI
		networksAPI        networksAPI
		subnetworksAPI     subnetworksAPI
		wantErr            bool
	}{
		"successful create": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
		},
		"failed wait global op": {
			operationGlobalAPI: stubOperationGlobalAPI{waitErr: someErr},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
		},
		"failed wait region op": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{waitErr: someErr},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
		},
		"failed insert networks": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{insertErr: someErr},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
		},
		"failed insert subnetworks": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{insertErr: someErr},
			wantErr:            true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()
			client := Client{
				project:            "project",
				zone:               "zone",
				name:               "name",
				uid:                "uid",
				operationGlobalAPI: tc.operationGlobalAPI,
				operationRegionAPI: tc.operationRegionAPI,
				networksAPI:        tc.networksAPI,
				subnetworksAPI:     tc.subnetworksAPI,
				nodes:              make(cloudtypes.Instances),
				coordinators:       make(cloudtypes.Instances),
			}

			if tc.wantErr {
				assert.Error(client.CreateVPCs(ctx))
			} else {
				assert.NoError(client.CreateVPCs(ctx))
				assert.NotNil(client.network)
			}
		})
	}
}

func TestTerminateVPCs(t *testing.T) {
	someErr := errors.New("failed")
	testCases := map[string]struct {
		operationGlobalAPI operationGlobalAPI
		operationRegionAPI operationRegionAPI
		networksAPI        networksAPI
		subnetworksAPI     subnetworksAPI
		firewalls          []string
		subnetwork         string
		wantErr            bool
	}{
		"successful terminate": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			subnetwork:         "subnetwork-id-1",
		},
		"subnetwork empty": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			subnetwork:         "",
		},
		"failed wait global op": {
			operationGlobalAPI: stubOperationGlobalAPI{waitErr: someErr},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
			subnetwork:         "subnetwork-id-1",
		},
		"failed delete networks": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{deleteErr: someErr},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
			subnetwork:         "subnetwork-id-1",
		},
		"failed delete subnetworks": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			operationRegionAPI: stubOperationRegionAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{deleteErr: someErr},
			wantErr:            true,
			subnetwork:         "subnetwork-id-1",
		},
		"must delete firewalls first": {
			firewalls:          []string{"firewall-1", "firewall-2"},
			operationRegionAPI: stubOperationRegionAPI{},
			operationGlobalAPI: stubOperationGlobalAPI{},
			networksAPI:        stubNetworksAPI{},
			subnetworksAPI:     stubSubnetworksAPI{},
			wantErr:            true,
			subnetwork:         "subnetwork-id-1",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()
			client := Client{
				project:            "project",
				zone:               "zone",
				name:               "name",
				uid:                "uid",
				operationGlobalAPI: tc.operationGlobalAPI,
				operationRegionAPI: tc.operationRegionAPI,
				networksAPI:        tc.networksAPI,
				subnetworksAPI:     tc.subnetworksAPI,
				firewalls:          tc.firewalls,
				network:            "network-id-1",
				subnetwork:         tc.subnetwork,
			}

			if tc.wantErr {
				assert.Error(client.TerminateVPCs(ctx))
			} else {
				assert.NoError(client.TerminateVPCs(ctx))
				assert.Empty(client.network)
			}
		})
	}
}

func TestCreateFirewall(t *testing.T) {
	someErr := errors.New("failed")
	testFirewallInput := FirewallInput{
		Ingress: cloudtypes.Firewall{
			cloudtypes.FirewallRule{
				Name:        "test-1",
				Description: "test-1 description",
				Protocol:    "tcp",
				IPRange:     "192.0.2.0/24",
				FromPort:    9000,
			},
			cloudtypes.FirewallRule{
				Name:        "test-2",
				Description: "test-2 description",
				Protocol:    "udp",
				IPRange:     "192.0.2.0/24",
				FromPort:    51820,
			},
		},
		Egress: cloudtypes.Firewall{},
	}

	testCases := map[string]struct {
		network            string
		operationGlobalAPI operationGlobalAPI
		firewallsAPI       firewallsAPI
		firewallInput      FirewallInput
		wantErr            bool
	}{
		"successful create": {
			network:            "network",
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{},
		},
		"failed wait global op": {
			network:            "network",
			operationGlobalAPI: stubOperationGlobalAPI{waitErr: someErr},
			firewallsAPI:       stubFirewallsAPI{},
			wantErr:            true,
		},
		"failed insert networks": {
			network:            "network",
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{insertErr: someErr},
			wantErr:            true,
		},
		"no network set": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{},
			wantErr:            true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()
			client := Client{
				project:            "project",
				zone:               "zone",
				name:               "name",
				uid:                "uid",
				network:            tc.network,
				operationGlobalAPI: tc.operationGlobalAPI,
				firewallsAPI:       tc.firewallsAPI,
			}

			if tc.wantErr {
				assert.Error(client.CreateFirewall(ctx, testFirewallInput))
			} else {
				assert.NoError(client.CreateFirewall(ctx, testFirewallInput))
				assert.ElementsMatch([]string{"test-1", "test-2"}, client.firewalls)
			}
		})
	}
}

func TestTerminateFirewall(t *testing.T) {
	someErr := errors.New("failed")

	testCases := map[string]struct {
		operationGlobalAPI operationGlobalAPI
		firewallsAPI       firewallsAPI
		firewalls          []string
		wantErr            bool
	}{
		"successful terminate": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{},
			firewalls:          []string{"firewall-1", "firewall-2"},
		},
		"successful terminate when no firewall exists": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{},
			firewalls:          []string{},
		},
		"failed to wait on global operation": {
			operationGlobalAPI: stubOperationGlobalAPI{waitErr: someErr},
			firewallsAPI:       stubFirewallsAPI{},
			firewalls:          []string{"firewall-1", "firewall-2"},
			wantErr:            true,
		},
		"failed to delete firewalls": {
			operationGlobalAPI: stubOperationGlobalAPI{},
			firewallsAPI:       stubFirewallsAPI{deleteErr: someErr},
			firewalls:          []string{"firewall-1", "firewall-2"},
			wantErr:            true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()
			client := Client{
				project:            "project",
				zone:               "zone",
				name:               "name",
				uid:                "uid",
				firewalls:          tc.firewalls,
				operationGlobalAPI: tc.operationGlobalAPI,
				firewallsAPI:       tc.firewallsAPI,
			}

			if tc.wantErr {
				assert.Error(client.TerminateFirewall(ctx))
			} else {
				assert.NoError(client.TerminateFirewall(ctx))
				assert.Empty(client.firewalls)
			}
		})
	}
}