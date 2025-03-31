package k3d

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Test_CreateAndVerifyCluster(t *testing.T) {
	assert := require.New(t)
	ctx := context.TODO()
	cluster := &K3dCluster{}
	err := cluster.CreateCluster(ctx)
	assert.NoError(err, "expected no error during cluster creation")
	restConfig, err := cluster.GetKubeConfig(ctx)
	assert.NoError(err, "expected no error during kubeconfig fetch")
	client, err := kubernetes.NewForConfig(restConfig)
	assert.NoError(err, "expected no error during client creation")
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	assert.NoError(err, "expected no error while fetching nodes")
	assert.Len(nodes.Items, 3, "expected to find 3 nodes in the test cluster")
}

func Test_DeleteCluster(t *testing.T) {
	assert := require.New(t)
	ctx := context.TODO()
	cluster := &K3dCluster{}
	err := cluster.DeleteCluster(ctx)
	assert.NoError(err, "expected no error during cluster deletion")
}
