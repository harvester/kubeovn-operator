package templates

var (
	ovs_ovn_dpdk_daemonset = `{{- if .Values.hybridDPDK }}
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovs-ovn-dpdk
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      This daemon set launches the openvswitch daemon.
spec:
  selector:
    matchLabels:
      app: ovs-dpdk
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: ovs-dpdk
        component: network
        type: infra
    spec:
      tolerations:
      - operator: Exists
      priorityClassName: system-node-critical
      serviceAccountName: ovn-ovs
      hostNetwork: true
      hostPID: true
      containers:
        - name: openvswitch
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}-dpdk
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command: ["/kube-ovn/start-ovs-dpdk-v2.sh"]
          securityContext:
            runAsUser: 0
            privileged: true
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: HW_OFFLOAD
              value: "{{- .Values.components.hardwareOffload }}"
            - name: TUNNEL_TYPE
              value: "{{- .Values.networking.tunnelType }}"
            - name: DPDK_TUNNEL_IFACE
              value: "{{- .Values.networking.dpdkTunnelInterface }}"
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OVN_DB_IPS
              value: "{{ .Values.MASTER_NODES | default (include "kubeovn.nodeIPs" .) }}"
            - name: OVN_REMOTE_PROBE_INTERVAL
              value: "{{ .Values.networking.ovnRemoteProbeInterval }}"
            - name: OVN_REMOTE_OPENFLOW_INTERVAL
              value: "{{ .Values.networking.ovnRemoteOpenflowInterval }}"
          volumeMounts:
            - mountPath: /opt/ovs-config
              name: host-config-ovs
            - name: shareddir
              mountPath: {{ .Values.kubeletConfig.kubeletDir }}/pods
            - name: hugepage
              mountPath: /dev/hugepages
            - mountPath: /lib/modules
              name: host-modules
              readOnly: true
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
              mountPropagation: HostToContainer
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /sys
              name: host-sys
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            - mountPath: /var/run/tls
              name: kube-ovn-tls
          readinessProbe:
            exec:
              command:
                - bash
                - /kube-ovn/ovs-healthcheck.sh
            periodSeconds: 5
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
                - bash
                - /kube-ovn/ovs-healthcheck.sh
            initialDelaySeconds: 60
            periodSeconds: 5
            failureThreshold: 5
            timeoutSeconds: 45
          resources:
            requests:
              cpu: {{ index .Values "ovsOVN" "requests" "cpu" }}
              memory: {{ index .Values "ovsOVN" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "ovsOVN" "limits" "cpu" }}
              {{.Values.hugepageSizeType}}: {{.Values.hugePages}}
              memory: {{ index .Values "ovsOVN" "limits" "memory" }}
      nodeSelector:
        kubernetes.io/os: "linux"
        ovn.kubernetes.io/ovs_dp_type: "userspace"
      volumes:
        - name: host-config-ovs
          hostPath:
            path: /opt/ovs-config
            type: DirectoryOrCreate
        - name: shareddir
          hostPath:
            path: {{ .Values.kubeletConfig.kubeletDir }}/pods
            type: ''
        - name: hugepage
          emptyDir:
            medium: HugePages
        - name: host-modules
          hostPath:
            path: /lib/modules
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-sys
          hostPath:
            path: /sys
        - name: host-config-openvswitch
          hostPath:
            path: {{ .Values.openVSwitchDir }}
        - name: host-config-ovn
          hostPath:
            path: {{ .Values.ovnDir }}
        - name: host-log-ovs
          hostPath:
            path: {{ .Values.logConfig.logDir }}/openvswitch
        - name: host-log-ovn
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls
{{- end }}`

	kube_ovn_cni_daemonset = `kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: kube-ovn-cni
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      This daemon set launches the kube-ovn cni daemon.
spec:
  selector:
    matchLabels:
      app: kube-ovn-cni
  template:
    metadata:
      labels:
        app: kube-ovn-cni
        component: network
        type: infra
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
      priorityClassName: system-node-critical
      serviceAccountName: kube-ovn-cni
      hostNetwork: true
      hostPID: true
      initContainers:
      - name: hostpath-init
        image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
        imagePullPolicy: {{ .Values.imagePullPolicy }}
        command:
          - sh
          - -xec
          - {{ if not .Values.disableModulesManagement -}}
            iptables -V
            {{- else -}}
            echo "nothing to do"
            {{- end }}
        securityContext:
          allowPrivilegeEscalation: true
          capabilities:
            drop:
              - ALL
          privileged: true
          runAsUser: 0
          runAsGroup: 0
        volumeMounts:
          - name: usr-local-sbin
            mountPath: /usr/local/sbin
          - mountPath: /run/xtables.lock
            name: xtables-lock
            readOnly: false
          - mountPath: /var/run/netns
            name: host-ns
            readOnly: false
          - name: kube-ovn-log
            mountPath: /var/log/kube-ovn
      - name: install-cni
        image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
        imagePullPolicy: {{ .Values.imagePullPolicy }}
        command:
          - /kube-ovn/install-cni.sh
          - --cni-conf-dir={{ .Values.cniConf.cniConfigDir }}
          - --cni-conf-file={{ .Values.cniConf.cniConfFile }}
          - --cni-conf-name={{- .Values.cniConf.cniConfigPriority -}}-kube-ovn.conflist
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
          - mountPath: /opt/cni/bin
            name: cni-bin
          - mountPath: /etc/cni/net.d
            name: cni-conf
          {{- if .Values.cniConf.mountLocalBinDir }}
          - mountPath: /usr/local/bin
            name: local-bin
          {{- end }}
      containers:
      - name: cni-server
        image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
        imagePullPolicy: {{ .Values.imagePullPolicy }}
        command:
          - bash
          - /kube-ovn/start-cniserver.sh
        args:
          - --enable-mirror={{- .Values.debug.enableMirror | default "false"}}
          - --mirror-iface={{- .Values.debug.mirrorInterface }}
          - --node-switch={{ .Values.networking.nodeSubnet }}
          - --encap-checksum=true
          - --service-cluster-ip-range=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dual_stack.serviceCIDR }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.serviceCIDR }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.serviceCIDR }}
          {{- end }}
          {{- if eq .Values.networking.networkType "vlan" }}
          - --iface=
          {{- else}}
          - --iface={{- .Values.networking.interface | default ""}}
          {{- end }}
          - --dpdk-tunnel-iface={{- .Values.networking.dpdkTunnelInterface | default "br-phy"}}
          - --network-type={{- .Values.networking.tunnelType }}
          - --default-interface-name={{- .Values.networking.vlan.vlanInterface | default ""}}
          - --logtostderr=false
          - --alsologtostderr=true
          - --log_file=/var/log/kube-ovn/kube-ovn-cni.log
          - --log_file_max_size=200
          - --enable-metrics={{- .Values.networking.enableMetrics }}
          - --kubelet-dir={{ .Values.kubeletConfig.kubeletDir }}
          - --enable-tproxy={{ .Values.components.enableTProxy }}
          - --ovs-vsctl-concurrency={{ .Values.performance.ovsVSCtlConcurrency }}
          - --secure-serving={{- .Values.components.secureServing }}
          - --enable-ovn-ipsec={{- .Values.components.enableOVNIPSec }}
          - --set-vxlan-tx-off={{- .Values.components.setVLANTxOff }}
        securityContext:
          runAsUser: 0
          privileged: false
          capabilities:
            add:
              - NET_ADMIN
              - NET_BIND_SERVICE
              - NET_RAW
              - SYS_ADMIN
              - SYS_PTRACE
              {{- if not .Values.disableModulesManagement }}
              - SYS_MODULE
              {{- end }}
              - SYS_NICE
        env:
          - name: ENABLE_SSL
            value: "{{ .Values.networking.enableSSL }}"
          - name: POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
          - name: KUBE_NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: POD_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: POD_IPS
            valueFrom:
              fieldRef:
                fieldPath: status.podIPs
          - name: ENABLE_BIND_LOCAL_IP
            value: "{{- .Values.components.enableBindLocalIP }}"
          - name: DBUS_SYSTEM_BUS_ADDRESS
            value: "unix:path=/host/var/run/dbus/system_bus_socket"
        volumeMounts:
          - name: usr-local-sbin
            mountPath: /usr/local/sbin
          - name: host-modules
            mountPath: /lib/modules
            readOnly: true
          - mountPath: /run/xtables.lock
            name: xtables-lock
            readOnly: false
          - name: shared-dir
            mountPath: {{ .Values.kubeletConfig.kubeletDir }}/pods
          - mountPath: /etc/openvswitch
            name: systemid
            readOnly: true
          - mountPath: /run/openvswitch
            name: host-run-ovs
            mountPropagation: HostToContainer
          - mountPath: /run/ovn
            name: host-run-ovn
          - mountPath: /host/var/run/dbus
            name: host-dbus
            mountPropagation: HostToContainer
          - mountPath: /var/run/netns
            name: host-ns
            mountPropagation: HostToContainer
          - mountPath: /var/log/kube-ovn
            name: kube-ovn-log
          - mountPath: /var/log/openvswitch
            name: host-log-ovs
          - mountPath: /var/log/ovn
            name: host-log-ovn
          - mountPath: /etc/localtime
            name: localtime
            readOnly: true
        {{- if .Values.components.enableOVNIPSec }}
          - mountPath: /etc/ovs_ipsec_keys
            name: ovs-ipsec-keys
        {{- end }}
        readinessProbe:
          failureThreshold: 3
          periodSeconds: 7
          successThreshold: 1
          httpGet:
            port: 10665
            path: /readyz
            scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
          timeoutSeconds: 5
        livenessProbe:
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 7
          successThreshold: 1
          httpGet:
            port: 10665
            path: /livez
            scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
          timeoutSeconds: 5
        resources:
          requests:
            cpu: {{ index .Values "kubeOvnCNI" "requests" "cpu" }}
            memory: {{ index .Values "kubeOvnCNI" "requests" "memory" }}
          limits:
            cpu: {{ index .Values "kubeOvnCNI" "limits" "cpu" }}
            memory: {{ index .Values "kubeOvnCNI" "limits" "memory" }}
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: usr-local-sbin
          emptyDir: {}
        - name: host-modules
          hostPath:
            path: /lib/modules
        - name: xtables-lock
          hostPath:
            path: /run/xtables.lock
            type: FileOrCreate
        - name: shared-dir
          hostPath:
            path: {{ .Values.kubeletConfig.kubeletDir }}/pods
        - name: systemid
          hostPath:
            path: {{ .Values.openVSwitchDir }}
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: cni-conf
          hostPath:
            path: {{ .Values.cniConf.cniConfigDir }}
        - name: cni-bin
          hostPath:
            path: {{ .Values.cniConf.cniBinDir }}
        - name: host-ns
          hostPath:
            path: /var/run/netns
        - name: host-dbus
          hostPath:
            path: /var/run/dbus
        - name: kube-ovn-log
          hostPath:
            path: {{ .Values.logConfig.logDir }}/kube-ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: host-log-ovs
          hostPath:
            path: {{ .Values.logConfig.logDir }}/openvswitch
        - name: host-log-ovn
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        {{- if .Values.cniConf.mountLocalBinDir }}
        - name: local-bin
          hostPath:
            path: {{ .Values.cniConfig.mountLocalBinDir }}
        {{- end }}
        {{- if .Values.components.enableOVNIPSec }}
        - name: ovs-ipsec-keys
          hostPath:
            path: /etc/origin/ovs_ipsec_keys
        {{- end }}`

	ovs_ovn_daemonset = `kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovs-ovn
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      This daemon set launches the openvswitch daemon.
spec:
  selector:
    matchLabels:
      app: ovs
  updateStrategy:
    type: {{ include "kubeovn.ovs-ovn.upgradeStrategy" . }}
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: ovs
        component: network
        type: infra
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
      priorityClassName: system-node-critical
      serviceAccountName: ovn-ovs
      hostNetwork: true
      hostPID: true
      initContainers:
        - name: hostpath-init
          {{- if .values.dpdk }}
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.dpdkRepository }}:{{ .values.dpdkVersion }}-{{ .Values.global.images.kubeovn.tag }}
          {{- else }}
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          {{- end }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - sh
            - -xec
            - |
              chown -R nobody: /var/run/ovn /var/log/ovn /etc/openvswitch /var/run/openvswitch /var/log/openvswitch
              {{- if not .Values.disableModulesManagement }}
              iptables -V
              {{- else }}
              ln -sf /bin/true /usr/local/sbin/modprobe
              ln -sf /bin/true /usr/local/sbin/modinfo
              ln -sf /bin/true /usr/local/sbin/rmmod
              {{- end }}
          securityContext:
            allowPrivilegeEscalation: true
            capabilities:
              drop:
                - ALL
            privileged: true
            runAsUser: 0
          volumeMounts:
            - mountPath: /usr/local/sbin
              name: usr-local-sbin
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
      containers:
        - name: openvswitch
          {{- if .values.dpdk }}
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.dpdkRepository }}:{{ .values.dpdkVersion }}-{{ .Values.global.images.kubeovn.tag }}
          {{- else }}
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          {{- end }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          {{- if .values.dpdk }}
          command: ["/kube-ovn/start-ovs-dpdk.sh"]
          {{- else }}
          command: ["/kube-ovn/start-ovs.sh"]
          {{- end }}
          securityContext:
            runAsUser: {{ include "kubeovn.runAsUser" . }}
            privileged: false
            capabilities:
              add:
                - NET_ADMIN
                - NET_BIND_SERVICE
                {{- if not .Values.disableModulesManagement }}
                - SYS_MODULE
                {{- end }}
                - NET_RAW
                - SYS_NICE
                - SYS_ADMIN
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: HW_OFFLOAD
              value: "{{- .Values.components.hardwareOffload }}"
            - name: TUNNEL_TYPE
              value: "{{- .Values.networking.tunnelType }}"
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OVN_DB_IPS
              value: "{{ .Values.MASTER_NODES | default (include "kubeovn.nodeIPs" .) }}"
            - name: OVN_REMOTE_PROBE_INTERVAL
              value: "{{ .Values.networking.ovnRemoteProbeInterval }}"
            - name: OVN_REMOTE_OPENFLOW_INTERVAL
              value: "{{ .Values.networking.ovnRemoteOpenFlowInterval }}"
          volumeMounts:
            - mountPath: /usr/local/sbin
              name: usr-local-sbin
            - mountPath: /lib/modules
              name: host-modules
              readOnly: true
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            - mountPath: /var/run/tls
              name: kube-ovn-tls
            - mountPath: /var/run/containerd
              name: cruntime
              readOnly: true
            {{- if .values.dpdk }}
            - mountPath: /opt/ovs-config
              name: host-config-ovs
            - mountPath: /dev/hugepages
              name: hugepage
            {{- end }}
          readinessProbe:
            exec:
              {{- if .values.dpdk }}
              command:
                - bash
                - /kube-ovn/ovs-dpdk-healthcheck.sh
              {{- else }}
              command:
                - bash
                - /kube-ovn/ovs-healthcheck.sh
              {{- end }}
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 45
          livenessProbe:
            exec:
              {{- if .values.dpdk }}
              command:
                - bash
                - /kube-ovn/ovs-dpdk-healthcheck.sh
              {{- else }}
              command:
                - bash
                - /kube-ovn/ovs-healthcheck.sh
              {{- end }}
            initialDelaySeconds: 60
            periodSeconds: 5
            failureThreshold: 5
            timeoutSeconds: 45
          resources:
            requests:
              {{- if .values.dpdk }}
              cpu: {{ .values.dpdkCPU }}
              memory: {{ .values.dpdkMEMORY }}
              {{- else }}
              cpu: {{ index .Values "ovsOVN" "requests" "cpu" }}
              memory: {{ index .Values "ovsOVN" "requests" "memory" }}
              {{- end }}
            limits:
              {{- if .values.dpdk }}
              cpu: {{ .values.dpdkCPU }}
              memory: {{ .values.dpdkMEMORY }}
              hugepages-1Gi: 1Gi
              {{- else }}
              cpu: {{ index .Values "ovsOVN" "limits" "cpu" }}
              memory: {{ index .Values "ovsOVN" "limits" "memory" }}
              {{- end }}
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: usr-local-sbin
          emptyDir: {}
        - name: host-modules
          hostPath:
            path: /lib/modules
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-config-openvswitch
          hostPath:
            path: {{ .Values.openVSwitchDir }}
        - name: host-log-ovs
          hostPath:
            path: {{ .Values.logConfig.logDir }}/openvswitch
        - name: host-log-ovn
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls
        - hostPath:
            path: /var/run/containerd
          name: cruntime
        {{- if .values.dpdk }}
        - name: host-config-ovs
          hostPath:
            path: /opt/ovs-config
            type: DirectoryOrCreate
        - name: hugepage
          emptyDir:
            medium: HugePages
        {{- end }}`

	kube_ovn_pinger_daemonsets = `kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: kube-ovn-pinger
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      This daemon set launches the openvswitch daemon.
spec:
  selector:
    matchLabels:
      app: kube-ovn-pinger
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: kube-ovn-pinger
        component: network
        type: infra
    spec:
      priorityClassName: system-node-critical
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
      serviceAccountName: kube-ovn-app
      hostPID: true
      initContainers:
        - name: hostpath-init
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - sh
            - -c
            - "chown -R nobody: /var/log/kube-ovn"
          securityContext:
            allowPrivilegeEscalation: true
            capabilities:
              drop:
                - ALL
            privileged: true
            runAsUser: 0
          volumeMounts:
            - name: kube-ovn-log
              mountPath: /var/log/kube-ovn
      containers:
        - name: pinger
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          command:
          - /kube-ovn/kube-ovn-pinger
          args:
          - --external-address=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.pingerExternalAddress }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.pingerExternalAddress }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.pingerExternalAddress }}
          {{- end }}
          - --external-dns=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.pingerExternalDomain }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.pingerExternalDomain }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.pingerExternalDomain }}
          {{- end }}
          - --ds-namespace={{ .Values.namespace }}
          - --logtostderr=false
          - --alsologtostderr=true
          - --log_file=/var/log/kube-ovn/kube-ovn-pinger.log
          - --log_file_max_size=200
          - --enable-metrics={{- .Values.networking.enableMetrics }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          securityContext:
            runAsUser: {{ include "kubeovn.runAsUser" . }}
            privileged: false
            capabilities:
              add:
                - NET_BIND_SERVICE
                - NET_RAW
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: HOST_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.hostIP
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
              readOnly: true
            - mountPath: /var/log/ovn
              name: host-log-ovn
              readOnly: true
            - mountPath: /var/log/kube-ovn
              name: kube-ovn-log
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            - mountPath: /var/run/tls
              name: kube-ovn-tls
          resources:
            requests:
              cpu: {{ index .Values "kubeOvnPinger" "requests" "cpu" }}
              memory: {{ index .Values "kubeOvnPinger" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "kubeOvnPinger" "limits" "cpu" }}
              memory: {{ index .Values "kubeOvnPinger" "limits" "memory" }}
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-config-openvswitch
          hostPath:
            path: {{ .Values.openVSwitchDir }}
        - name: host-log-ovs
          hostPath:
            path: {{ .Values.logConfig.logDir }}/openvswitch
        - name: kube-ovn-log
          hostPath:
            path: {{ .Values.logConfig.logDir }}/kube-ovn
        - name: host-log-ovn
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls`

	DaemonsetList = []string{ovs_ovn_dpdk_daemonset, kube_ovn_cni_daemonset, ovs_ovn_daemonset, kube_ovn_pinger_daemonsets}
)
