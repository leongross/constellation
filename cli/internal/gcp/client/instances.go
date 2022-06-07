package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/edgelesssys/constellation/cli/cloud/cloudtypes"
	"github.com/edgelesssys/constellation/coordinator/role"
	"google.golang.org/api/iterator"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

// CreateInstances creates instances (virtual machines) on Google Compute Engine.
//
// A separate managed instance group is created for coordinators and nodes, the function
// waits until the instances are up and stores the public and private IPs of the instances
// in the client. If the client's network must be set before instances can be created.
func (c *Client) CreateInstances(ctx context.Context, input CreateInstancesInput) error {
	if c.network == "" {
		return errors.New("client has no network")
	}
	ops := []Operation{}

	nodeTemplateInput := insertInstanceTemplateInput{
		Name:                         c.name + "-worker-" + c.uid,
		Network:                      c.network,
		SecondarySubnetworkRangeName: c.secondarySubnetworkRange,
		Subnetwork:                   c.subnetwork,
		ImageId:                      input.ImageId,
		InstanceType:                 input.InstanceType,
		StateDiskSizeGB:              int64(input.StateDiskSizeGB),
		Role:                         role.Node.String(),
		KubeEnv:                      input.KubeEnv,
		Project:                      c.project,
		Zone:                         c.zone,
		Region:                       c.region,
		UID:                          c.uid,
	}
	op, err := c.insertInstanceTemplate(ctx, nodeTemplateInput)
	if err != nil {
		return fmt.Errorf("inserting instanceTemplate failed: %w", err)
	}
	ops = append(ops, op)
	c.nodeTemplate = nodeTemplateInput.Name

	coordinatorTemplateInput := insertInstanceTemplateInput{
		Name:                         c.name + "-control-plane-" + c.uid,
		Network:                      c.network,
		Subnetwork:                   c.subnetwork,
		SecondarySubnetworkRangeName: c.secondarySubnetworkRange,
		ImageId:                      input.ImageId,
		InstanceType:                 input.InstanceType,
		StateDiskSizeGB:              int64(input.StateDiskSizeGB),
		Role:                         role.Coordinator.String(),
		KubeEnv:                      input.KubeEnv,
		Project:                      c.project,
		Zone:                         c.zone,
		Region:                       c.region,
		UID:                          c.uid,
	}
	op, err = c.insertInstanceTemplate(ctx, coordinatorTemplateInput)
	if err != nil {
		return fmt.Errorf("inserting instanceTemplate failed: %w", err)
	}
	ops = append(ops, op)
	c.coordinatorTemplate = coordinatorTemplateInput.Name
	if err := c.waitForOperations(ctx, ops); err != nil {
		return err
	}
	ops = []Operation{}

	coordinatorGroupInput := instanceGroupManagerInput{
		Count:    input.CountCoordinators,
		Name:     strings.Join([]string{c.name, "control-plane", c.uid}, "-"),
		Template: c.coordinatorTemplate,
		UID:      c.uid,
		Project:  c.project,
		Zone:     c.zone,
	}
	op, err = c.insertInstanceGroupManger(ctx, coordinatorGroupInput)
	if err != nil {
		return fmt.Errorf("inserting instanceGroupManager failed: %w", err)
	}
	ops = append(ops, op)
	c.coordinatorInstanceGroup = coordinatorGroupInput.Name

	nodeGroupInput := instanceGroupManagerInput{
		Count:    input.CountNodes,
		Name:     strings.Join([]string{c.name, "worker", c.uid}, "-"),
		Template: c.nodeTemplate,
		UID:      c.uid,
		Project:  c.project,
		Zone:     c.zone,
	}
	op, err = c.insertInstanceGroupManger(ctx, nodeGroupInput)
	if err != nil {
		return fmt.Errorf("inserting instanceGroupManager failed: %w", err)
	}
	ops = append(ops, op)
	c.nodesInstanceGroup = nodeGroupInput.Name

	if err := c.waitForOperations(ctx, ops); err != nil {
		return err
	}

	if err := c.waitForInstanceGroupScaling(ctx, c.nodesInstanceGroup); err != nil {
		return fmt.Errorf("waiting for instanceGroupScaling failed: %w", err)
	}

	if err := c.waitForInstanceGroupScaling(ctx, c.coordinatorInstanceGroup); err != nil {
		return fmt.Errorf("waiting for instanceGroupScaling failed: %w", err)
	}

	if err := c.getInstanceIPs(ctx, c.nodesInstanceGroup, c.nodes); err != nil {
		return fmt.Errorf("failed to get instanceIPs: %w", err)
	}
	if err := c.getInstanceIPs(ctx, c.coordinatorInstanceGroup, c.coordinators); err != nil {
		return fmt.Errorf("failed to get instanceIPs: %w", err)
	}
	return nil
}

// TerminateInstances terminates the clients instances.
func (c *Client) TerminateInstances(ctx context.Context) error {
	ops := []Operation{}
	if c.nodesInstanceGroup != "" {
		op, err := c.deleteInstanceGroupManager(ctx, c.nodesInstanceGroup)
		if err != nil {
			return fmt.Errorf("deleting instanceGroupManager '%s' failed: %w", c.nodesInstanceGroup, err)
		}
		ops = append(ops, op)
		c.nodesInstanceGroup = ""
		c.nodes = make(cloudtypes.Instances)
	}

	if c.coordinatorInstanceGroup != "" {
		op, err := c.deleteInstanceGroupManager(ctx, c.coordinatorInstanceGroup)
		if err != nil {
			return fmt.Errorf("deleting instanceGroupManager '%s' failed: %w", c.coordinatorInstanceGroup, err)
		}
		ops = append(ops, op)
		c.coordinatorInstanceGroup = ""
		c.coordinators = make(cloudtypes.Instances)
	}
	if err := c.waitForOperations(ctx, ops); err != nil {
		return err
	}
	ops = []Operation{}

	if c.nodeTemplate != "" {
		op, err := c.deleteInstanceTemplate(ctx, c.nodeTemplate)
		if err != nil {
			return fmt.Errorf("deleting instanceTemplate failed: %w", err)
		}
		ops = append(ops, op)
		c.nodeTemplate = ""
	}
	if c.coordinatorTemplate != "" {
		op, err := c.deleteInstanceTemplate(ctx, c.coordinatorTemplate)
		if err != nil {
			return fmt.Errorf("deleting instanceTemplate failed: %w", err)
		}
		ops = append(ops, op)
		c.coordinatorTemplate = ""
	}
	return c.waitForOperations(ctx, ops)
}

func (c *Client) insertInstanceTemplate(ctx context.Context, input insertInstanceTemplateInput) (Operation, error) {
	req := input.insertInstanceTemplateRequest()
	return c.instanceTemplateAPI.Insert(ctx, req)
}

func (c *Client) deleteInstanceTemplate(ctx context.Context, name string) (Operation, error) {
	req := &computepb.DeleteInstanceTemplateRequest{
		InstanceTemplate: name,
		Project:          c.project,
	}
	return c.instanceTemplateAPI.Delete(ctx, req)
}

func (c *Client) insertInstanceGroupManger(ctx context.Context, input instanceGroupManagerInput) (Operation, error) {
	req := input.InsertInstanceGroupManagerRequest()
	return c.instanceGroupManagersAPI.Insert(ctx, &req)
}

func (c *Client) deleteInstanceGroupManager(ctx context.Context, instanceGroupManagerName string) (Operation, error) {
	req := &computepb.DeleteInstanceGroupManagerRequest{
		InstanceGroupManager: instanceGroupManagerName,
		Project:              c.project,
		Zone:                 c.zone,
	}
	return c.instanceGroupManagersAPI.Delete(ctx, req)
}

func (c *Client) waitForInstanceGroupScaling(ctx context.Context, groupId string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		listReq := &computepb.ListManagedInstancesInstanceGroupManagersRequest{
			InstanceGroupManager: groupId,
			Project:              c.project,
			Zone:                 c.zone,
		}
		it := c.instanceGroupManagersAPI.ListManagedInstances(ctx, listReq)
		for {
			resp, err := it.Next()
			if errors.Is(err, iterator.Done) {
				return nil
			}
			if err != nil {
				return err
			}
			if resp.CurrentAction == nil {
				return errors.New("currentAction is nil")
			}
			if *resp.CurrentAction != computepb.ManagedInstance_NONE.String() {
				time.Sleep(5 * time.Second)
				break
			}
		}
	}
}

// getInstanceIPs requests the IPs of the client's instances.
func (c *Client) getInstanceIPs(ctx context.Context, groupId string, list cloudtypes.Instances) error {
	req := &computepb.ListInstancesRequest{
		Filter:  proto.String("name=" + groupId + "*"),
		Project: c.project,
		Zone:    c.zone,
	}
	it := c.instanceAPI.List(ctx, req)
	for {
		resp, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		if resp.Name == nil {
			return errors.New("instance name is nil pointer")
		}
		if len(resp.NetworkInterfaces) == 0 {
			return errors.New("network interface is empty")
		}
		if resp.NetworkInterfaces[0].NetworkIP == nil {
			return errors.New("networkIP is nil")
		}
		if len(resp.NetworkInterfaces[0].AccessConfigs) == 0 {
			return errors.New("access configs is empty")
		}
		if resp.NetworkInterfaces[0].AccessConfigs[0].NatIP == nil {
			return errors.New("natIP is nil")
		}
		instance := cloudtypes.Instance{
			PrivateIP: *resp.NetworkInterfaces[0].NetworkIP,
			PublicIP:  *resp.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		}
		list[*resp.Name] = instance
	}
}

type instanceGroupManagerInput struct {
	Count    int
	Name     string
	Template string
	Project  string
	Zone     string
	UID      string
}

func (i *instanceGroupManagerInput) InsertInstanceGroupManagerRequest() computepb.InsertInstanceGroupManagerRequest {
	return computepb.InsertInstanceGroupManagerRequest{
		InstanceGroupManagerResource: &computepb.InstanceGroupManager{
			BaseInstanceName: proto.String(i.Name),
			InstanceTemplate: proto.String("projects/" + i.Project + "/global/instanceTemplates/" + i.Template),
			Name:             proto.String(i.Name),
			TargetSize:       proto.Int32(int32(i.Count)),
		},
		Project: i.Project,
		Zone:    i.Zone,
	}
}

// CreateInstancesInput is the input for a CreatInstances operation.
type CreateInstancesInput struct {
	CountNodes        int
	CountCoordinators int
	ImageId           string
	InstanceType      string
	StateDiskSizeGB   int
	KubeEnv           string
}

type insertInstanceTemplateInput struct {
	Name                         string
	Network                      string
	Subnetwork                   string
	SecondarySubnetworkRangeName string
	ImageId                      string
	InstanceType                 string
	StateDiskSizeGB              int64
	Role                         string
	KubeEnv                      string
	Project                      string
	Zone                         string
	Region                       string
	UID                          string
}

func (i insertInstanceTemplateInput) insertInstanceTemplateRequest() *computepb.InsertInstanceTemplateRequest {
	req := computepb.InsertInstanceTemplateRequest{
		InstanceTemplateResource: &computepb.InstanceTemplate{
			Description: proto.String("This instance belongs to a Constellation cluster."),
			Name:        proto.String(i.Name),
			Properties: &computepb.InstanceProperties{
				ConfidentialInstanceConfig: &computepb.ConfidentialInstanceConfig{
					EnableConfidentialCompute: proto.Bool(true),
				},
				Description: proto.String("This instance belongs to a Constellation cluster."),
				Disks: []*computepb.AttachedDisk{
					{
						InitializeParams: &computepb.AttachedDiskInitializeParams{
							DiskSizeGb:  proto.Int64(10),
							SourceImage: proto.String(i.ImageId),
						},
						AutoDelete: proto.Bool(true),
						Boot:       proto.Bool(true),
						Mode:       proto.String(computepb.AttachedDisk_READ_WRITE.String()),
					},
					{
						InitializeParams: &computepb.AttachedDiskInitializeParams{
							DiskSizeGb: proto.Int64(i.StateDiskSizeGB),
						},
						AutoDelete: proto.Bool(true),
						DeviceName: proto.String("state-disk"),
						Mode:       proto.String(computepb.AttachedDisk_READ_WRITE.String()),
						Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
					},
				},
				MachineType: proto.String(i.InstanceType),
				Metadata: &computepb.Metadata{
					Items: []*computepb.Items{
						{
							Key:   proto.String("kube-env"),
							Value: proto.String(i.KubeEnv),
						},
						{
							Key:   proto.String("constellation-uid"),
							Value: proto.String(i.UID),
						},
						{
							Key:   proto.String("constellation-role"),
							Value: proto.String(i.Role),
						},
					},
				},
				NetworkInterfaces: []*computepb.NetworkInterface{
					{
						Network:    proto.String("projects/" + i.Project + "/global/networks/" + i.Network),
						Subnetwork: proto.String("regions/" + i.Region + "/subnetworks/" + i.Subnetwork),
						AccessConfigs: []*computepb.AccessConfig{
							{Type: proto.String(computepb.AccessConfig_ONE_TO_ONE_NAT.String())},
						},
					},
				},
				Scheduling: &computepb.Scheduling{
					OnHostMaintenance: proto.String(computepb.Scheduling_TERMINATE.String()),
				},
				ServiceAccounts: []*computepb.ServiceAccount{
					{
						Scopes: []string{
							"https://www.googleapis.com/auth/compute",
							"https://www.googleapis.com/auth/servicecontrol",
							"https://www.googleapis.com/auth/service.management.readonly",
							"https://www.googleapis.com/auth/devstorage.read_only",
							"https://www.googleapis.com/auth/logging.write",
							"https://www.googleapis.com/auth/monitoring.write",
							"https://www.googleapis.com/auth/trace.append",
						},
					},
				},
				ShieldedInstanceConfig: &computepb.ShieldedInstanceConfig{
					EnableIntegrityMonitoring: proto.Bool(true),
					EnableSecureBoot:          proto.Bool(true),
					EnableVtpm:                proto.Bool(true),
				},
				Tags: &computepb.Tags{
					Items: []string{"constellation-" + i.UID},
				},
			},
		},
		Project: i.Project,
	}

	// if there is an secondary IP range defined, we use it as an alias IP range
	if i.SecondarySubnetworkRangeName != "" {
		req.InstanceTemplateResource.Properties.NetworkInterfaces[0].AliasIpRanges = []*computepb.AliasIpRange{
			{
				IpCidrRange:         proto.String("/24"),
				SubnetworkRangeName: proto.String(i.SecondarySubnetworkRangeName),
			},
		}
	}

	return &req
}