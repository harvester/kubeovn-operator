package templates

var (
	ovn_sa_template = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn
  namespace: {{ .Values.namespace }}
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
{{-  if .Values.global.registry.imagePullSecrets }}
imagePullSecrets:
{{- range $index, $secret := .Values.global.registry.imagePullSecrets }}
{{- if $secret }}
- name: {{ $secret | quote}}
{{- end }}
{{- end }}
{{- end }}`

	ServiceAccountList = []string{ovn_sa_template, ovn_ovs_template, kube_ovn_cni, kube_ovn_app_template}
)
