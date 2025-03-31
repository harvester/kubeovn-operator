package testinfra

import (
	"context"
	"os"

	"k8s.io/client-go/rest"

	"github.com/harvester/kubeovn-operator/internal/testinfra/existing"
	"github.com/harvester/kubeovn-operator/internal/testinfra/k3d"
)

type TestCluster interface {
	CreateCluster(ctx context.Context) error
	DeleteCluster(ctx context.Context) error
	GetKubeConfig(ctx context.Context) (*rest.Config, error)
}

func NewTestCluster() TestCluster {
	if os.Getenv("USE_EXISTING_CLUSTER") == "true" {
		return &existing.ExistingCluster{}
	}
	return &k3d.K3dCluster{}
}
