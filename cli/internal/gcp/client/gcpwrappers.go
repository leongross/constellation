package client

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"
	admin "cloud.google.com/go/iam/admin/apiv1"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"github.com/googleapis/gax-go/v2"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

type instanceClient struct {
	*compute.InstancesClient
}

func (c *instanceClient) Close() error {
	return c.InstancesClient.Close()
}

func (c *instanceClient) List(ctx context.Context, req *computepb.ListInstancesRequest,
	opts ...gax.CallOption,
) InstanceIterator {
	return c.InstancesClient.List(ctx, req)
}

type firewallsClient struct {
	*compute.FirewallsClient
}

func (c *firewallsClient) Close() error {
	return c.FirewallsClient.Close()
}

func (c *firewallsClient) Delete(ctx context.Context, req *computepb.DeleteFirewallRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.FirewallsClient.Delete(ctx, req)
}

func (c *firewallsClient) Insert(ctx context.Context, req *computepb.InsertFirewallRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.FirewallsClient.Insert(ctx, req)
}

type networksClient struct {
	*compute.NetworksClient
}

func (c *networksClient) Close() error {
	return c.NetworksClient.Close()
}

func (c *networksClient) Insert(ctx context.Context, req *computepb.InsertNetworkRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.NetworksClient.Insert(ctx, req)
}

func (c *networksClient) Delete(ctx context.Context, req *computepb.DeleteNetworkRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.NetworksClient.Delete(ctx, req)
}

type subnetworksClient struct {
	*compute.SubnetworksClient
}

func (c *subnetworksClient) Close() error {
	return c.SubnetworksClient.Close()
}

func (c *subnetworksClient) Insert(ctx context.Context, req *computepb.InsertSubnetworkRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.SubnetworksClient.Insert(ctx, req)
}

func (c *subnetworksClient) Delete(ctx context.Context, req *computepb.DeleteSubnetworkRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.SubnetworksClient.Delete(ctx, req)
}

type instanceTemplateClient struct {
	*compute.InstanceTemplatesClient
}

func (c *instanceTemplateClient) Close() error {
	return c.InstanceTemplatesClient.Close()
}

func (c *instanceTemplateClient) Delete(ctx context.Context, req *computepb.DeleteInstanceTemplateRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.InstanceTemplatesClient.Delete(ctx, req)
}

func (c *instanceTemplateClient) Insert(ctx context.Context, req *computepb.InsertInstanceTemplateRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.InstanceTemplatesClient.Insert(ctx, req)
}

type instanceGroupManagersClient struct {
	*compute.InstanceGroupManagersClient
}

func (c *instanceGroupManagersClient) Close() error {
	return c.InstanceGroupManagersClient.Close()
}

func (c *instanceGroupManagersClient) Delete(ctx context.Context, req *computepb.DeleteInstanceGroupManagerRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.InstanceGroupManagersClient.Delete(ctx, req)
}

func (c *instanceGroupManagersClient) Insert(ctx context.Context, req *computepb.InsertInstanceGroupManagerRequest,
	opts ...gax.CallOption,
) (Operation, error) {
	return c.InstanceGroupManagersClient.Insert(ctx, req)
}

func (c *instanceGroupManagersClient) ListManagedInstances(ctx context.Context, req *computepb.ListManagedInstancesInstanceGroupManagersRequest,
	opts ...gax.CallOption,
) ManagedInstanceIterator {
	return c.InstanceGroupManagersClient.ListManagedInstances(ctx, req)
}

type iamClient struct {
	*admin.IamClient
}

func (c *iamClient) Close() error {
	return c.IamClient.Close()
}

func (c *iamClient) CreateServiceAccount(ctx context.Context, req *adminpb.CreateServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	return c.IamClient.CreateServiceAccount(ctx, req)
}

func (c *iamClient) CreateServiceAccountKey(ctx context.Context, req *adminpb.CreateServiceAccountKeyRequest, opts ...gax.CallOption) (*adminpb.ServiceAccountKey, error) {
	return c.IamClient.CreateServiceAccountKey(ctx, req)
}

func (c *iamClient) DeleteServiceAccount(ctx context.Context, req *adminpb.DeleteServiceAccountRequest, opts ...gax.CallOption) error {
	return c.IamClient.DeleteServiceAccount(ctx, req)
}

type projectsClient struct {
	*resourcemanager.ProjectsClient
}

func (c *projectsClient) Close() error {
	return c.ProjectsClient.Close()
}

func (c *projectsClient) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return c.ProjectsClient.GetIamPolicy(ctx, req)
}

func (c *projectsClient) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return c.ProjectsClient.SetIamPolicy(ctx, req)
}