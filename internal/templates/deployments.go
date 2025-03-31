package templates

var (
	ovn_central_deployment = `kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovn-central
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      OVN components: northd, nb and sb.
spec:
  replicas: {{ include "kubeovn.nodeCount" . }}
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
  selector:
    matchLabels:
      app: ovn-central
  template:
    metadata:
      labels:
        app: ovn-central
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
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: ovn-central
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn-ovs
      hostNetwork: true
      initContainers:
        - name: hostpath-init
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - sh
            - -c
            - "chown -R nobody: /var/run/ovn /etc/ovn /var/log/ovn"
          securityContext:
            allowPrivilegeEscalation: true
            capabfserilities:
              drop:
                - ALL
            privileged: true
            runAsUser: 0
          volumeMounts:
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/ovn
              name: host-log-ovn
      containers:
        - name: ovn-central
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command: 
          - bash
          - /kube-ovn/start-db.sh
          securityContext:
            runAsUser: {{ include "kubeovn.runAsUser" . }}
            privileged: false
            capabilities:
              add:
                - NET_BIND_SERVICE
                - SYS_NICE
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: NODE_IPS
              value: "{{ .Values.MASTER_NODES | default (include "kubeovn.nodeIPs" .) }}"
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
            - name: POD_IPS
              valueFrom:
                fieldRef:
                  fieldPath: status.podIPs
            - name: ENABLE_BIND_LOCAL_IP
              value: "{{- .Values.components.enableBindLocalIP }}"
            - name: PROBE_INTERVAL
              value: "{{ .Values.networking.probeInterval }}"
            - name: OVN_NORTHD_PROBE_INTERVAL
              value: "{{ .Values.networking.ovnNorthdProbeInterval}}"
            - name: OVN_LEADER_PROBE_INTERVAL
              value: "{{ .Values.networking.ovnLeaderProbeInterval }}"
            - name: OVN_NORTHD_N_THREADS
              value: "{{ .Values.networking.ovnNorthdNThreads }}"
            - name: ENABLE_COMPACT
              value: "{{ .Values.networking.enableCompact }}"
            - name: OVN_VERSION_COMPATIBILITY
              value: '{{ include "kubeovn.ovn.versionCompatibility" . }}'
          resources:
            requests:
              cpu: {{ index .Values "ovnCentral" "requests" "cpu" }}
              memory: {{ index .Values "ovnCentral" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "ovnCentral" "limits" "cpu" }}
              memory: {{ index .Values "ovnCentral" "limits" "memory" }}
          volumeMounts:
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/ovn
              name: host-config-ovn
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
                - /kube-ovn/ovn-healthcheck.sh
            periodSeconds: 15
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
                - bash
                - /kube-ovn/ovn-healthcheck.sh
            initialDelaySeconds: 30
            periodSeconds: 15
            failureThreshold: 5
            timeoutSeconds: 45
      nodeSelector:
        kubernetes.io/os: "linux"
        {{- with splitList "=" .Values.masterNodesLabel }}
        {{ index . 0 }}: "{{ if eq (len .) 2 }}{{ index . 1 }}{{ end }}"
        {{- end }}
      volumes:
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-config-ovn
          hostPath:
            path: {{ .Values.ovnDir }}
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

	kube_ovn_controller_deployment = `kind: Deployment
apiVersion: apps/v1
metadata:
  name: kube-ovn-controller
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      kube-ovn controller
spec:
  replicas: {{ include "kubeovn.nodeCount" . }}
  selector:
    matchLabels:
      app: kube-ovn-controller
  strategy:
    rollingUpdate:
      maxSurge: 0%
      maxUnavailable: 100%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: kube-ovn-controller
        component: network
        type: infra
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - preference:
                matchExpressions:
                  - key: "ovn.kubernetes.io/ic-gw"
                    operator: NotIn
                    values:
                      - "true"
              weight: 100
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: kube-ovn-controller
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
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
        - name: kube-ovn-controller
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          args:
          - /kube-ovn/start-controller.sh
          - --default-ls={{ .Values.networking.defaultSubnet }}
          - --default-cidr=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.podCIDR }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.podCIDR }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.podCIDR }}
          {{- end }}
          - --default-gateway=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.podGateway }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.podGateway }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.podGateway }}
          {{- end }}
          - --default-gateway-check={{- .Values.components.checkGateway }}
          - --default-logical-gateway={{- .Values.components.logicalGateway }}
          - --default-u2o-interconnection={{- .Values.components.u2oInterconnection }}
          - --default-exclude-ips={{- .Values.networking.excludeIPS | default ""}}
          - --cluster-router={{ .Values.networking.defaultVPC }}
          - --node-switch={{ .Values.networking.nodeSubnet }}
          - --node-switch-cidr=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.joinCIDR }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.joinCIDR }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.joinCIDR }}
          {{- end }}
          - --service-cluster-ip-range=
          {{- if eq .Values.networking.netStack "dual_stack" -}}
          {{ .Values.dualStack.serviceCIDR }}
          {{- else if eq .Values.networking.netStack "ipv4" -}}
          {{ .Values.ipv4.serviceCIDR }}
          {{- else if eq .Values.networking.netStack "ipv6" -}}
          {{ .Values.ipv6.serviceCIDR }}
          {{- end }}
          - --network-type={{- .Values.networking.networkType }}
          - --default-provider-name={{ .Values.networking.vlan.providerName }}
          {{ if .Values.networking.vlan.vlanInterface -}}
          - --default-interface-name={{- .Values.networking.vlan.vlanInterface }}
          {{- end }}
          - --default-exchange-link-name={{- .Values.networking.exchangeLinkName }}
          - --default-vlan-name={{- .Values.networking.vlan.vlanName }}
          - --default-vlan-id={{- .Values.networking.vlan.vlanId }}
          - --ls-dnat-mod-dl-dst={{- .Values.components.lsDnatModDlDst }}
          - --ls-ct-skip-dst-lport-ips={{- .Values.components.lsCtSkipOstLportIPS }}
          - --pod-nic-type={{- .Values.networking.podNicType }}
          - --enable-lb={{- .Values.components.enableLB }}
          - --enable-np={{- .Values.components.enableNP }}
          - --enable-eip-snat={{- .Values.networking.enableEIPSNAT }}
          - --enable-external-vpc={{- .Values.components.enableExternalVPC }}
          - --enable-ecmp={{- .Values.networking.enableECMP }}
          - --logtostderr=false
          - --alsologtostderr=true
          - --gc-interval={{- .Values.performance.gcInterval }}
          - --inspect-interval={{- .Values.performance.inspectInterval }}
          - --log_file=/var/log/kube-ovn/kube-ovn-controller.log
          - --log_file_max_size=200
          - --enable-lb-svc={{- .Values.components.enableLBSVC }}
          - --keep-vm-ip={{- .Values.components.enableKeepVMIP }}
          - --enable-metrics={{- .Values.networking.enableMetrics }}
          {{ if .Values.networking.nodeLocalDNSIPS -}}
          - --node-local-dns-ip={{- .Values.networking.nodeLocalDNSIPS }}
          {{- end }}
          - --secure-serving={{- .Values.components.secureServing }}
          - --enable-ovn-ipsec={{- .Values.components.enableOVNIPSec }}
          - --enable-anp={{- .Values.components.enableANP }}
          - --ovsdb-con-timeout={{- .Values.components.OVSDBConTimeout }}
          - --ovsdb-inactivity-timeout={{- .Values.components.OVSDBInactivityTimeout }}
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
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: KUBE_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OVN_DB_IPS
              value: "{{ .Values.MASTER_NODES | default (include "kubeovn.nodeIPs" .) }}"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_IPS
              valueFrom:
                fieldRef:
                  fieldPath: status.podIPs
            - name: ENABLE_BIND_LOCAL_IP
              value: "{{- .Values.components.enableBindLocalIP }}"
          volumeMounts:
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            - mountPath: /var/log/kube-ovn
              name: kube-ovn-log
            # ovn-ic log directory
            - mountPath: /var/log/ovn
              name: ovn-log
            - mountPath: /var/run/tls
              name: kube-ovn-tls
          readinessProbe:
            httpGet:
              port: 10660
              path: /readyz
              scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
            periodSeconds: 3
            timeoutSeconds: 5
          livenessProbe:
            httpGet:
              port: 10660
              path: /livez
              scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
            initialDelaySeconds: 300
            periodSeconds: 7
            failureThreshold: 5
            timeoutSeconds: 5
          resources:
            requests:
              cpu: {{ index .Values "kubeOvnController" "requests" "cpu" }}
              memory: {{ index .Values "kubeOvnController" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "kubeOvnController" "limits" "cpu" }}
              memory: {{ index .Values "kubeOvnController" "limits" "memory" }}
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: kube-ovn-log
          hostPath:
            path: {{ .Values.logConfig.logDir }}/kube-ovn
        - name: ovn-log
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls`

	ovn_ic_controller_deployment = `{{- if .Values.components.enableIC }}
kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovn-ic-controller
  namespace: kube-system
  annotations:
    kubernetes.io/description: |
      OVN IC Client
spec:
  replicas: 1
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
  selector:
    matchLabels:
      app: ovn-ic-controller
  template:
    metadata:
      labels:
        app: ovn-ic-controller
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
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: ovn-ic-controller
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
      initContainers:
        - name: hostpath-init
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - sh
            - -c
            - "chown -R nobody: /var/run/ovn /var/log/ovn /var/log/kube-ovn"
          securityContext:
            allowPrivilegeEscalation: true
            capabilities:
              drop:
                - ALL
            privileged: true
            runAsUser: 0
          volumeMounts:
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - name: kube-ovn-log
              mountPath: /var/log/kube-ovn
      containers:
        - name: ovn-ic-controller
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command: ["/kube-ovn/start-ic-controller.sh"]
          args:
          - --log_file=/var/log/kube-ovn/kube-ovn-ic-controller.log
          - --log_file_max_size=200
          - --logtostderr=false
          - --alsologtostderr=true
          securityContext:
            runAsUser: {{ include "kubeovn.runAsUser" . }}
            privileged: false
            capabilities:
              add:
                - NET_BIND_SERVICE
                - SYS_NICE
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OVN_DB_IPS
              value: "{{ .Values.MASTER_NODES | default (include "kubeovn.nodeIPs" .) }}"
          resources:
            requests:
              cpu: 300m
              memory: 200Mi
            limits:
              cpu: 3
              memory: 1Gi
          volumeMounts:
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /etc/localtime
              name: localtime
            - mountPath: /var/run/tls
              name: kube-ovn-tls
            - mountPath: /var/log/kube-ovn
              name: kube-ovn-log
      nodeSelector:
        kubernetes.io/os: "linux"
        kube-ovn/role: "master"
      volumes:
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-log-ovn
          hostPath:
            path: /var/log/ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        - name: kube-ovn-log
          hostPath:
            path: /var/log/kube-ovn
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls
{{- end }}`

	kube_ovn_monitor_deployment = `kind: Deployment
apiVersion: apps/v1
metadata:
  name: kube-ovn-monitor
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      Metrics for OVN components: northd, nb and sb.
spec:
  replicas: 1
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  selector:
    matchLabels:
      app: kube-ovn-monitor
  template:
    metadata:
      labels:
        app: kube-ovn-monitor
        component: network
        type: infra
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: kube-ovn-monitor
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: kube-ovn-app
      hostNetwork: true
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
        - name: kube-ovn-monitor
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command: ["/kube-ovn/start-ovn-monitor.sh"]
          args:
          - --secure-serving={{- .Values.components.secureServing }}
          - --log_file=/var/log/kube-ovn/kube-ovn-monitor.log
          - --logtostderr=false
          - --alsologtostderr=true
          - --log_file_max_size=200
          - --enable-metrics={{- .Values.networking.enableMetrics }}
          securityContext:
            runAsUser: {{ include "kubeovn.runAsUser" . }}
            privileged: false
            capabilities:
              add:
                - NET_BIND_SERVICE
          env:
            - name: ENABLE_SSL
              value: "{{ .Values.networking.enableSSL }}"
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_IPS
              valueFrom:
                fieldRef:
                  fieldPath: status.podIPs
            - name: ENABLE_BIND_LOCAL_IP
              value: "{{- .Values.components.enableBindLocalIP }}"
          resources:
            requests:
              cpu: {{ index .Values "kubeOvnMonitor" "requests" "cpu" }}
              memory: {{ index .Values "kubeOvnMonitor" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "kubeOvnMonitor" "limits" "cpu" }}
              memory: {{ index .Values "kubeOvnMonitor" "limits" "memory" }}
          volumeMounts:
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/ovn
              name: host-log-ovn
              readOnly: true
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            - mountPath: /var/run/tls
              name: kube-ovn-tls
            - mountPath: /var/log/kube-ovn
              name: kube-ovn-log
          livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 30
            periodSeconds: 7
            successThreshold: 1
            httpGet:
              port: 10661
              path: /livez
              scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
            timeoutSeconds: 5
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 30
            periodSeconds: 7
            successThreshold: 1
            httpGet:
              port: 10661
              path: /readyz
              scheme: '{{ ternary "HTTPS" "HTTP" .Values.components.secureServing }}'
            timeoutSeconds: 5
      nodeSelector:
        kubernetes.io/os: "linux"
        {{- with splitList "=" .Values.masterNodesLabel }}
        {{ index . 0 }}: "{{ if eq (len .) 2 }}{{ index . 1 }}{{ end }}"
        {{- end }}
      volumes:
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-config-ovn
          hostPath:
            path: {{ .Values.ovnDir }}
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
        - name: kube-ovn-log
          hostPath:
            path: {{ .Values.logConfig.logDir }}/kube-ovn
`

	DeploymentList = []string{ovn_central_deployment, kube_ovn_controller_deployment, ovn_ic_controller_deployment, kube_ovn_monitor_deployment}
)
