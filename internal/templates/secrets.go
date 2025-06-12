package templates

var (
	kube_ovn_tls_secret = `{{- if .Values.networking.enableSSL }}
{{- $cn := "ovn" -}}
{{- $ca := genCA "ovn-ca" 3650 -}}
---
apiVersion: v1
kind: Secret
metadata:
  name: kube-ovn-tls
  namespace: {{ .Values.namespace }}
data:
{{- $existingSecret := lookup "v1" "Secret" .Values.namespace "kube-ovn-tls" }}
  {{- if $existingSecret }}
  cacert: {{ index $existingSecret.data "cacert" }}
  cert: {{ index $existingSecret.data "cert" }}
  key: {{ index $existingSecret.data "key" }}
  {{- else }}
  {{- with genSignedCert $cn nil nil 3650 $ca }}
  cacert: {{ b64enc $ca.Cert }}
  cert: {{ b64enc .Cert }}
  key: {{ b64enc .Key }}
  {{- end }}
  {{- end }}
{{- end }}`

	SecretList = []string{kube_ovn_tls_secret}
)
