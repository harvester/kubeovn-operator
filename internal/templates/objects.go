package templates

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var OrderedObjectList = map[client.Object][]string{
	&apiextensionsv1.CustomResourceDefinition{}: CRDList,
	&corev1.Secret{}:             SecretList,
	&corev1.ServiceAccount{}:     ServiceAccountList,
	&rbacv1.RoleBinding{}:        RoleBindingList,
	&rbacv1.ClusterRole{}:        ClusterRoleList,
	&rbacv1.ClusterRoleBinding{}: ClusterRoleBindingList,
	&corev1.ConfigMap{}:          ConfigMapList,
	&appsv1.Deployment{}:         DeploymentList,
	&appsv1.DaemonSet{}:          DaemonsetList,
	&corev1.Service{}:            ServicesList,
}
