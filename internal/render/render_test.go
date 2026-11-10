package render

import (
	"fmt"
	"os"
	"slices"
	"testing"

	ovnoperatorv1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/harvester/kubeovn-operator/internal/templates"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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

const (
	testMasterLabel     = "node-role.kubernetes.io/control-plane=true"
	testOVNCentralLabel = "node-role.kubernetes.io/etcd=true"
)

var (
	testMasterAddresses     = []string{"10.0.0.1", "10.0.0.2"}
	testOVNCentralAddresses = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
)

// renderWorkloads renders deployments and daemonsets for the configuration indexed by name
func renderWorkloads(t *testing.T, c *ovnoperatorv1.Configuration) (map[string]*appsv1.Deployment, map[string]*appsv1.DaemonSet) {
	t.Helper()
	assert := require.New(t)
	deployments := map[string]*appsv1.Deployment{}
	objs, err := GenerateObjects(templates.DeploymentList, c.DeepCopy(), &appsv1.Deployment{}, nil, "v1.14.0", "caCertString")
	assert.NoError(err)
	for _, obj := range objs {
		d, ok := obj.(*appsv1.Deployment)
		assert.True(ok, "expected *appsv1.Deployment, got %T", obj)
		deployments[d.Name] = d
	}
	daemonsets := map[string]*appsv1.DaemonSet{}
	objs, err = GenerateObjects(templates.DaemonsetList, c.DeepCopy(), &appsv1.DaemonSet{}, nil, "v1.14.0", "caCertString")
	assert.NoError(err)
	for _, obj := range objs {
		d, ok := obj.(*appsv1.DaemonSet)
		assert.True(ok, "expected *appsv1.DaemonSet, got %T", obj)
		daemonsets[d.Name] = d
	}
	return deployments, daemonsets
}

func containerEnv(t *testing.T, spec corev1.PodSpec, container, name string) string {
	t.Helper()
	for _, c := range spec.Containers {
		if c.Name != container {
			continue
		}
		for _, e := range c.Env {
			if e.Name == name {
				return e.Value
			}
		}
	}
	t.Fatalf("env %s not found in container %s", name, container)
	return ""
}

func nodeAffinityKey(t *testing.T, spec corev1.PodSpec) string {
	t.Helper()
	require.NotNil(t, spec.Affinity)
	require.NotNil(t, spec.Affinity.NodeAffinity)
	terms := spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 1)
	require.Len(t, terms[0].MatchExpressions, 1)
	return terms[0].MatchExpressions[0].Key
}

func newNodesLabelTestConfig(t *testing.T) *ovnoperatorv1.Configuration {
	t.Helper()
	c, err := generateConfigObject()
	require.NoError(t, err)
	c.Namespace = "kube-system"
	c.Spec.MasterNodesLabel = testMasterLabel
	c.Status.MatchingNodeAddresses = slices.Clone(testMasterAddresses)
	return c
}

func Test_OVNCentralNodeAffinityRendering(t *testing.T) {
	assert := require.New(t)
	config := &ovnoperatorv1.Configuration{
		Spec: ovnoperatorv1.ConfigurationSpec{
			MasterNodesLabel: testMasterLabel,
		},
	}
	masterAffinity, err := generateMasterNodeAffinity(config)
	assert.NoError(err)

	// fallback to masterNodesLabel when nodesLabel is empty
	result, err := generateOVNCentralNodeAffinity(config)
	assert.NoError(err)
	assert.Equal(masterAffinity, result)

	config.Spec.OVNCentral.NodesLabel = testOVNCentralLabel
	result, err = generateOVNCentralNodeAffinity(config)
	assert.NoError(err)
	assert.Contains(result, "key: node-role.kubernetes.io/etcd")
	assert.Contains(result, `- "true"`)

	// key only label results in Exists operator
	config.Spec.OVNCentral.NodesLabel = "node-role.kubernetes.io/etcd"
	result, err = generateOVNCentralNodeAffinity(config)
	assert.NoError(err)
	assert.Contains(result, "key: node-role.kubernetes.io/etcd")
	assert.Contains(result, "operator: Exists")

	// master affinity is never affected by nodesLabel
	result, err = generateMasterNodeAffinity(config)
	assert.NoError(err)
	assert.Equal(masterAffinity, result)
}

// Test_OVNCentralNodesLabelEmptyUnchanged ensures that rendering is unchanged when nodesLabel is
// not set, even if status.ovnCentralNodeAddresses contains a different set of addresses
func Test_OVNCentralNodesLabelEmptyUnchanged(t *testing.T) {
	assert := require.New(t)
	base := newNodesLabelTestConfig(t)
	withStatus := base.DeepCopy()
	withStatus.Status.OVNCentralNodeAddresses = slices.Clone(testOVNCentralAddresses)

	assert.Equal(generateIncludeValues(base), generateIncludeValues(withStatus))

	baseDeployments, baseDaemonsets := renderWorkloads(t, base)
	deployments, daemonsets := renderWorkloads(t, withStatus)
	assert.Equal(baseDeployments, deployments)
	assert.Equal(baseDaemonsets, daemonsets)

	central := deployments["ovn-central"]
	assert.NotNil(central)
	assert.Equal(int32(2), *central.Spec.Replicas)
	assert.Equal("10.0.0.1,10.0.0.2", containerEnv(t, central.Spec.Template.Spec, "ovn-central", "NODE_IPS"))
	assert.Equal("node-role.kubernetes.io/control-plane", nodeAffinityKey(t, central.Spec.Template.Spec))
}

func Test_OVNCentralNodesLabelSet(t *testing.T) {
	assert := require.New(t)
	base := newNodesLabelTestConfig(t)
	c := base.DeepCopy()
	c.Spec.OVNCentral.NodesLabel = testOVNCentralLabel
	c.Status.OVNCentralNodeAddresses = slices.Clone(testOVNCentralAddresses)
	baseDeployments, _ := renderWorkloads(t, base)
	deployments, daemonsets := renderWorkloads(t, c)

	// ovn-central follows nodesLabel
	central := deployments["ovn-central"]
	assert.NotNil(central)
	assert.Equal(int32(3), *central.Spec.Replicas)
	assert.Equal("10.0.0.1,10.0.0.2,10.0.0.3", containerEnv(t, central.Spec.Template.Spec, "ovn-central", "NODE_IPS"))
	assert.Equal("node-role.kubernetes.io/etcd", nodeAffinityKey(t, central.Spec.Template.Spec))

	// kube-ovn-controller keeps the master node replica count and its affinity but connects to all raft members
	controller := deployments["kube-ovn-controller"]
	assert.NotNil(controller)
	assert.Equal(int32(2), *controller.Spec.Replicas)
	assert.Equal(baseDeployments["kube-ovn-controller"].Spec.Template.Spec.Affinity, controller.Spec.Template.Spec.Affinity)
	assert.Equal("10.0.0.1,10.0.0.2,10.0.0.3", containerEnv(t, controller.Spec.Template.Spec, "kube-ovn-controller", "OVN_DB_IPS"))

	// kube-ovn-monitor keeps the master node affinity
	monitor := deployments["kube-ovn-monitor"]
	assert.NotNil(monitor)
	assert.Equal("node-role.kubernetes.io/control-plane", nodeAffinityKey(t, monitor.Spec.Template.Spec))

	// ovs-ovn connects to all raft members
	ovsovn := daemonsets["ovs-ovn"]
	assert.NotNil(ovsovn)
	assert.Equal("10.0.0.1,10.0.0.2,10.0.0.3", containerEnv(t, ovsovn.Spec.Template.Spec, "openvswitch", "OVN_DB_IPS"))
}

func Test_OVNCentralNodesLabelIgnoredInSingleMode(t *testing.T) {
	assert := require.New(t)
	base := newNodesLabelTestConfig(t)
	base.Spec.OVNCentralMode = "single"
	c := base.DeepCopy()
	c.Spec.OVNCentral.NodesLabel = testOVNCentralLabel
	c.Status.OVNCentralNodeAddresses = slices.Clone(testOVNCentralAddresses)

	baseDeployments, _ := renderWorkloads(t, base)
	deployments, _ := renderWorkloads(t, c)
	assert.Equal(baseDeployments["ovn-central"], deployments["ovn-central"])
	assert.Equal(int32(1), *deployments["ovn-central"].Spec.Replicas)
}
