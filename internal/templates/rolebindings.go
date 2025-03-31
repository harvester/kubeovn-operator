package templates

var (
	kube_ovn_app_rolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kube-ovn-app
  namespace: {{ .Values.namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
  - kind: ServiceAccount
    name: kube-ovn-app
    namespace: {{ .Values.namespace }}`

	kube_ovn_cni_rolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kube-ovn-cni
  namespace: {{ .Values.namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
  - kind: ServiceAccount
    name: kube-ovn-cni
    namespace: {{ .Values.namespace }}`

	ovn_rolebinding = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ovn
  namespace: {{ .Values.namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
  - kind: ServiceAccount
    name: ovn
    namespace: {{ .Values.namespace }}`

	RoleBindingList = []string{kube_ovn_cni_rolebinding, kube_ovn_app_rolebinding, ovn_rolebinding}
)
