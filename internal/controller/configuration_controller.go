/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubeovniov1 "github.com/harvester/kubeovn-operator/api/v1"
	"github.com/harvester/kubeovn-operator/internal/render"
	"github.com/harvester/kubeovn-operator/internal/templates"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ConfigurationReconciler reconciles a Configuration object
type ConfigurationReconciler struct {
	client.Client
	RestConfig    *rest.Config
	Scheme        *runtime.Scheme
	Namespace     string
	EventRecorder record.EventRecorder
	Log           logr.Logger
	Version       string
}

type reconcileFuncs func(context.Context, *kubeovniov1.Configuration) error

// +kubebuilder:rbac:groups=kubeovn.io,resources=configurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeovn.io,resources=configurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeovn.io,resources=configurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/scale,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeovn.io,resources='*',verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Configuration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *ConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	configObj := &kubeovniov1.Configuration{}
	err := r.Get(ctx, req.NamespacedName, configObj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.WithValues("name", configObj.Name).Info("configuration not found")
			return ctrl.Result{}, nil
		}
		r.Log.WithValues("name", configObj.Name).Error(err, "error fetching object")
		return ctrl.Result{}, err
	}

	config := configObj.DeepCopy()
	// if deletiontimestamp is set, then no further processing is needed as we let k8s gc the associated objects
	if config.DeletionTimestamp != nil {
		return reconcile.Result{}, r.deleteClusterScopedReference(ctx, config)
	}

	// check and add finalizer
	if !controllerutil.ContainsFinalizer(config, kubeovniov1.KubeOVNConfigurationFinalizer) {
		controllerutil.AddFinalizer(config, kubeovniov1.KubeOVNConfigurationFinalizer)
		return ctrl.Result{}, r.Client.Patch(ctx, config, client.MergeFrom(configObj))
	}

	reconcileSteps := []reconcileFuncs{r.initializeConditions, r.reconcileClusterScopedReference, r.findMasterNodes, r.applyObject}

	for _, v := range reconcileSteps {
		if err := v(ctx, config); err != nil {
			return ctrl.Result{}, err
		}
	}

	if reflect.DeepEqual(configObj.Status, config.Status) {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.Client.Status().Patch(ctx, config, client.MergeFrom(configObj))
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&kubeovniov1.Configuration{}).
		Named("kubeovn-configuration-controller")
	return r.AddWatches(b).Complete(r)
}

// applyObject will check if Config object is not already deploying. If a change is needed then it triggers
// create/update of associated objects
func (r *ConfigurationReconciler) applyObject(ctx context.Context, config *kubeovniov1.Configuration) error {
	if len(config.Status.MatchingNodeAddresses) == 0 {
		r.Log.WithValues("name", config.Name).Info("waiting for matching master node requirement to be met")
		return nil
	}
	if config.Status.Status == kubeovniov1.ConfigurationStatusDeploying {
		r.Log.WithValues("name", config.Name).Info("skipping applying objects as objects are already deploying")
		return nil
	}

	fakeNSObj := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: kubeovniov1.KubeOVNFakeNamespace}, fakeNSObj)
	if err != nil {
		return fmt.Errorf("error looking up fake namespaced object: %v", err)
	}

	for objectType, objectList := range templates.OrderedObjectList {
		r.Log.WithValues("objectType", objectType).Info("processing object type")
		// chceck if objectType is a clusterscoped object so we can defined correct ownership
		namespaced, err := apiutil.IsObjectNamespaced(objectType, r.Scheme, r.Client.RESTMapper())
		if err != nil {
			return fmt.Errorf("unable to identify if objecttype %s is namedspaced: %v", objectType.GetObjectKind(), err)
		}
		objs, err := render.GenerateObjects(objectList, config, objectType, r.RestConfig, r.Version)
		if err != nil {
			return fmt.Errorf("error during object generation for type %s: %v", objectType.GetObjectKind().GroupVersionKind(), err)
		}
		for _, obj := range objs {
			var ownerObj client.Object
			if namespaced {
				ownerObj = config
			} else {
				ownerObj = fakeNSObj
			}
			err = controllerutil.SetControllerReference(ownerObj, obj, r.Scheme)
			if err != nil {
				return fmt.Errorf("error setting controller reference on object %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
			}

			if err = r.reconcileObject(ctx, obj); err != nil {
				return fmt.Errorf("error reconcilling object %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
			}
		}
	}
	config.Status.Status = kubeovniov1.ConfigurationStatusDeployed
	return nil
}

// reconcileObject will mimic kubectl apply to apply objects
func (r *ConfigurationReconciler) reconcileObject(ctx context.Context, obj client.Object) error {
	var err error
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("error convering new object %s to unstructured object %v", obj.GetName(), err)
	}
	return r.Patch(ctx, unstructuredObj, client.Apply, client.ForceOwnership, client.FieldOwner("kubeovn-operator"))
}

// filterObject returns the configuration object if object is owned by the configuratino controller
func (r *ConfigurationReconciler) filterObject(ctx context.Context, obj client.Object) []ctrl.Request {
	ownerRefs := obj.GetOwnerReferences()
	result := []ctrl.Request{}
	if len(ownerRefs) == 0 {
		return result
	}

	for _, v := range ownerRefs {
		if v.Kind == kubeovniov1.Kind && v.APIVersion == kubeovniov1.APIVersion {
			result = append(result, ctrl.Request{NamespacedName: types.NamespacedName{Name: v.Name, Namespace: r.Namespace}})
		}
	}
	return result
}

// AddWatches adds watches for all objects types being managed by the controller to ensure any changes
// to managed objects results in reconcile of configuration object
func (r *ConfigurationReconciler) AddWatches(b *builder.Builder) *builder.Builder {

	updatePred := predicate.Funcs{
		// ignore changes to managed fields
		UpdateFunc: func(e event.UpdateEvent) bool {
			e.ObjectOld.SetManagedFields([]metav1.ManagedFieldsEntry{})
			e.ObjectNew.SetManagedFields([]metav1.ManagedFieldsEntry{})
			return !reflect.DeepEqual(e.ObjectOld, e.ObjectNew)
		},

		// Allow create events
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},

		// Allow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		// Allow generic events (e.g., external triggers)
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
	for key := range templates.OrderedObjectList {
		b.Watches(key, handler.EnqueueRequestsFromMapFunc(r.filterObject), builder.WithPredicates(updatePred))
	}
	return b
}

// findMasterNodes will find nodes matching the master label criteria in the configuration
func (r *ConfigurationReconciler) findMasterNodes(ctx context.Context, config *kubeovniov1.Configuration) error {

	set, err := labels.ConvertSelectorToLabelsMap(config.Spec.MasterNodesLabel)
	if err != nil {
		return fmt.Errorf("error parsing label selector %s: %v", config.Spec.MasterNodesLabel, true)
	}
	nodeList := &corev1.NodeList{}

	err = r.List(ctx, nodeList, client.MatchingLabels(set))
	if err != nil {
		return fmt.Errorf("error listing nodes :%v", err)
	}

	var nodeAddresses []string
	for _, v := range nodeList.Items {
		address := nodeInternalIP(v)
		if len(address) > 0 {
			nodeAddresses = append(nodeAddresses, address)
		}
	}

	// if no nodeAddresses are found then it is likely we had no matching nodes
	// we need to pause reconcile of the object until label matches
	if len(nodeAddresses) == 0 && !config.ConditionTrue(kubeovniov1.WaitingForMatchignNodesCondition) {
		r.EventRecorder.Event(config, corev1.EventTypeWarning,
			"ReconcilePaused", "no nodes matching master node labels found")
		config.SetCondition(kubeovniov1.WaitingForMatchignNodesCondition, metav1.ConditionTrue, "Waiting for matching nodes", kubeovniov1.NodesNotFoundReason)
		return nil
	}

	if !addressArrayEqual(nodeAddresses, config.Status.MatchingNodeAddresses) {
		config.SetCondition(kubeovniov1.WaitingForMatchignNodesCondition, metav1.ConditionFalse, fmt.Sprintf("found nodes %s", strings.Join(nodeAddresses, ",")), kubeovniov1.NodesFoundReason)
		config.Status.MatchingNodeAddresses = nodeAddresses
	}
	return nil
}

// initializeConditions will initialise baseline conditions for the configuration object
func (r *ConfigurationReconciler) initializeConditions(ctx context.Context, config *kubeovniov1.Configuration) error {
	if !config.ConditionExists(kubeovniov1.WaitingForMatchignNodesCondition) {
		config.SetCondition(kubeovniov1.WaitingForMatchignNodesCondition, metav1.ConditionUnknown, "Unknown", kubeovniov1.ConditionUnknown)
	}

	if !config.ConditionExists(kubeovniov1.OVNNBLeaderFound) {
		config.SetCondition(kubeovniov1.OVNNBLeaderFound, metav1.ConditionUnknown, "Unknown", kubeovniov1.ConditionUnknown)
	}

	if !config.ConditionExists(kubeovniov1.OVNSBLeaderFound) {
		config.SetCondition(kubeovniov1.OVNSBLeaderFound, metav1.ConditionUnknown, "Unknown", kubeovniov1.ConditionUnknown)
	}

	if !config.ConditionExists(kubeovniov1.OVNNBDBHealth) {
		config.SetCondition(kubeovniov1.OVNNBDBHealth, metav1.ConditionUnknown, "Unknown", kubeovniov1.ConditionUnknown)
	}

	if !config.ConditionExists(kubeovniov1.OVNSBDBHealth) {
		config.SetCondition(kubeovniov1.OVNSBDBHealth, metav1.ConditionUnknown, "Unknown", kubeovniov1.ConditionUnknown)
	}
	return nil
}

// reconcileClusterScopedReference creates a dummy namespace and sets that as owner for cluster scoped objects
// this is deleted when the Configuration is deleted to ensure that the cluster scoped objects managed the configuration
// are GC'd
func (r *ConfigurationReconciler) reconcileClusterScopedReference(ctx context.Context, config *kubeovniov1.Configuration) error {
	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: kubeovniov1.KubeOVNFakeNamespace}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			newNS := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: kubeovniov1.KubeOVNFakeNamespace,
				},
			}
			return r.Client.Create(ctx, newNS)
		}
		return err
	}
	return nil
}

// deleteClusterScopedReference is triggered during configuration deletion
// and triggers deletion of cluster scoped objects
func (r *ConfigurationReconciler) deleteClusterScopedReference(ctx context.Context, config *kubeovniov1.Configuration) error {
	// enusre CRD objects are deleted first, this ensures that they are GC'd by the controllers
	// as some of them may have finalizers which may need some work to be done by the ovn controllers
	crdObjs, err := render.GenerateObjects(templates.CRDList, config, &apiextensionsv1.CustomResourceDefinition{}, r.RestConfig, r.Version)
	if err != nil {
		return fmt.Errorf("error rendering CRDs during configuration cleanup: %v", err)
	}

	if err := r.ensureCRDObjectCleanup(ctx, crdObjs); err != nil {
		return fmt.Errorf("error cleaning objects: %v", err)
	}

	// ensure crdObjs are cleaned up
	configObj := config.DeepCopy()
	newNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeovniov1.KubeOVNFakeNamespace,
		},
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: kubeovniov1.KubeOVNFakeNamespace}, newNS)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		r.Log.WithValues("name", config.Name).Info("namespace does not exist", "namespace", kubeovniov1.KubeOVNFakeNamespace)
	} else {
		if err := r.Client.Delete(ctx, newNS); err != nil {
			return err
		}
	}

	if controllerutil.ContainsFinalizer(config, kubeovniov1.KubeOVNConfigurationFinalizer) {
		controllerutil.RemoveFinalizer(config, kubeovniov1.KubeOVNConfigurationFinalizer)
		return r.Client.Patch(ctx, config, client.MergeFrom(configObj))
	}

	// remove node finalizers to ensure nodes can be managed later on
	nodeList := corev1.NodeList{}
	err = r.Client.List(ctx, &nodeList)
	if err != nil {
		return fmt.Errorf("error fetching node list during configuration deletion: %v", err)
	}

	for _, v := range nodeList.Items {
		node := &v
		nodeCopy := node.DeepCopy()
		if controllerutil.ContainsFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer) {
			controllerutil.RemoveFinalizer(node, kubeovniov1.KubeOVNNodeFinalizer)
			if err := r.Client.Patch(ctx, node, client.MergeFrom(nodeCopy)); err != nil {
				return fmt.Errorf("error patching node %s: %v", node.GetName(), err)
			}
		}
	}
	return nil
}

func nodeInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

func addressArrayEqual(existing []string, discovered []string) bool {
	slices.Sort(existing)
	slices.Sort(discovered)
	return slices.Equal(existing, discovered)
}

func (r *ConfigurationReconciler) ensureCRDObjectCleanup(ctx context.Context, objs []client.Object) error {
	dynClient, err := dynamic.NewForConfig(r.RestConfig)
	if err != nil {
		return fmt.Errorf("error generating dynamic client: %v", err)
	}

	var objectCount int
	for _, obj := range objs {
		crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
		if !ok {
			r.Log.WithValues("name", obj.GetName()).Error(fmt.Errorf("unable to assert obj to custom resource definition"), obj.GetName())
			continue
		}

		// dynamic client needs plural versions and we will check all possible versions for the crd
		for _, version := range crd.Spec.Versions {
			gvr := schema.GroupVersionResource{Group: crd.Spec.Group, Version: version.Name, Resource: crd.Spec.Names.Plural}
			r.Log.WithValues("GroupVersionResource", gvr).Info("looking up objects for gvr")
			var resourceInterface dynamic.ResourceInterface
			if crd.Spec.Scope == apiextensionsv1.NamespaceScoped {
				resourceInterface = dynClient.Resource(gvr).Namespace("")
			} else {
				resourceInterface = dynClient.Resource(gvr)
			}
			dynObjects, err := resourceInterface.List(ctx, metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("error checking getting list of objects for type %s: %v", obj.GetName(), err)
			}

			objectCount = objectCount + len(dynObjects.Items)
			r.Log.WithValues("objectType", obj.GetName()).Info("deleting objects as pre-requisite for configuration cleanup")
			for _, v := range dynObjects.Items {
				if !v.GetDeletionTimestamp().IsZero() {
					continue
				}
				if err := resourceInterface.Delete(ctx, v.GetName(), metav1.DeleteOptions{}); err != nil {
					return fmt.Errorf("error deleting resource %s: %v", v.GetName(), err)
				}
			}
		}

	}

	// we requeue if object count is not 0, this will ensure we delete objects and requeue forcing objects to be checked again
	// this gives controllers time to cleanup objects before CRD definitions are deleted
	if objectCount > 0 {
		return fmt.Errorf("%d objects still exist in the cluster, waiting for them to be GC'd by the controller", objectCount)
	}
	return nil
}
