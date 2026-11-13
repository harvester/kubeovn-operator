package templates

var (
	ipsec_issuer_policy_validatingadmissionpolicies = `{{- if .Values.ipsec.certManager.enabled }}
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: kube-ovn-ipsec-issuer
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
      - apiGroups: [cert-manager.io]
        apiVersions: [v1]
        operations: [CREATE, UPDATE]
        resources: [certificaterequests]
  matchConditions:
    - name: dedicated-ipsec-issuer
      expression: >-
        object.spec.issuerRef.name == {{ .Values.ipsec.certManager.issuerName | toJson }} &&
        has(object.spec.issuerRef.kind) && object.spec.issuerRef.kind == "ClusterIssuer" &&
        (!has(object.spec.issuerRef.group) || object.spec.issuerRef.group == "cert-manager.io")
  validations:
    - expression: >-
        (request.operation == "UPDATE" && object.spec == oldObject.spec) ||
        request.userInfo.username == "system:serviceaccount:{{ .Values.namespace }}:kube-ovn-controller" &&
        object.metadata.namespace == {{ .Values.namespace | toJson }}
      message: The IPsec ClusterIssuer accepts only authorized kube-ovn-controller requests
{{- end }}
`
	ValidatingAdmissionPoliciesList = []string{ipsec_issuer_policy_validatingadmissionpolicies}
)
