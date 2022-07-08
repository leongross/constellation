package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

// getVMInterfaces retrieves all network interfaces referenced by a virtual machine.
func (m *Metadata) getVMInterfaces(ctx context.Context, vm armcompute.VirtualMachine, resourceGroup string) ([]armnetwork.Interface, error) {
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil {
		return []armnetwork.Interface{}, nil
	}
	interfaceNames := extractInterfaceNamesFromInterfaceReferences(vm.Properties.NetworkProfile.NetworkInterfaces)
	networkInterfaces := []armnetwork.Interface{}
	for _, interfaceName := range interfaceNames {
		networkInterfacesResp, err := m.networkInterfacesAPI.Get(ctx, resourceGroup, interfaceName, nil)
		if err != nil {
			return nil, fmt.Errorf("retrieving network interface %v: %w", interfaceName, err)
		}
		networkInterfaces = append(networkInterfaces, networkInterfacesResp.Interface)
	}
	return networkInterfaces, nil
}

// getScaleSetVMInterfaces retrieves all network interfaces referenced by a scale set virtual machine.
func (m *Metadata) getScaleSetVMInterfaces(ctx context.Context, vm armcompute.VirtualMachineScaleSetVM, resourceGroup, scaleSet, instanceID string) ([]armnetwork.Interface, error) {
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil {
		return []armnetwork.Interface{}, nil
	}
	interfaceNames := extractInterfaceNamesFromInterfaceReferences(vm.Properties.NetworkProfile.NetworkInterfaces)
	networkInterfaces := []armnetwork.Interface{}
	for _, interfaceName := range interfaceNames {
		networkInterfacesResp, err := m.networkInterfacesAPI.GetVirtualMachineScaleSetNetworkInterface(ctx, resourceGroup, scaleSet, instanceID, interfaceName, nil)
		if err != nil {
			return nil, fmt.Errorf("retrieving network interface %v: %w", interfaceName, err)
		}
		networkInterfaces = append(networkInterfaces, networkInterfacesResp.Interface)
	}
	return networkInterfaces, nil
}

// getScaleSetVMPublicIPAddresses retrieves all public IP addresses from a network interface which is referenced by a scale set virtual machine.
func (m *Metadata) getScaleSetVMPublicIPAddresses(ctx context.Context, resourceGroup, scaleSet, instanceID string,
	networkInterfaces []armnetwork.Interface,
) ([]string, error) {
	var publicIPAddresses []string
	for _, networkInterface := range networkInterfaces {
		if networkInterface.Properties == nil || networkInterface.Name == nil {
			continue
		}
		for _, config := range networkInterface.Properties.IPConfigurations {
			if config == nil || config.Properties == nil || config.Properties.PublicIPAddress == nil || config.Name == nil {
				continue
			}
			publicIPAddressName := *config.Properties.PublicIPAddress.ID
			publicIPAddressNameParts := strings.Split(publicIPAddressName, "/")
			publicIPAddressName = publicIPAddressNameParts[len(publicIPAddressNameParts)-1]
			publicIPAddress, err := m.publicIPAddressesAPI.GetVirtualMachineScaleSetPublicIPAddress(ctx, resourceGroup, scaleSet, instanceID, *networkInterface.Name, *config.Name, publicIPAddressName, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve public ip address %v: %w", publicIPAddressName, err)
			}
			if publicIPAddress.Properties == nil || publicIPAddress.Properties.IPAddress == nil {
				return nil, errors.New("retrieved public ip address has invalid ip address")
			}
			publicIPAddresses = append(publicIPAddresses, *publicIPAddress.Properties.IPAddress)
		}
	}
	return publicIPAddresses, nil
}

// extractPrivateIPs extracts private IPs from a list of network interface IP configurations.
func extractPrivateIPs(networkInterfaces []armnetwork.Interface) []string {
	addresses := []string{}
	for _, networkInterface := range networkInterfaces {
		if networkInterface.Properties == nil || len(networkInterface.Properties.IPConfigurations) == 0 {
			continue
		}
		for _, config := range networkInterface.Properties.IPConfigurations {
			if config == nil || config.Properties == nil || config.Properties.PrivateIPAddress == nil {
				continue
			}
			addresses = append(addresses, *config.Properties.PrivateIPAddress)
		}
	}
	return addresses
}

// extractInterfaceNamesFromInterfaceReferences extracts the name of a network interface from a reference id.
// Format:
// - "/subscriptions/<subscription>/resourceGroups/<resource-group>/providers/Microsoft.Network/networkInterfaces/<interface-name>"
// - "/subscriptions/<subscription>/resourceGroups/<resource-group>/providers/Microsoft.Compute/virtualMachineScaleSets/<scale-set-name>/virtualMachines/<instanceID>/networkInterfaces/<interface-name>".
func extractInterfaceNamesFromInterfaceReferences(references []*armcompute.NetworkInterfaceReference) []string {
	interfaceNames := []string{}
	for _, interfaceReference := range references {
		if interfaceReference == nil || interfaceReference.ID == nil {
			continue
		}
		interfaceIDParts := strings.Split(*interfaceReference.ID, "/")
		if len(interfaceIDParts) < 1 {
			continue
		}
		interfaceName := interfaceIDParts[len(interfaceIDParts)-1]
		interfaceNames = append(interfaceNames, interfaceName)
	}
	return interfaceNames
}