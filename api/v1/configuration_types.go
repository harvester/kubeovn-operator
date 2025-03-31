/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfigurationSpec defines the desired state of Configuration.
type ConfigurationSpec struct {
	Global GlobalSpec `json:"global,omitempty"`
	// +kubebuilder:default:="IfNotPresent"
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
	// +kubebuilder:default:="kube-ovn/role=master"
	MasterNodesLabel string `json:"masterNodesLabel,omitempty"`
	// +kubebuilder:default:={}
	Networking NetworkingSpec `json:"networking,omitempty"`
	// +kubebuilder:default:={}
	Component ComponentSpec `json:"components,omitempty"`
	// +kubebuilder:default:={podCIDR:"10.16.0.0/16",podGateway:"10.16.0.1", serviceCIDR:"10.96.0.0/12",joinCIDR:"100.64.0.0/16",pingerExternalAddress:"1.1.1.1",pingerExternalDomain:"google.com."}
	IPv4 NetworkStackSpec `json:"ipv4,omitempty"`
	// +kubebuilder:default:={podCIDR:"fd00:10:16::/112",podGateway:"fd00:10:16::1", serviceCIDR:"fd00:10:96::/112",joinCIDR:"fd00:100:64::/112",pingerExternalAddress:"2606:4700:4700::1111",pingerExternalDomain:"google.com."}
	IPv6 NetworkStackSpec `json:"ipv6,omitempty"`
	// +kubebuilder:default:={podCIDR:"10.16.0.0/16,fd00:10:16::/112",podGateway:"10.16.0.1,fd00:10:16::1", serviceCIDR:"10.96.0.0/12,fd00:10:96::/112",joinCIDR:"100.64.0.0/16,fd00:100:64::/112",pingerExternalAddress:"1.1.1.1,2606:4700:4700::1111",pingerExternalDomain:"google.com."}
	DualStack NetworkStackSpec `json:"dualStack,omitempty"`
	// +kubebuilder:default:={}
	Performance PerformanceSpec `json:"performance,omitempty"`
	// +kubebuilder:default:={}
	Debug DebugSpec `json:"debug,omitempty"`
	// +kubebuilder:default:={}
	CNIConf CNIConfSpec `json:"cniConf,omitempty"`
	// +kubebuilder:default:={}
	KubeletConfig KubeletConfigSpec `json:"kubeletConfig,omitempty"`
	// +kubebuilder:default:={}
	LogConfig LogConfigSpec `json:"logConfig,omitempty"`
	// +kubebuilder:default:="/etc/origin/openvswitch"
	OpenVSwitchDir string `json:"openVSwitchDir,omitempty"`
	// +kubebuilder:default:="/etc/origin/ovn"
	OVNDir                   string `json:"ovnDir,omitempty"`
	DisableModulesManagement *bool  `json:"disableModulesManagement,omitempty"`
	HybridDPDK               *bool  `json:"hybridDPDK,omitempty"`
	// +kubebuilder:default:="hugepages-2Mi"
	HugepageSizeType string `json:"hugepageSizeType,omitempty"`
	// +kubebuilder:default:="1Gi"
	HugePages resource.Quantity `json:"hugePages,omitempty"`
	DPDK      *bool             `json:"dpdk,omitempty"`
	// +kubebuilder:default:="19.11"
	DPDKVersion string `json:"dpdkVersion,omitempty"`
	// +kubebuilder:default:="1000m"
	DPDKCPU resource.Quantity `json:"dpdkCPU,omitempty"`
	// +kubebuilder:default:="2Gi"
	DPDKMemory resource.Quantity `json:"dpdkMEMORY,omitempty"`
	// +kubebuilder:default:={requests:{},limits:{}}
	OVNCentral ResourceSpec `json:"ovnCentral,omitempty"`
	// +kubebuilder:default:={requests:{cpu:"200m",memory:"200Mi"},limits:{cpu:"2", memory:"1000Mi"}}
	OVSOVN ResourceSpec `json:"ovsOVN,omitempty"`
	// +kubebuilder:default:={requests:{cpu:"200m",memory:"200Mi"},limits:{cpu:"1",memory:"1000Mi"}}
	KubeOVNController ResourceSpec `json:"kubeOvnController,omitempty"`
	// +kubebuilder:default:={requests:{cpu:"100m",memory:"100Mi"},limits:{cpu:"1",memory:"1000Mi"}}
	KubeOVNCNI ResourceSpec `json:"kubeOvnCNI,omitempty"`
	// +kubebuilder:default:={requests:{cpu:"100m",memory:"100Mi"},limits:{cpu:"200m",memory:"400Mi"}}
	KubeOVNPinger ResourceSpec `json:"kubeOvnPinger,omitempty"`
	// +kubebuilder:default:={requests:{cpu:"200m",memory:"200Mi"},limits:{cpu:"200m",memory:"200Mi"}}
	KubeOVNMonitor ResourceSpec `json:"kubeOvnMonitor,omitempty"`
}

type GlobalSpec struct {
	Registry RegistrySpec `json:"registry,omitempty"`
	Images   ImageDetails `json:"images,omitempty"`
}

type RegistrySpec struct {
	// +kubebuilder:default:="docker.io/kubeovn"
	Address          string   `json:"address,omitempty"`
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

type ImageDetails struct {
	KubeOVNImage KubeOVNImageSpec `json:"kubeovn,omitempty"`
}

type KubeOVNImageSpec struct {
	// +kubebuilder:default:="kube-ovn"
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"` // defaults to version passed from command line
	// +kubebuilder:default:="kube-ovn-dpdk"
	DpdkRepository string `json:"dpdkRepository,omitempty"`
	// +kubebuilder:default:="vpc-nat-gateway"
	VpcRepository string `json:"vpcRepository,omitempty"`
	// +kubebuilder:default:=true
	SupportArm *bool `json:"supportArm,omitempty"`
	// +kubebuilder:default:=true
	ThirdParty *bool `json:"thirdParty,omitempty"`
}

type NetworkingSpec struct {
	// +kubebuilder:default:="ipv4"
	// +kubebuilder:validation:Enum=ipv4;ipv6;dual_stack
	NetStack string `json:"netStack,omitempty"`
	// +kubebuilder:default:=false
	EnableSSL *bool `json:"enableSSL,omitempty"`
	// +kubebuilder:default:="geneve"
	// +kubebuilder:validation:Enum=geneve;vlan
	NetworkType string `json:"networkType,omitempty"`
	// +kubebuilder:default:="geneve"
	// +kubebuilder:validation:Enum=geneve;vxlan;stt
	TunnelType          string `json:"tunnelType,omitempty"`
	Interface           string `json:"interface,omitempty"`
	DpdkTunnelInterface string `json:"dpdkTunnelInterface,omitempty"`
	ExcludeIPS          string `json:"excludeIPS,omitempty"`
	// +kubebuilder:default:="veth-pair"
	PodNicType string `json:"podNicType,omitempty"`
	// +kubebuilder:default:={}
	Vlan VlanSpec `json:"vlan,omitempty"`
	// +kubebuilder:default:=false
	ExchangeLinkName *bool `json:"exchangeLinkName,omitempty"`
	// +kubebuilder:default:=true
	EnableEIPSNAT *bool `json:"enableEIPSNAT,omitempty"`
	// +kubebuilder:default:="ovn-default"
	DefaultSubnet string `json:"defaultSubnet,omitempty"`
	// +kubebuilder:default:="ovn-cluster"
	DefaultVPC string `json:"defaultVPC,omitempty"`
	// +kubebuilder:default:="join"
	NodeSubnet string `json:"nodeSubnet,omitempty"`
	// +kubebuilder:default:=false
	EnableECMP *bool `json:"enableECMP,omitempty"`
	// +kubebuilder:default:=true
	EnableMetrics   *bool  `json:"enableMetrics,omitempty"`
	NodeLocalDNSIPS string `json:"nodeLocalDNSIPS,omitempty"`
	// +kubebuilder:default:=180000
	// +kubebuilder:validation:Minimum=1
	ProbeInterval int `json:"probeInterval,omitempty"`
	// +kubebuilder:default:=5000
	// +kubebuilder:validation:Minimum=1
	OvnNorthdProbeInterval int `json:"ovnNorthdProbeInterval,omitempty"`
	// +kubebuilder:default:=5
	// +kubebuilder:validation:Minimum=1
	OvnLeaderProbeInterval int `json:"ovnLeaderProbeInterval,omitempty"`
	// +kubebuilder:default:=10000
	// +kubebuilder:validation:Minimum=1
	OvnRemoteProbeInterval int `json:"ovnRemoteProbeInterval,omitempty"`
	// +kubebuilder:default:=10
	// +kubebuilder:validation:Minimum=1
	OvnRemoteOpenflowInterval int `json:"ovnRemoteOpenflowInterval,omitempty"`
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum=1
	OvnNorthdNThreads int `json:"ovnNorthdNThreads,omitempty"`
	// +kubebuilder:default:=false
	EnableCompact *bool `json:"enableCompact,omitempty"`
}

type VlanSpec struct {
	// +kubebuilder:default:="provider"
	ProviderName  string `json:"providerName,omitempty"`
	VlanInterface string `json:"vlanInterface,omitempty"`
	// +kubebuilder:default:="ovn-vlan"
	VlanName string `json:"vlanName,omitempty"`
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4094
	VlanID int `json:"vlanId,omitempty"`
}

type ComponentSpec struct {
	// +kubebuilder:default:=true
	EnableLB *bool `json:"enableLB,omitempty"`
	// +kubebuilder:default:=true
	EnableNP *bool `json:"enableNP,omitempty"`
	// +kubebuilder:default:=true
	EnableExternalVPC *bool `json:"enableExternalVPC,omitempty"`
	// +kubebuilder:default:=false
	HardwareOffload *bool `json:"hardwareOffload,omitempty"`
	// +kubebuilder:default:=false
	EnableLBSVC *bool `json:"enableLBSVC,omitempty"`
	// +kubebuilder:default:=true
	EnableKeepVMIP *bool `json:"enableKeepVMIP,omitempty"`
	// +kubebuilder:default:=true
	LsDnatModDlDst *bool `json:"lsDnatModDlDst,omitempty"`
	// +kubebuilder:default:=true
	LsCtSkipOstLportIPS *bool `json:"lsCtSkipOstLportIPS,omitempty"`
	// +kubebuilder:default:=true
	CheckGateway *bool `json:"checkGateway,omitempty"`
	// +kubebuilder:default:=false
	LogicalGateway *bool `json:"logicalGateway,omitempty"`
	// +kubebuilder:default:=true
	EnableBindLocalIP *bool `json:"enableBindLocalIP,omitempty"`
	// +kubebuilder:default:=false
	SecureServing **bool `json:"secureServing,omitempty"`
	// +kubebuilder:default:=false
	U2OInterconnection *bool `json:"u2oInterconnection,omitempty"`
	// +kubebuilder:default:=false
	EnableTProxy *bool `json:"enableTProxy,omitempty"`
	// +kubebuilder:default:=false
	EnableIC *bool `json:"enableIC,omitempty"`
	// +kubebuilder:default:=true
	EnableNATGateway *bool `json:"enableNATGateway,omitempty"`
	// +kubebuilder:default:=false
	EnableOVNIPSec *bool `json:"enableOVNIPSec,omitempty"`
	// +kubebuilder:default:=false
	EnableANP *bool `json:"enableANP,omitempty"`
	// +kubebuilder:default:=false
	SetVLANTxOff *bool `json:"setVLANTxOff,omitempty"`
	// +kubebuilder:default:=3
	// +kubebuilder:validation:Minimum=1
	OVSDBConTimeout int `json:"OVSDBConTimeout,omitempty"`
	// +kubebuilder:default:=10
	// +kubebuilder:validation:Minimum=1
	OVSDBInactivityTimeout int `json:"OVSDBInactivityTimeout,omitempty"`
	// +kubebuilder:default:=true
	EnableLiveMigrationOptimize *bool `json:"enableLiveMigrationOptimize,omitempty"`
}

type NetworkStackSpec struct {
	// +kubebuilder:default:="fd00:10:16::/112"
	PodCIDR string `json:"podCIDR,omitempty"`
	// +kubebuilder:default:="fd00:10:16::1"
	PodGateway string `json:"podGateway,omitempty"`
	// +kubebuilder:default:="fd00:10:96::/112"
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
	// +kubebuilder:default:="fd00:100:64::/112"
	JoinCIDR string `json:"joinCIDR,omitempty"`
	// +kubebuilder:default:="2606:4700:4700::1111"
	PingerExternalAddress string `json:"pingerExternalAddress,omitempty"`
	// +kubebuilder:default:="google.com."
	PingerExternalDomain string `json:"pingerExternalDomain,omitempty"`
}

type PerformanceSpec struct {
	// +kubebuilder:default:=360
	// +kubebuilder:validation:Minimum=0
	GCInterval int `json:"gcInterval,omitempty"`
	// +kubebuilder:default:=20
	// +kubebuilder:validation:Minimum=1
	InspectInterval int `json:"inspectInterval,omitempty"`
	// +kubebuilder:default:=100
	// +kubebuilder:validation:Minimum=0
	OVSVSCtlConcurrency int `json:"ovsVSCtlConcurrency,omitempty"`
}

type DebugSpec struct {
	EnableMirror *bool `json:"enableMirror,omitempty"`
	// +kubebuilder:default:="mirror0"
	MirrorInterface string `json:"mirrorInterface,omitempty"`
}

type CNIConfSpec struct {
	// +kubebuilder:default:="01"
	CNIConfigPriority string `json:"cniConfigPriority,omitempty"`
	// +kubebuilder:default:="/etc/cni/net.d"
	CNIConfigDir string `json:"cniConfigDir,omitempty"`
	// +kubebuilder:default:="/opt/cni/bin"
	CNIBinDir string `json:"cniBinDir,omitempty"`
	// +kubebuilder:default:="/kube-ovn/01-kube-ovn.conflist"
	CNIConfFile string `json:"cniConfFile,omitempty"`
	// +kubebuilder:default:="/usr/local/bin"
	LocalBinDir      string `json:"localBinDir,omitempty"`
	MountLocalBinDir *bool  `json:"mountLocalBinDir,omitempty"`
}
type KubeletConfigSpec struct {
	// +kubebuilder:default:="/var/lib/kubelet"
	KubeletDir string `json:"kubeletDir,omitempty"`
}

type LogConfigSpec struct {
	// +kubebuilder:default:="/var/log"
	LogDir string `json:"logDir,omitempty"`
}

type ResourceSpec struct {
	// +kubebuilder:default:={cpu:"200m",memory:"200Mi"}
	Requests CPUMemSpec `json:"requests,omitempty"`
	// +kubebuilder:default:={cpu:2, memory:"1000Mi"}
	Limits CPUMemSpec `json:"limits,omitempty"`
}

type CPUMemSpec struct {
	CPU    resource.Quantity `json:"cpu,omitempty"`
	Memory resource.Quantity `json:"memory,omitempty"`
}

// ConfigurationStatus defines the observed state of Configuration.
type ConfigurationStatus struct {
	MatchingNodeAddresses []string           `json:"matchingNodeAddresses,omitempty"`
	Conditions            []metav1.Condition `json:"conditions,omitempty"`
	Status                string             `json:"status,omitempty"`
	ManagedObjects        []ObjectReference  `json:"managedObjects,omitempty"`
}

type ObjectReference struct {
	GVK  GroupVersionKind `json:"gvk,omitempty"`
	Name string           `json:"name,omitempty"`
}

type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Configuration is the Schema for the configurations API.
type Configuration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationSpec   `json:"spec,omitempty"`
	Status ConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfigurationList contains a list of Configuration.
type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Configuration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Configuration{}, &ConfigurationList{})
}

const (
	// ConfigurationStatusDeploying indicates that changes are being deployed to the stack
	ConfigurationStatusDeploying = "Deploying"
	// ConfigurationStatusDeployed indicates that changes have been deployed
	ConfigurationStatusDeployed      = "Deployed"
	DefaultConfigurationName         = "kubeovn"
	WaitingForMatchignNodesCondition = "waitingForMatchignNodes"
	OVNNBLeaderFound                 = "ovnNorthDBLeaderFound"
	OVNSBLeaderFound                 = "ovnSouthDBLeaderFound"
	OVNNBDBHealth                    = "ovnNBDBHealth"
	OVNSBDBHealth                    = "ovnSBDBHealth"
	KubeOVNFakeNamespace             = "kubeovn-owner-namespace"
	KubeOVNConfigurationFinalizer    = "finalizer.kubeovn.io/configuration"
	KubeOVNNodeFinalizer             = "finalizer.kubeovn.io/node"
	SBLeaderLabel                    = "ovn-sb-leader=true"
	NBLeaderLabel                    = "ovn-nb-leader=true"
	OVNCentralContainerName          = "ovn-central"
	NBCheckScript                    = `ovs-appctl -t /var/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound`
	SBCheckScript                    = `ovs-appctl -t /var/run/ovn/ovnsb_db.ctl cluster/status OVN_Southbound`
	NodesFoundReason                 = "NodesFound"
	NodesNotFoundReason              = "NodesNotFound"
	ConditionUnknown                 = "ConditionUnknown"
	ConditionCheckFailed             = "ConditionCheckFailed"
	ConditionCheckExecuted           = "ConditionCheckExecuted"
	LeaderFound                      = "LeaderFound"
	LeaderNotFound                   = "LeaderNotFound"
	DBHealth                         = "DBHealth"
)

var (
	APIVersion = fmt.Sprintf("%s/%s", GroupVersion.Group, GroupVersion.Version)
	Kind       = "Configuration"
)

// SetCondition updates or adds a new condition to the object
func (c *Configuration) SetCondition(conditionType string, conditionStatus metav1.ConditionStatus, message string, reason string) {
	for i := range c.Status.Conditions {
		if c.Status.Conditions[i].Type == conditionType {
			c.Status.Conditions[i].Status = conditionStatus
			c.Status.Conditions[i].Message = message
			c.Status.Conditions[i].Reason = reason
			c.Status.Conditions[i].LastTransitionTime = metav1.Now()
			c.Status.Conditions[i].ObservedGeneration = c.ObjectMeta.Generation
			return
		}
	}
	c.Status.Conditions = append(c.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
		ObservedGeneration: c.ObjectMeta.Generation,
	})
}

func (c *Configuration) ConditionTrue(conditionType string) bool {
	condition := c.LookupCondition(conditionType)
	return condition.Status == metav1.ConditionTrue
}

func (c *Configuration) ConditionFalse(conditionType string) bool {
	condition := c.LookupCondition(conditionType)
	return condition.Status == metav1.ConditionFalse
}

func (c *Configuration) ConditionUnknown(conditionType string) bool {
	condition := c.LookupCondition(conditionType)
	return condition.Status == metav1.ConditionUnknown
}

func (c *Configuration) LookupCondition(conditionType string) metav1.Condition {
	for _, v := range c.Status.Conditions {
		if v.Type == conditionType {
			return v
		}
	}
	return metav1.Condition{}
}

func (c *Configuration) ConditionExists(conditionType string) bool {
	for _, v := range c.Status.Conditions {
		if v.Type == conditionType {
			return true
		}
	}
	return false
}
