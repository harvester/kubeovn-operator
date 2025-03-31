package templates

var (
	system_ovn_clusterrolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ovn
roleRef:
  name: system:ovn
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: ovn
    namespace: {{ .Values.namespace }}`

	ovn_ovs_clusterrolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ovn-ovs
roleRef:
  name: system:ovn-ovs
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: ovn-ovs
    namespace: {{ .Values.namespace }}`

	system_kube_ovn_cni_clusterrolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-ovn-cni
roleRef:
  name: system:kube-ovn-cni
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: kube-ovn-cni
    namespace: {{ .Values.namespace }}`

	system_kube_ovn_app_clusterrolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-ovn-app
roleRef:
  name: system:kube-ovn-app
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: kube-ovn-app
    namespace: {{ .Values.namespace }}`

	ClusterRoleBindingList = []string{system_ovn_clusterrolebinding, ovn_ovs_clusterrolebinding, system_kube_ovn_cni_clusterrolebinding, system_kube_ovn_app_clusterrolebinding}
)
