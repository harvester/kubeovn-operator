package templates

var (
	ovn_vpc_nat_config = `kind: ConfigMap
apiVersion: v1
metadata:
  name: ovn-vpc-nat-config
  namespace: {{ .Values.namespace }}
annotations:
  kubernetes.io/description: "kube-ovn vpc-nat common config"
data:
  image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.vpcRepository }}:{{ .Values.global.images.kubeovn.tag }}`

	ovn_vpc_nat_gw_config = `kind: ConfigMap
apiVersion: v1
metadata:
  name: ovn-vpc-nat-gw-config
  namespace: kube-system
data:
  enable-vpc-nat-gw: "{{ .Values.components.enableNATGateway }}"`

	ConfigMapList = []string{ovn_vpc_nat_config, ovn_vpc_nat_gw_config}
)
