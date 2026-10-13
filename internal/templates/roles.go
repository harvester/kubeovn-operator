package templates

var (
	system_ovn_ipsec_role = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-reader-ovn-ipsec
  namespace: {{ .Values.namespace }}
rules:
- apiGroups:
    - ""
  resources:
    - "secrets"
  resourceNames:
    - "ovn-ipsec-ca"
  verbs:
    - "get"
    - "list"
    - "watch"`

	system_ovn_central_role = `
{{- if index .Values "ovnCentral" "hcp" "enabled" }}
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ovn-central
  namespace: {{ include "kubeovn.centralNamespace" . }}
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
      - patch
  - apiGroups:
      - ""
    resources:
      - services
    verbs:
      - get
  - apiGroups:
      - discovery.k8s.io
    resources:
      - endpointslices
    verbs:
      - get
      - list
      - watch
{{- end }}`
	RoleList = []string{system_ovn_ipsec_role, system_ovn_central_role}
)
