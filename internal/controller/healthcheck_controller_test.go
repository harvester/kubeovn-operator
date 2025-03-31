package controller

import (
	"context"
	"os"
	"testing"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_ScriptExecution(t *testing.T) {
	assert := require.New(t)

	assert.NoError(os.Setenv("KUBECONFIG", "/Users/gauravmehta/.config/k3d/kubeconfig-kubeovn-test.yaml"))
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	assert.NoError(err)
	schema := runtime.NewScheme()
	assert.NoError(clientgoscheme.AddToScheme(schema))
	cl, err := client.New(config, client.Options{})
	assert.NoError(err)
	result, err := executeOVNCentralCommand(context.TODO(), kubeovniov1.SBCheckScript, kubeovniov1.SBLeaderLabel, cl, config, "kube-system")
	assert.NoError(err)
	t.Log(string(result))
}
