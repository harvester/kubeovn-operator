package templates

var (
	ipsec_issuer_policy_validatingadmissionpolicybindings = `{{- if .Values.ipsec.certManager.enabled }}
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: kube-ovn-ipsec-issuer
spec:
  policyName: kube-ovn-ipsec-issuer
  validationActions: [Deny]
{{- end }}
`
	ValidatingAdmissionPolicyBindingsList = []string{ipsec_issuer_policy_validatingadmissionpolicybindings}
)
