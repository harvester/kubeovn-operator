package templates

var (
	kubeovn_validatingwebhook_configuration = `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: kube-ovn-webhook
webhooks:
- name: pod-ip-validating.kube-ovn.io
  rules:
    - operations:
        - CREATE
      apiGroups:
        - "apps"
      apiVersions:
        - v1
      resources:
        - deployments
        - statefulsets
        - daemonsets
    - operations:
        - CREATE
      apiGroups:
        - "batch"
      apiVersions:
        - v1
      resources:
        - jobs
        - cronjobs
    - operations:
        - CREATE
      apiGroups:
        - ""
      apiVersions:
        - v1
      resources:
        - pods
    - operations:
        - CREATE
        - UPDATE
        - DELETE
      apiGroups:
        - "kubeovn.io"
      apiVersions:
        - v1
      resources:
        - subnets
        - vpcs
        - vips
        - vpc-nat-gateways
        - iptables-eips
        - iptables-dnat-rules
        - iptables-snat-rules
        - iptables-fip-rules
  failurePolicy: Ignore
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  timeoutSeconds: 5
  clientConfig:
    caBundle: {{ .Values.caCert | b64enc}}
    service:
      namespace: kube-system
      name: kube-ovn-webhook
      path: /validating
      port: 443`

	ValidatingWebhookConfigurationList = []string{kubeovn_validatingwebhook_configuration}
)
