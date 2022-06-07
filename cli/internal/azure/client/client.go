package client

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/edgelesssys/constellation/cli/cloud/cloudtypes"
	"github.com/edgelesssys/constellation/internal/cloud/cloudprovider"
	"github.com/edgelesssys/constellation/internal/state"
)

const (
	graphAPIResource      = "https://graph.windows.net"
	managementAPIResource = "https://management.azure.com"
)

// Client is a client for Azure.
type Client struct {
	networksAPI
	networkSecurityGroupsAPI
	resourceGroupAPI
	scaleSetsAPI
	publicIPAddressesAPI
	networkInterfacesAPI
	loadBalancersAPI
	virtualMachinesAPI
	applicationsAPI
	servicePrincipalsAPI
	roleAssignmentsAPI

	adReplicationLagCheckInterval   time.Duration
	adReplicationLagCheckMaxRetries int

	nodes        cloudtypes.Instances
	coordinators cloudtypes.Instances

	name                 string
	uid                  string
	resourceGroup        string
	location             string
	subscriptionID       string
	tenantID             string
	subnetID             string
	coordinatorsScaleSet string
	nodesScaleSet        string
	loadBalancerName     string
	loadBalancerPubIP    string
	networkSecurityGroup string
	adAppObjectID        string
}

// NewFromDefault creates a client with initialized clients.
func NewFromDefault(subscriptionID, tenantID string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	graphAuthorizer, err := getAuthorizer(graphAPIResource)
	if err != nil {
		return nil, err
	}
	managementAuthorizer, err := getAuthorizer(managementAPIResource)
	if err != nil {
		return nil, err
	}
	netAPI := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	netSecGrpAPI := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	resGroupAPI := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	scaleSetAPI := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)
	publicIPAddressesAPI := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	networkInterfacesAPI := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
	loadBalancersAPI := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	virtualMachinesAPI := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	applicationsAPI := graphrbac.NewApplicationsClient(tenantID)
	applicationsAPI.Authorizer = graphAuthorizer
	servicePrincipalsAPI := graphrbac.NewServicePrincipalsClient(tenantID)
	servicePrincipalsAPI.Authorizer = graphAuthorizer
	roleAssignmentsAPI := authorization.NewRoleAssignmentsClient(subscriptionID)
	roleAssignmentsAPI.Authorizer = managementAuthorizer

	return &Client{
		networksAPI:                     &networksClient{netAPI},
		networkSecurityGroupsAPI:        &networkSecurityGroupsClient{netSecGrpAPI},
		resourceGroupAPI:                &resourceGroupsClient{resGroupAPI},
		scaleSetsAPI:                    &virtualMachineScaleSetsClient{scaleSetAPI},
		publicIPAddressesAPI:            &publicIPAddressesClient{publicIPAddressesAPI},
		networkInterfacesAPI:            &networkInterfacesClient{networkInterfacesAPI},
		loadBalancersAPI:                &loadBalancersClient{loadBalancersAPI},
		applicationsAPI:                 &applicationsClient{&applicationsAPI},
		servicePrincipalsAPI:            &servicePrincipalsClient{&servicePrincipalsAPI},
		roleAssignmentsAPI:              &roleAssignmentsClient{&roleAssignmentsAPI},
		virtualMachinesAPI:              &virtualMachinesClient{virtualMachinesAPI},
		subscriptionID:                  subscriptionID,
		tenantID:                        tenantID,
		nodes:                           cloudtypes.Instances{},
		coordinators:                    cloudtypes.Instances{},
		adReplicationLagCheckInterval:   adReplicationLagCheckInterval,
		adReplicationLagCheckMaxRetries: adReplicationLagCheckMaxRetries,
	}, nil
}

// NewInitialized creates and initializes client by setting the subscriptionID, location and name
// of the Constellation.
func NewInitialized(subscriptionID, tenantID, name, location string) (*Client, error) {
	client, err := NewFromDefault(subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}
	err = client.init(location, name)
	return client, err
}

// init initializes the client.
func (c *Client) init(location, name string) error {
	c.location = location
	c.name = name
	uid, err := c.generateUID()
	if err != nil {
		return err
	}
	c.uid = uid

	return nil
}

// GetState returns the state of the client as ConstellationState.
func (c *Client) GetState() (state.ConstellationState, error) {
	var stat state.ConstellationState
	stat.CloudProvider = cloudprovider.Azure.String()
	if len(c.resourceGroup) == 0 {
		return state.ConstellationState{}, errors.New("client has no resource group")
	}
	stat.AzureResourceGroup = c.resourceGroup
	if c.name == "" {
		return state.ConstellationState{}, errors.New("client has no name")
	}
	stat.Name = c.name
	if len(c.uid) == 0 {
		return state.ConstellationState{}, errors.New("client has no uid")
	}
	stat.UID = c.uid
	if len(c.location) == 0 {
		return state.ConstellationState{}, errors.New("client has no location")
	}
	stat.AzureLocation = c.location
	if len(c.subscriptionID) == 0 {
		return state.ConstellationState{}, errors.New("client has no subscription")
	}
	stat.AzureSubscription = c.subscriptionID
	if len(c.tenantID) == 0 {
		return state.ConstellationState{}, errors.New("client has no tenant")
	}
	stat.AzureTenant = c.tenantID
	if len(c.subnetID) == 0 {
		return state.ConstellationState{}, errors.New("client has no subnet")
	}
	stat.AzureSubnet = c.subnetID
	if len(c.networkSecurityGroup) == 0 {
		return state.ConstellationState{}, errors.New("client has no network security group")
	}
	stat.AzureNetworkSecurityGroup = c.networkSecurityGroup
	if len(c.nodesScaleSet) == 0 {
		return state.ConstellationState{}, errors.New("client has no nodes scale set")
	}
	stat.AzureNodesScaleSet = c.nodesScaleSet
	if len(c.coordinatorsScaleSet) == 0 {
		return state.ConstellationState{}, errors.New("client has no coordinators scale set")
	}
	stat.AzureCoordinatorsScaleSet = c.coordinatorsScaleSet
	if len(c.nodes) == 0 {
		return state.ConstellationState{}, errors.New("client has no nodes")
	}
	stat.AzureNodes = c.nodes
	if len(c.coordinators) == 0 {
		return state.ConstellationState{}, errors.New("client has no coordinators")
	}
	stat.AzureCoordinators = c.coordinators
	// AD App Object ID does not have to be set at all times
	stat.AzureADAppObjectID = c.adAppObjectID

	return stat, nil
}

// SetState sets the state of the client to the handed ConstellationState.
func (c *Client) SetState(stat state.ConstellationState) error {
	if stat.CloudProvider != cloudprovider.Azure.String() {
		return errors.New("state is not azure state")
	}
	if len(stat.AzureResourceGroup) == 0 {
		return errors.New("state has no resource group")
	}
	c.resourceGroup = stat.AzureResourceGroup
	if stat.Name == "" {
		return errors.New("state has no name")
	}
	c.name = stat.Name
	if len(stat.UID) == 0 {
		return errors.New("state has no uuid")
	}
	c.uid = stat.UID
	if len(stat.AzureLocation) == 0 {
		return errors.New("state has no location")
	}
	c.location = stat.AzureLocation
	if len(stat.AzureSubscription) == 0 {
		return errors.New("state has no subscription")
	}
	c.subscriptionID = stat.AzureSubscription
	if len(stat.AzureTenant) == 0 {
		return errors.New("state has no tenant")
	}
	c.tenantID = stat.AzureTenant
	if len(stat.AzureSubnet) == 0 {
		return errors.New("state has no subnet")
	}
	c.subnetID = stat.AzureSubnet
	if len(stat.AzureNetworkSecurityGroup) == 0 {
		return errors.New("state has no subnet")
	}
	c.networkSecurityGroup = stat.AzureNetworkSecurityGroup
	if len(stat.AzureNodesScaleSet) == 0 {
		return errors.New("state has no nodes scale set")
	}
	c.nodesScaleSet = stat.AzureNodesScaleSet
	if len(stat.AzureCoordinatorsScaleSet) == 0 {
		return errors.New("state has no nodes scale set")
	}
	c.coordinatorsScaleSet = stat.AzureCoordinatorsScaleSet
	if len(stat.AzureNodes) == 0 {
		return errors.New("state has no nodes")
	}
	c.nodes = stat.AzureNodes
	if len(stat.AzureCoordinators) == 0 {
		return errors.New("state has no coordinators")
	}
	c.coordinators = stat.AzureCoordinators
	// AD App Object ID does not have to be set at all times
	c.adAppObjectID = stat.AzureADAppObjectID

	return nil
}

func (c *Client) generateUID() (string, error) {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	const uidLen = 5
	uid := make([]byte, uidLen)
	for i := 0; i < uidLen; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		uid[i] = letters[n.Int64()]
	}
	return string(uid), nil
}

// getAuthorizer creates an autorest.Authorizer for different Azure AD APIs using either environment variables or azure cli credentials.
func getAuthorizer(resource string) (autorest.Authorizer, error) {
	authorizer, cliErr := auth.NewAuthorizerFromCLIWithResource(resource)
	if cliErr == nil {
		return authorizer, nil
	}
	authorizer, envErr := auth.NewAuthorizerFromEnvironmentWithResource(resource)
	if envErr == nil {
		return authorizer, nil
	}
	return nil, fmt.Errorf("unable to create authorizer from env or cli: %v %v", envErr, cliErr)
}