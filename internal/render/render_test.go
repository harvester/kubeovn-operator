package render

import (
	"fmt"
	"os"
	"testing"

	ovnoperatorv1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/harvester/kubeovn-operator/internal/templates"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func generateConfigObject() (*ovnoperatorv1.Configuration, error) {
	content, err := os.ReadFile("../../config/samples/kubeovn.io_v1_configuration.yaml")
	if err != nil {
		return nil, fmt.Errorf("error reading sample configuration file: %v", err)
	}
	config := &ovnoperatorv1.Configuration{}
	err = yaml.Unmarshal(content, config)
	return config, err
}

func Test_ObjectRendering(t *testing.T) {
	assert := require.New(t)
	c, err := generateConfigObject()
	assert.NoError(err, "expected no error while generating config object")
	for objectType, objectList := range templates.OrderedObjectList {
		returnedObjects, err := GenerateObjects(objectList, c, objectType, nil, "v1.14.0", "caCertString")
		assert.NoError(err, "expected no error while generating object", objectType)
		for _, object := range returnedObjects {
			assert.NotEmpty(object.GetName())
		}
	}
}

func Test_CleanupRendering(t *testing.T) {
	assert := require.New(t)
	nodeIP := "192.168.1.128"
	for _, v := range []string{templates.CleanupNBDB, templates.CleanupSBDB} {
		_, err := generateScript(nodeIP, v)
		assert.NoError(err)
	}
}

func Test_MasterNodeAffinityRendering(t *testing.T) {
	assert := require.New(t)
	config := &ovnoperatorv1.Configuration{
		Spec: ovnoperatorv1.ConfigurationSpec{
			MasterNodesLabel: "node-role.kubernetes.io/control-plane=true",
		},
	}
	result, err := generateMasterNodeAffinity(config)
	assert.NoError(err)
	assert.Contains(result, "key: node-role.kubernetes.io/control-plane")

}

func Test_MasterNodeAffinityRenderingWithEmptyLabel(t *testing.T) {
	assert := require.New(t)
	config := &ovnoperatorv1.Configuration{}
	result, err := generateMasterNodeAffinity(config)
	assert.NoError(err)
	assert.Len(result, 0, "expected empty result when MasterNodesLabel is not set")
}

func Test_ServiceIPFamilyPolicy(t *testing.T) {
	// Services whose templates opt into PreferDualStack when netStack is dual_stack.
	dualStackServices := []string{
		"kube-ovn-controller", "kube-ovn-monitor", "ovn-nb", "ovn-northd",
		"kube-ovn-cni", "kube-ovn-pinger", "ovn-sb",
	}
	for _, netStack := range []string{"ipv4", "ipv6", "dual_stack"} {
		t.Run(netStack, func(t *testing.T) {
			assert := require.New(t)
			c, err := generateConfigObject()
			assert.NoError(err)
			c.Spec.Networking.NetStack = netStack

			policies := map[string]string{}
			for _, tmpl := range templates.ServicesList {
				objs, err := GenerateObjects([]string{tmpl}, c, &corev1.Service{}, nil, "v1.14.0", "caCertString")
				assert.NoError(err)
				for _, obj := range objs {
					svc, ok := obj.(*corev1.Service)
					assert.True(ok, "expected *corev1.Service, got %T", obj)
					policies[svc.Name] = ""
					if p := svc.Spec.IPFamilyPolicy; p != nil {
						policies[svc.Name] = string(*p)
					}
				}
			}
			for _, name := range dualStackServices {
				got, ok := policies[name]
				assert.True(ok, "service %s was not rendered", name)
				if netStack == "dual_stack" {
					assert.Equal("PreferDualStack", got, "service %s", name)
				} else if name != "kube-ovn-controller" && name != "kube-ovn-monitor" {
					assert.Empty(got, "service %s", name)
				}
			}
		})
	}
}
