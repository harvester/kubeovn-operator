package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
)

func testNode(name, ip string, nodeLabels map[string]string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: nodeLabels},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: ip}},
		},
	}
}

// harvesterNodes returns 2 control-plane nodes and 1 witness node
func harvesterNodes() []client.Object {
	cp := map[string]string{
		"node-role.kubernetes.io/control-plane": "true",
		"node-role.kubernetes.io/etcd":          "true",
	}
	witness := map[string]string{
		"node-role.kubernetes.io/etcd": "true",
	}
	return []client.Object{
		testNode("node-3", "10.0.0.3", witness),
		testNode("node-2", "10.0.0.2", cp),
		testNode("node-1", "10.0.0.1", cp),
	}
}

func newTestConfigurationReconciler(t *testing.T, objs ...client.Object) *ConfigurationReconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kubeovniov1.AddToScheme(scheme))
	return &ConfigurationReconciler{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Scheme:        scheme,
		EventRecorder: record.NewFakeRecorder(10),
		Log:           logr.Discard(),
	}
}

func newTestConfiguration(nodesLabel string) *kubeovniov1.Configuration {
	return &kubeovniov1.Configuration{
		ObjectMeta: metav1.ObjectMeta{Name: kubeovniov1.DefaultConfigurationName, Namespace: "kube-system"},
		Spec: kubeovniov1.ConfigurationSpec{
			MasterNodesLabel: "node-role.kubernetes.io/control-plane=true",
			OVNCentral:       kubeovniov1.OVNCentralResourceSpec{NodesLabel: nodesLabel},
		},
	}
}

func Test_FindMasterNodesWithoutOVNCentralNodesLabel(t *testing.T) {
	assert := require.New(t)
	r := newTestConfigurationReconciler(t, harvesterNodes()...)
	cfg := newTestConfiguration("")
	assert.NoError(r.findMasterNodes(context.TODO(), cfg))
	assert.Equal([]string{"10.0.0.1", "10.0.0.2"}, cfg.Status.MatchingNodeAddresses)
	assert.Equal(cfg.Status.MatchingNodeAddresses, cfg.Status.OVNCentralNodeAddresses)
}

func Test_FindMasterNodesWithOVNCentralNodesLabel(t *testing.T) {
	assert := require.New(t)
	r := newTestConfigurationReconciler(t, harvesterNodes()...)

	for _, nodesLabel := range []string{"node-role.kubernetes.io/etcd=true", "node-role.kubernetes.io/etcd"} {
		cfg := newTestConfiguration(nodesLabel)
		assert.NoError(r.findMasterNodes(context.TODO(), cfg))
		assert.Equal([]string{"10.0.0.1", "10.0.0.2"}, cfg.Status.MatchingNodeAddresses, nodesLabel)
		assert.Equal([]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}, cfg.Status.OVNCentralNodeAddresses, nodesLabel)
	}
}

func Test_FindMasterNodesOVNCentralNodesLabelIgnoredForHCP(t *testing.T) {
	assert := require.New(t)
	r := newTestConfigurationReconciler(t, harvesterNodes()...)
	cfg := newTestConfiguration("node-role.kubernetes.io/etcd=true")
	cfg.Spec.OVNCentral.HCP.Enabled = ptr.To(true)
	assert.NoError(r.findMasterNodes(context.TODO(), cfg))
	assert.Equal(cfg.Status.MatchingNodeAddresses, cfg.Status.OVNCentralNodeAddresses)
}

func Test_FindMasterNodesOVNCentralNodesLabelNoMatch(t *testing.T) {
	assert := require.New(t)
	r := newTestConfigurationReconciler(t, harvesterNodes()...)
	cfg := newTestConfiguration("does-not-exist=true")
	cfg.Status.OVNCentralNodeAddresses = []string{"10.0.0.9"}
	assert.NoError(r.findMasterNodes(context.TODO(), cfg))
	assert.Equal([]string{"10.0.0.1", "10.0.0.2"}, cfg.Status.MatchingNodeAddresses)
	assert.Empty(cfg.Status.OVNCentralNodeAddresses)
}

func Test_NodeChangedPredicate(t *testing.T) {
	assert := require.New(t)
	p := nodeChangedPredicate()
	oldNode := testNode("node-1", "10.0.0.1", map[string]string{"a": "b"})

	same := oldNode.DeepCopy()
	same.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}
	assert.False(p.Update(event.UpdateEvent{ObjectOld: oldNode, ObjectNew: same}))

	relabeled := oldNode.DeepCopy()
	relabeled.Labels["node-role.kubernetes.io/etcd"] = "true"
	assert.True(p.Update(event.UpdateEvent{ObjectOld: oldNode, ObjectNew: relabeled}))

	readdressed := oldNode.DeepCopy()
	readdressed.Status.Addresses[0].Address = "10.0.0.10"
	assert.True(p.Update(event.UpdateEvent{ObjectOld: oldNode, ObjectNew: readdressed}))

	assert.True(p.Create(event.CreateEvent{Object: oldNode}))
	assert.True(p.Delete(event.DeleteEvent{Object: oldNode}))
}
