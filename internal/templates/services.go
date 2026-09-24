package templates

var (
	kube_ovn_controller_service = `
kind: Service
apiVersion: v1
metadata:
  name: kube-ovn-controller
  namespace: {{ .Values.namespace }}
  labels:
    app: kube-ovn-controller
spec:
  selector:
    app: kube-ovn-controller
  ports:
    - port: 10660
      name: metrics
  {{- if eq .Values.networking.netStack "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}`

	kube_ovn_monitor_service = `kind: Service
apiVersion: v1
metadata:
  name: kube-ovn-monitor
  namespace: {{ .Values.namespace }}
  labels:
    app: kube-ovn-monitor
spec:
  ports:
    - name: metrics
      port: 10661
  type: ClusterIP
  selector:
    app: kube-ovn-monitor
  sessionAffinity: None
  {{- if eq .Values.networking.netStack "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}`

	ovn_nb_service = `{{- $svc := index .Values "ovnCentral" "service" }}
{{- $hcp := index .Values "ovnCentral" "hcp" }}
{{- $hcpSvc := index .Values "ovnCentral" "hcp" "service" }}
kind: Service
apiVersion: v1
metadata:
  name: ovn-nb
  namespace: {{ include "kubeovn.centralNamespace" . }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  annotations:
    # MetalLB allows multiple Services to share a single VIP only when they
    # carry the same allow-shared-ip annotation. The three ovn-* Services all
    # opt into the same group so a Kamaji-style data plane can dial nb / sb /
    # northd on one VIP, distinguished only by port. Harmless on non-MetalLB
    # load balancers; remove the annotation if your provider rejects it.
    metallb.universe.tf/allow-shared-ip: kube-ovn-central
  {{- end }}
spec:
  ports:
    - name: ovn-nb
      protocol: TCP
      port: 6641
      targetPort: 6641
      {{- if and $hcp.enabled (or (eq $hcpSvc.type "NodePort") (eq $hcpSvc.type "LoadBalancer")) }}
      nodePort: {{ $hcpSvc.nbNodePort }}
      {{- end }}
  type: {{ ternary ($hcpSvc.type | default "ClusterIP") ($svc.type | default "ClusterIP") $hcp.enabled }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  loadBalancerIP: {{ $svc.loadBalancerIP | quote }}
  {{- end }}
  {{- if and (not $hcp.enabled) (or (eq $svc.type "LoadBalancer") (eq $svc.type "NodePort")) $svc.externalTrafficPolicy }}
  externalTrafficPolicy: {{ $svc.externalTrafficPolicy }}
  {{- end }}
  {{- if eq .Values.networking.NET_STACK "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}
  selector:
    app: ovn-central
    {{- if or $hcp.enabled (ne .Values.ovnCentralMode "single") }}
    ovn-nb-leader: "true"
    {{- end }}
  sessionAffinity: None`

	ovn_northd_service = `
{{- $svc := index .Values "ovnCentral" "service" }}
{{- $hcp := index .Values "ovnCentral" "hcp" }}
kind: Service
apiVersion: v1
metadata:
  name: ovn-northd
  namespace: {{ include "kubeovn.centralNamespace" . }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  annotations:
    metallb.universe.tf/allow-shared-ip: kube-ovn-central
  {{- end }}
spec:
  ports:
    - name: ovn-northd
      protocol: TCP
      port: 6643
      targetPort: 6643
  type: {{ ternary "ClusterIP" ($svc.type | default "ClusterIP") $hcp.enabled }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  loadBalancerIP: {{ $svc.loadBalancerIP | quote }}
  {{- end }}
  {{- if and (not $hcp.enabled) (or (eq $svc.type "LoadBalancer") (eq $svc.type "NodePort")) $svc.externalTrafficPolicy }}
  externalTrafficPolicy: {{ $svc.externalTrafficPolicy }}
  {{- end }}
  {{- if eq .Values.networking.NET_STACK "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}
  selector:
    app: ovn-central
    {{- if or $hcp.enabled (ne .Values.ovnCentralMode "single") }}
    ovn-northd-leader: "true"
    {{- end }}
  sessionAffinity: None`

	kube_ovn_cni_service = `kind: Service
apiVersion: v1
metadata:
  name: kube-ovn-cni
  namespace: {{ .Values.namespace }}
  labels:
    app: kube-ovn-cni
spec:
  selector:
    app: kube-ovn-cni
  ports:
    - port: 10665
      name: metrics
  {{- if eq .Values.networking.NET_STACK "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}`

	kube_ovn_pinger_service = `kind: Service
apiVersion: v1
metadata:
  name: kube-ovn-pinger
  namespace: {{ .Values.namespace }}
  labels:
    app: kube-ovn-pinger
spec:
  selector:
    app: kube-ovn-pinger
  ports:
    - port: 8080
      name: metrics
  {{- if eq .Values.networking.NET_STACK "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}`

	ovn_sb_service = `
{{- $svc := index .Values "ovnCentral" "service" }}
{{- $hcp := index .Values "ovnCentral" "hcp" }}
{{- $hcpSvc := index .Values "ovnCentral" "hcp" "service" }}
kind: Service
apiVersion: v1
metadata:
  name: ovn-sb
  namespace: {{ include "kubeovn.centralNamespace" . }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  annotations:
    metallb.universe.tf/allow-shared-ip: kube-ovn-central
  {{- end }}
spec:
  ports:
    - name: ovn-sb
      protocol: TCP
      port: 6642
      targetPort: 6642
      {{- if and $hcp.enabled (or (eq $hcpSvc.type "NodePort") (eq $hcpSvc.type "LoadBalancer")) }}
      nodePort: {{ $hcpSvc.sbNodePort }}
      {{- end }}
  type: {{ ternary ($hcpSvc.type | default "ClusterIP") ($svc.type | default "ClusterIP") $hcp.enabled }}
  {{- if and (not $hcp.enabled) (eq $svc.type "LoadBalancer") $svc.loadBalancerIP }}
  loadBalancerIP: {{ $svc.loadBalancerIP | quote }}
  {{- end }}
  {{- if and (not $hcp.enabled) (or (eq $svc.type "LoadBalancer") (eq $svc.type "NodePort")) $svc.externalTrafficPolicy }}
  externalTrafficPolicy: {{ $svc.externalTrafficPolicy }}
  {{- end }}
  {{- if eq .Values.networking.NET_STACK "dual_stack" }}
  ipFamilyPolicy: PreferDualStack
  {{- end }}
  selector:
    app: ovn-central
    {{- if or $hcp.enabled (ne .Values.ovnCentralMode "single") }}
    ovn-sb-leader: "true"
    {{- end }}
  sessionAffinity: None`

	kubeovn_webhook_service = `kind: Service
apiVersion: v1
metadata:
  name: kube-ovn-webhook
  namespace: {{ .Values.namespace }}
spec:
  ports:
    - name: kube-ovn-webhook
      protocol: TCP
      port: 443
      targetPort: 8443
  type: ClusterIP
  selector:
    app: kube-ovn-webhook
  sessionAffinity: None
  `

	ovn_central_service = `
{{- if index .Values "ovnCentral" "hcp" "enabled" }}
kind: Service
apiVersion: v1
metadata:
  name: ovn-central
  namespace: {{ include "kubeovn.centralNamespace" . }}
spec:
  clusterIP: None
  publishNotReadyAddresses: true
  ports:
    - name: nb-raft
      protocol: TCP
      port: 6643
      targetPort: 6643
    - name: sb-raft
      protocol: TCP
      port: 6644
      targetPort: 6644
  selector:
    app: ovn-central
{{- end }}
`

	ServicesList = []string{kube_ovn_controller_service, kube_ovn_monitor_service, ovn_nb_service, ovn_northd_service, kube_ovn_cni_service, kube_ovn_pinger_service, ovn_sb_service, kubeovn_webhook_service, ovn_central_service}
)
