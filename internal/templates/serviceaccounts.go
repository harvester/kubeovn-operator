package templates

var (
	ovn_sa_template = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn
  namespace: {{ .Values.namespace }}
automountServiceAccountToken: false
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
- name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}
`

	ovn_ovs_template = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn-ovs
  namespace: {{ .Values.namespace }}
automountServiceAccountToken: false
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
- name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}`

	kube_ovn_cni = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-ovn-cni
  namespace: {{ .Values.namespace }}
automountServiceAccountToken: false
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
- name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}`

	kube_ovn_app_template = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-ovn-app
  namespace: {{ .Values.namespace }}
automountServiceAccountToken: false
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
- name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}`

	kube_ovn_central_template = `
{{- if index .Values "ovnCentral" "hcp" "enabled" }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn-central
  namespace: {{ include "kubeovn.centralNamespace" . }}
automountServiceAccountToken: false
{{- if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
  - name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
`

	kube_ovn_controller_template = `
{{- if include "kubeovn.renderDataPlane" . }}
apiVersion: v1
kind: ServiceAccount
metadata:
  # Dedicated identity for kube-ovn-controller, distinct from the "ovn" SA shared by the
  # webhook, ic-controller and speaker, so the validating webhook can recognize requests
  # that only the controller is trusted to make (the switch_lb_rule vip mac repair in
  # pkg/webhook/vip.go and the share DNAT accounting records of the nftable LB service
  # feature). The name is the one the webhook derives its identity from, see
  # util.KubeOvnControllerServiceAccountName.
  name: kube-ovn-controller
  namespace: {{ .Values.namespace }}
automountServiceAccountToken: true
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
  - name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
`
	ServiceAccountList = []string{ovn_sa_template, ovn_ovs_template, kube_ovn_cni, kube_ovn_app_template, kube_ovn_central_template, kube_ovn_controller_template}
)
