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

	system_controller_sec_role = `
{{- if include "kubeovn.renderDataPlane" . }}
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kube-ovn-controller-secrets
  namespace: {{ .Values.namespace }}
rules:
  - apiGroups: [""]
    resources: [secrets]
    verbs: [create]
  - apiGroups: [""]
    resources: [secrets]
    resourceNames: [kube-ovn-tls, ovn-ipsec-ca, ovn-ipsec-signer]
    verbs: [get, update]
{{- if .Values.ipsec.certManager.enabled }}
  - apiGroups: [cert-manager.io]
    resources: [certificaterequests]
    verbs: [get, list, watch, create]
{{- end }}
{{- end }}`

	RoleList = []string{system_ovn_ipsec_role, system_ovn_central_role, system_controller_sec_role}
)
