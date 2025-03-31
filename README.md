# kubeovn-operator
kubeovn-operator allows users to install and manage the lifecycle of kubeovn installation

## Description
kubeovn-operator allows users to install/upgrade and perform automated maintainance operations on their kubeovn installation

The operator introduces a new `configuration` CRD which allows users to define the desired state of their kubeovn installation.

Users can use the sample configuration available in `config/samples/kubeovn.io_v1_configuration.yaml` to get started.

```
apiVersion: kubeovn.io/v1
kind: Configuration
metadata:
  name: kubeovn
  namespace: kube-system
spec:
  cniConf:
    cniBinDir: /opt/cni/bin
    cniConfFile: /kube-ovn/01-kube-ovn.conflist
    cniConfigDir: /etc/cni/net.d
    cniConfigPriority: "90"
    localBinDir: /usr/local/bin
  components:
    OVSDBConTimeout: 3
    OVSDBInactivityTimeout: 10
    checkGateway: true
    enableANP: false
    enableBindLocalIP: true
    enableExternalVPC: true
    enableIC: false
    enableKeepVMIP: true
    enableLB: true
    enableLBSVC: false
    enableLiveMigrationOptimize: true
    enableNATGateway: true
    enableNP: true
    enableOVNIPSec: false
    enableTProxy: false
    hardwareOffload: false
    logicalGateway: false
    lsCtSkipOstLportIPS: true
    lsDnatModDlDst: true
    secureServing: false
    setVLANTxOff: false
    u2oInterconnection: false
  debug:
    mirrorInterface: mirror0
  dpdkCPU: "0"
  dpdkMEMORY: "0"
  dpdkVersion: "19.11"
  dualStack:
    joinCIDR: fd00:100:64::/112
    pingerExternalAddress: 2606:4700:4700::1111
    pingerExternalDomain: google.com.
    podCIDR: fd00:10:16::/112
    podGateway: fd00:10:16::1
    serviceCIDR: fd00:10:96::/112
  global:
    images:
      kubeovn:
        dpdkRepository: kube-ovn-dpdk
        repository: kube-ovn
        supportArm: true
        thirdParty: true
        vpcRepository: vpc-nat-gateway
    registry:
      address: docker.io/kubeovn
  hugePages: "0"
  hugepageSizeType: hugepages-2Mi
  imagePullPolicy: IfNotPresent
  ipv4:
    joinCIDR: 100.64.0.0/16
    pingerExternalAddress: 1.1.1.1
    pingerExternalDomain: google.com.
    podCIDR: 10.42.0.0/16
    podGateway: 10.42.0.1
    serviceCIDR: 10.43.0.0/16
  ipv6:
    joinCIDR: fd00:100:64::/112
    pingerExternalAddress: 2606:4700:4700::1111
    pingerExternalDomain: google.com.
    podCIDR: fd00:10:16::/112
    podGateway: fd00:10:16::1
    serviceCIDR: fd00:10:96::/112
  kubeOvnCNI:
    requests:
      cpu: "100m"
      memory: "100Mi"
    limits:
      cpu: "1"
      memory: "1Gi"
  kubeOvnController:
    requests:
      cpu: "200m"
      memory: "200Mi"
    limits:
      cpu: "1"
      memory: "1Gi"
  kubeOvnMonitor:
    requests:
      cpu: "200m"
      memory: "200Mi"
    limits:
      cpu: "200m"
      memory: "200Mi"
  kubeOvnPinger:
    requests:
      cpu: "100m"
      memory: "100Mi"
    limits:
      cpu: "200m"
      memory: "400Mi"
  kubeletConfig:
    kubeletDir: /var/lib/kubelet
  logConfig:
    logDir: /var/log
  masterNodesLabel: node-role.kubernetes.io/control-plane=true
  networking:
    defaultSubnet: ovn-default
    defaultVPC: ovn-cluster
    enableECMP: false
    enableEIPSNAT: true
    enableMetrics: true
    enableSSL: false
    netStack: ipv4
    networkType: geneve
    nodeSubnet: join
    ovnLeaderProbeInterval: 5
    ovnNorthdNThreads: 1
    ovnNorthdProbeInterval: 5000
    ovnRemoteOpenflowInterval: 10
    ovnRemoteProbeInterval: 10000
    podNicType: veth-pair
    probeInterval: 180000
    tunnelType: vxlan
    nodeLocalDNSIPS: ""
    vlan:
      providerName: provider
      vlanId: 1
      vlanName: ovn-vlan
  openVSwitchDir: /var/lib/rancher/origin/openvswitch
  ovnCentral:
    requests:
      cpu: "300m"
      memory: "200Mi"
    limits:
      cpu: "3"
      memory: "4Gi"
  ovnDir: /etc/origin/ovn
  ovsOVN:
    limits:
      cpu: "0"
      memory: "0"
    requests:
      cpu: "0"
      memory: "0"
  performance:
    gcInterval: 360
    inspectInterval: 20
    ovsVSCtlConcurrency: 100
```

The operator has 3 main reconcile loops:
* configuration controller: generates kubeovn objects from the [templates](./internal/templates/) and reconciles their eventual state against the generated state.
* healthcheck controller: reconciles ovn nb/sb databases from the cluster and updates the results in appropriate `configuration` status conditions
* node controller: watches node deleteion events and based on the role of node performs nb/sb member cleanup and chassis cleanup 

A sample status looks follows:

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/kubeovn-operator:dev
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/kubeovn-operator:dev
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/kubeovn-operator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/kubeovn-operator/<tag or branch>/dist/install.yaml
```


## License

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

