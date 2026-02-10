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

	RoleList = []string{system_ovn_ipsec_role}
)
