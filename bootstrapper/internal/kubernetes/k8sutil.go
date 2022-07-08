package kubernetes

import (
	"context"

	"github.com/edgelesssys/constellation/bootstrapper/internal/kubernetes/k8sapi"
	"github.com/edgelesssys/constellation/bootstrapper/internal/kubernetes/k8sapi/resources"
	"go.uber.org/zap"
)

type clusterUtil interface {
	InstallComponents(ctx context.Context, version string) error
	InitCluster(ctx context.Context, initConfig []byte, logger *zap.Logger) error
	JoinCluster(ctx context.Context, joinConfig []byte, logger *zap.Logger) error
	SetupPodNetwork(context.Context, k8sapi.SetupPodNetworkInput) error
	SetupAccessManager(kubectl k8sapi.Client, sshUsers resources.Marshaler) error
	SetupAutoscaling(kubectl k8sapi.Client, clusterAutoscalerConfiguration resources.Marshaler, secrets resources.Marshaler) error
	SetupJoinService(kubectl k8sapi.Client, joinServiceConfiguration resources.Marshaler) error
	SetupCloudControllerManager(kubectl k8sapi.Client, cloudControllerManagerConfiguration resources.Marshaler, configMaps resources.Marshaler, secrets resources.Marshaler) error
	SetupCloudNodeManager(kubectl k8sapi.Client, cloudNodeManagerConfiguration resources.Marshaler) error
	SetupKMS(kubectl k8sapi.Client, kmsConfiguration resources.Marshaler) error
	SetupVerificationService(kubectl k8sapi.Client, verificationServiceConfiguration resources.Marshaler) error
	SetupGCPGuestAgent(kubectl k8sapi.Client, gcpGuestAgentConfiguration resources.Marshaler) error
	StartKubelet() error
	RestartKubelet() error
	FixCilium(nodeNameK8s string)
}