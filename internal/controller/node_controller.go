package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/harvester/kubeovn-operator/internal/executor"
	"github.com/harvester/kubeovn-operator/internal/render"
)

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;create;update;patch;delete

type NodeReconciler struct {
	client.Client
	RestConfig    *rest.Config
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	Namespace     string
	Log           logr.Logger
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named("kubeovn-node-controller").Complete(r)
}

// Reconcile finds nodes matching the configuration .spec.masterNodesLabel
// and adds a finalizer to all nodes. This finalizer is then used during node deletion
// events to trigger cleanup of ovn north and south bound databases
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nodeObj := &corev1.Node{}
	err := r.Get(ctx, req.NamespacedName, nodeObj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.WithValues("name", req.Name).Info("node not found")
			return ctrl.Result{}, nil
		}
	}

	node := nodeObj.DeepCopy()
	// fetch configuration object and identify matching nodes
	config, err := fetchKubeovnConfig(ctx, r.Client, r.Namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("waiting for kubeovn configuration to be created, nothing to do")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !config.DeletionTimestamp.IsZero() {
		// config is being deleted, no more node reconcile is needed
		return ctrl.Result{}, nil
	}

	if node.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer) {
			controllerutil.AddFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer)
			return ctrl.Result{}, r.Patch(ctx, node, client.MergeFrom(nodeObj))
		}
		return ctrl.Result{}, nil
	}

	// reconcile ovn north and south databases and check if there is a condition matching
	r.Log.WithValues("node", node.GetName()).Info("reconilling deletion of node")
	return r.reconcileNodeDeletion(ctx, config, node)
}

// fetchKubeovnConfig fetches the default configuration
func fetchKubeovnConfig(ctx context.Context, client client.Client, namespace string) (*kubeovniov1.Configuration, error) {
	config := &kubeovniov1.Configuration{}
	err := client.Get(ctx, types.NamespacedName{Name: kubeovniov1.DefaultConfigurationName, Namespace: namespace}, config)
	return config, err
}

// reconcileNodeDeletion will delete the node entry from the ovn database and remove finalizer
// allowing node to be removed from apiserver
func (r *NodeReconciler) reconcileNodeDeletion(ctx context.Context, config *kubeovniov1.Configuration, node *corev1.Node) (ctrl.Result, error) {
	nodeObj := node.DeepCopy()

	nodeIP := nodeInternalIP(*nodeObj)
	if len(nodeIP) == 0 {
		// should not be possible but lets return and ignore this node
		r.Log.Error(errors.New("node has no internal ip so requeuing, manual cleanup of finalizer may be needed"), "node", node.GetName())
		return ctrl.Result{}, fmt.Errorf("node has no internal ip so ignoring %s", node.GetName())
	}
	if slices.Contains(config.Status.MatchingNodeAddresses, nodeIP) {
		r.Log.WithValues("name", node.Name).Info("node matches master node label, trigger nb/sb db cleanup")
		if err := r.reconcileOVNCentralState(ctx, nodeIP); err != nil {
			return ctrl.Result{}, err
		}
		// reconcile config and update master node details
		// this will result in regeneration of templates
		configObj := config.DeepCopy()
		config.Status.MatchingNodeAddresses = removeElement(config.Status.MatchingNodeAddresses, nodeIP)
		if err := r.Client.Status().Patch(ctx, config, client.MergeFrom(configObj)); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileOVSOVNState(ctx, nodeIP); err != nil {
		return ctrl.Result{}, nil
	}
	// perform ovn northbound and southbound db cleanup operations
	// remove finalizer to allow node to be cleaned up
	if controllerutil.ContainsFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer) {
		controllerutil.RemoveFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer)
		return ctrl.Result{}, r.Patch(ctx, node, client.MergeFrom(nodeObj))
	}
	return ctrl.Result{}, nil
}

// reconcileOVNCentralState attempts to perform the kubeovn cleanup procedure for ovn north and south
// databases running on ovn-central as documented at https://kubeovn.github.io/docs/v1.12.x/en/ops/change-ovn-central-node/
func (r *NodeReconciler) reconcileOVNCentralState(ctx context.Context, nodeIP string) error {
	// generate nb cleanup template
	r.Log.WithValues("nodeIP", nodeIP).Info("removing node from ovn northbound/southbound database")
	script, err := render.GenerateNorthBoundCleanupScript(nodeIP)
	if err != nil {
		return fmt.Errorf("error generating northbound cleanup script using ip %s: %v", nodeIP, err)
	}

	r.Log.WithValues("nodeIP", nodeIP).Info(fmt.Sprintf("north bound script %s\n", script))
	if err := r.executeRemoteScriptOnLeader(ctx, script, kubeovniov1.NBLeaderLabel, nodeIP); err != nil {
		return fmt.Errorf("error executing northbound cleanup script on node %s: %v", nodeIP, err)
	}

	// generate sb cleanup template
	script, err = render.GenerateSouthBoundCleanupScript(nodeIP)
	if err != nil {
		return fmt.Errorf("error generating southbound cleanup script using ip %s: %v", nodeIP, err)
	}
	return r.executeRemoteScriptOnLeader(ctx, script, kubeovniov1.SBLeaderLabel, nodeIP)
}

// reconcileOVSOVNState cleans up chassis-id from the ovn southbound db when any node is removed
func (r *NodeReconciler) reconcileOVSOVNState(ctx context.Context, hostname string) error {
	r.Log.WithValues("name", hostname).Info("removing node chassis entry")
	script, err := render.GenerateChassisCleanupScript(hostname)
	if err != nil {
		return fmt.Errorf("error rendering chassis cleanup script: %v", err)
	}
	return r.executeRemoteScriptOnLeader(ctx, script, kubeovniov1.SBLeaderLabel, hostname)
}

// podList is a helper function to help identify NB/SB leader pods
func podList(ctx context.Context, label string, k8sClient client.Client, namespace string) (*corev1.PodList, error) {
	selector, err := labels.Parse(label)
	if err != nil {
		return nil, fmt.Errorf("error unable to parse label list %s: %v", label, err)
	}

	podList := &corev1.PodList{}
	err = k8sClient.List(ctx, podList, &client.ListOptions{LabelSelector: selector, Namespace: namespace})
	return podList, err
}

// executeRemoteScriptOnLeader helps users execute remote scripts on a specific pod and return results
// it emulates kubectl exec against the OVNCentral pod
func (r *NodeReconciler) executeRemoteScriptOnLeader(ctx context.Context, script string, label string, node string) error {
	result, err := executeOVNCentralCommand(ctx, script, label, r.Client, r.RestConfig, r.Namespace)
	if err != nil {
		return fmt.Errorf("Error during southbound cleanup command execution %s: %v", string(result), err)
	}
	r.Log.WithValues("node", node).Info(string(result))
	return nil
}

// removeElement is a simple function to remove a matching element from an array of strings
func removeElement(elements []string, element string) []string {
	for i := range elements {
		if elements[i] == element {
			return append(elements[:i], elements[i+1:]...)
		}
	}
	return elements
}

// executeOVNCentralCommand is a wrapper to abstract OVNCentralCommand execution
func executeOVNCentralCommand(ctx context.Context, script string, label string, k8sClient client.Client, restConfig *rest.Config, namespace string) ([]byte, error) {
	podList, err := podList(ctx, label, k8sClient, namespace)
	if err != nil {
		return nil, fmt.Errorf("error generating pod list when checking for label %s: %v", label, err)
	}

	if len(podList.Items) == 0 || len(podList.Items) > 1 {
		return nil, fmt.Errorf("expected to find only one leader pod, but found %d, requeuing until condition is met", len(podList.Items))
	}
	pod := podList.Items[0]
	podExecutor, err := executor.NewRemoteCommandExecutor(ctx, restConfig, &pod)
	if err != nil {
		return nil, fmt.Errorf("error generating new remote command executor: %v", err)
	}
	return podExecutor.Run(kubeovniov1.OVNCentralContainerName, script)
}
