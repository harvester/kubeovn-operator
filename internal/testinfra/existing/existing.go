package existing

import (
	"context"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ExistingCluster struct{}

func (e *ExistingCluster) CreateCluster(ctx context.Context) error {
	// no op since we are going to read the existing kubeconfig and return the same
	return nil
}

func (e *ExistingCluster) DeleteCluster(ctx context.Context) error {
	// no op since are not going to delete the existing cluster
	return nil
}

// load existing kubeconfig file
func (e *ExistingCluster) GetKubeConfig(ctx context.Context) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}
