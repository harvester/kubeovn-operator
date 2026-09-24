package templates

var (
	ovn_central_deployment = `
{{- /*
Guard against in-place mode switches that would silently drop the live OVN
DB. Switching from raft cluster (hostPath /etc/ovn) to single (PVC) leaves
the new pod starting against an empty PVC; reverse direction loses the
PVC-backed DB. Detect the existing Deployment's volume type via lookup and
fail with explicit migration instructions if it disagrees with the rendered
OVN_CENTRAL_MODE. lookup returns empty during dry-run / template, so this
only fires on a real install/upgrade against a cluster.
*/}}
{{- $existing := lookup "apps/v1" "Deployment" .Values.namespace "ovn-central" }}
{{- if not (index .Values "ovnCentral" "hcp" "enabled") }}
{{- if $existing }}
  {{- range $existing.spec.template.spec.volumes }}
    {{- if eq .name "host-config-ovn" }}
      {{- if and (eq $.Values.ovnCentralMode "single") .hostPath }}
        {{- fail "Refusing to switch OVN_CENTRAL_MODE in place from raft (hostPath) to single (PVC). The new render would start ovn-central against an empty PVC and lose live OVN state. Migrate explicitly: (1) snapshot the current DB with 'bash dist/images/restore-ovn-nb-db.sh' style backup; (2) 'helm uninstall' the current release; (3) 'helm install' with OVN_CENTRAL_MODE=single onto a fresh namespace or after clearing the existing ovn-central Deployment; (4) restore the snapshot via 'bash dist/images/restore-ovn-nb-db.sh single <backup.db>'." }}
      {{- end }}
      {{- if and (ne $.Values.ovnCentralMode "single") .persistentVolumeClaim }}
        {{- fail "Refusing to switch OVN_CENTRAL_MODE in place from single (PVC) to raft (hostPath). The new render would lose the PVC-backed DB. Migrate explicitly: snapshot the DB, helm uninstall, then helm install in the target mode and restore the snapshot." }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}
kind: {{ ternary "StatefulSet" "Deployment" (index .Values "ovnCentral" "hcp" "enabled") }}
apiVersion: apps/v1
metadata:
  name: ovn-central
  namespace: {{ include "kubeovn.centralNamespace" . }}
  annotations:
    kubernetes.io/description: |
      OVN components: northd, nb and sb.
spec:
  replicas: {{ include "kubeovn.ovnCentralReplicas" . }}
  {{- if index .Values "ovnCentral" "hcp" "enabled" }}
  serviceName: ovn-central
  podManagementPolicy: Parallel
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Delete
    whenScaled: Delete
  {{- else }}
  strategy:
    {{- if eq .Values.ovnCentralMode "single" }}
    type: Recreate
    {{- else }}
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
    {{- end }}
  {{- end }}
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
        {{- with (index .Values "ovnCentral" "tolerations") }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      affinity:
        {{- if or (index .Values "ovnCentral" "hcp" "enabled") (ne .Values.ovnCentralMode "single") }}
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: ovn-central
              topologyKey: kubernetes.io/hostname
        {{- end }}
        {{- if not (index .Values "ovnCentral" "hcp" "enabled") }}
        {{- include "kubeovn.masterNodeAffinity" . | nindent 8 }}
        {{- end }}
      priorityClassName: system-cluster-critical
      serviceAccountName: {{ ternary "ovn-central" "ovn-ovs" (index .Values "ovnCentral" "hcp" "enabled") }}
      automountServiceAccountToken: true
      hostNetwork: {{ ternary "false" "true" (index .Values "ovnCentral" "hcp" "enabled") }}
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      initContainers:
        - name: {{ ternary "volume-init" "hostpath-init" (index .Values "ovnCentral" "hcp" "enabled") }}
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - sh
            - -c
            {{- if index .Values "ovnCentral" "hcp" "enabled" }}
            - "chown -R nobody: /var/run/ovn /etc/ovn /var/log/ovn"
            {{- else }}
            {{- if eq .Values.ovnCentralMode "single" }}
            # /etc/ovn comes from a PVC in single-replica mode. Backends like
            # root-squashed NFS reject chown by root, so make the OVN DB
            # directory chown best-effort; the OVN containers run as "nobody"
            # and will create new files with the correct owner regardless.
            - "chown -R nobody: /var/run/ovn /var/log/ovn && (chown -R nobody: /etc/ovn 2>/dev/null || echo 'chown /etc/ovn skipped (likely root-squashed NFS); ovn will write as nobody anyway')"
            {{- else }}
            - "chown -R nobody: /var/run/ovn /etc/ovn /var/log/ovn"
            {{- end }}
            {{- end }}
          securityContext:
            allowPrivilegeEscalation: {{ ternary "false" "true" (index .Values "ovnCentral" "hcp" "enabled") }}
            capabilities:
              {{- if index .Values "ovnCentral" "hcp" "enabled" }}
              add:
                - CHOWN
              {{- end }}
              drop:
                - ALL
            privileged: {{ ternary "false" "true" (index .Values "ovnCentral" "hcp" "enabled") }}
            runAsUser: 0
          volumeMounts:
            - mountPath: /var/run/ovn
              name: {{ ternary "run-ovn" "host-run-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
            - mountPath: /etc/ovn
              name: {{ ternary "ovn-data" "host-config-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
            - mountPath: /var/log/ovn
              name: {{ ternary "log-ovn" "host-log-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
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
{{- include "kubeovn.ovnCentralTLSEnv" . | nindent 12 }}
            {{- if index .Values "ovnCentral" "hcp" "enabled" }}
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            {{- end }}
            - name: NODE_IPS
              value: "{{ include "kubeovn.ovnCentralNodeIPs" . }}"
            {{- if index .Values "ovnCentral" "hcp" "enabled" }}
            - name: DB_CLUSTER_ADDR
              value: "$(POD_NAME).ovn-central.$(POD_NAMESPACE).svc"
            {{- end }}
            - name: POD_IP
              {{- if index .Values "ovnCentral" "hcp" "enabled" }}
              value: "$(DB_CLUSTER_ADDR)"
              {{- else }}
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
              {{- end }}
            {{- if not (index .Values "ovnCentral" "hcp" "enabled") }}
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            {{- end }}
            - name: POD_IPS
              valueFrom:
                fieldRef:
                  fieldPath: status.podIPs
            - name: ENABLE_BIND_LOCAL_IP
              value: "{{ ternary false .Values.func.ENABLE_BIND_LOCAL_IP (index .Values "ovnCentral" "hcp" "enabled") }}"
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
          resources:
            requests:
              cpu: {{ index .Values "ovnCentral" "requests" "cpu" }}
              memory: {{ index .Values "ovnCentral" "requests" "memory" }}
            limits:
              cpu: {{ index .Values "ovnCentral" "limits" "cpu" }}
              memory: {{ index .Values "ovnCentral" "limits" "memory" }}
              ephemeral-storage: {{ index .Values "ovnCentral" "limits" "ephemeralStorage" }}
          volumeMounts:
            - mountPath: /var/run/ovn
              name: {{ ternary "run-ovn" "host-run-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
            - mountPath: /etc/ovn
              name: {{ ternary "ovn-data" "host-config-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
            - mountPath: /var/log/ovn
              name: {{ ternary "log-ovn" "host-log-ovn" (index .Values "ovnCentral" "hcp" "enabled") }}
            {{- if not (index .Values "ovnCentral" "hcp" "enabled") }}
            - mountPath: /etc/localtime
              name: localtime
              readOnly: true
            {{- end }}
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
      {{- $nodeSelector := index .Values "ovnCentral" "nodeSelector" }}
      {{- if $nodeSelector }}
      nodeSelector:
        {{- toYaml $nodeSelector | nindent 8 }}
      {{- end }}
      volumes:
        {{- if index .Values "ovnCentral" "hcp" "enabled" }}
        - name: run-ovn
          emptyDir: {}
        - name: log-ovn
          emptyDir: {}
        {{- else }}
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-config-ovn
          {{- if eq .Values.ovnCentralMode "single" }}
          persistentVolumeClaim:
            claimName: {{ (index .Values "ovnCentral" "storage").existingClaim | default "ovn-central-data" }}
          {{- else }}
          hostPath:
            path: {{ .Values.ovnDir }}
          {{- end }}
        - name: host-log-ovn
          hostPath:
            path: {{ .Values.logConfig.logDir }}/ovn
        - name: localtime
          hostPath:
            path: /etc/localtime
        {{- end }}
        - name: kube-ovn-tls
          secret:
            optional: true
            secretName: kube-ovn-tls

  {{- if index .Values "ovnCentral" "hcp" "enabled" }}
  volumeClaimTemplates:
    - metadata:
        name: ovn-data
        labels:
          app: ovn-central
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: {{ index .Values "ovnCentral" "hcp" "storage" "size" }}
        {{- if index .Values "ovnCentral" "hcp" "storage" "storageClassName" }}
        storageClassName: {{ index .Values "ovnCentral" "hcp" "storage" "storageClassName" | quote }}
        {{- end }}
  {{- end }}
  `

	kube_ovn_controller_deployment = `kind: Deployment
apiVersion: apps/v1
metadata:
  name: kube-ovn-controller
  namespace: {{ .Values.namespace }}
  annotations:
    kubernetes.io/description: |
      kube-ovn controller
spec:
  replicas: {{ include "kubeovn.controllerReplicas" . }}
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
      automountServiceAccountToken: true
      hostNetwork: true
      securityContext:
        seccompProfile:
          type: RuntimeDefault
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
          {{- if (index .Values "kubeOvnController" "leaderElection" "leaseDuration") }}
          - --leader-elect-lease-duration={{ index .Values "kubeOvnController" "leaderElection" "leaseDuration" }}
          {{- end }}
          {{- if (index .Values "kubeOvnController" "leaderElection" "renewDeadline") }}
          - --leader-elect-renew-deadline={{ index .Values "kubeOvnController" "leaderElection" "renewDeadline" }}
          {{- end }}
          {{- if (index .Values "kubeOvnController" "leaderElection" "retryPeriod") }}
          - --leader-elect-retry-period={{ index .Values "kubeOvnController" "leaderElection" "retryPeriod" }}
          {{- end }}
          - --non-primary-cni-mode={{ .Values.cniConf.nonPrimaryCNI }}
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
          {{- if .Values.networking.externalGatewayConfigNS }}
          - --external-gateway-config-ns={{- .Values.networking.externalGatewayConfigNS }}
          {{- end }}
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
          - --np-enforcement={{- .Values.components.npEnforcement }}
          - --enable-acl-sampling={{- .Values.aclSampling.enabled }}
          - --acl-sampling-set-id={{- .Values.aclSampling.setID }}
          - --acl-sampling-app-id-new={{- .Values.aclSampling.appIDNew }}
          - --acl-sampling-app-id-established={{- .Values.aclSampling.appIDEstablished }}
          - --acl-sampling-collector-id-allow={{- .Values.aclSampling.collectorIDAllow }}
          - --acl-sampling-collector-id-default-deny={{- .Values.aclSampling.collectorIDDefaultDeny }}
          - --acl-sampling-allow-probability-percent={{- .Values.aclSampling.allowProbabilityPercent }}
          - --acl-sampling-default-deny-probability-percent={{- .Values.aclSampling.defaultDenyProbabilityPercent }}
          - --enable-live-migration-optimize={{- .Values.components.enableLiveMigrationOptimize }}
          - --enable-ovn-lb-prefer-local={{- .Values.components.enableOVNLBPreferLocal }}
          - --image={{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          - --skip-conntrack-dst-cidrs={{- .Values.networking.skipConnTrackDstCIDRs | default ""}}
          - --enable-dns-name-resolver={{- .Values.components.enableDNSNameResolver }}
          {{- if or .Values.networking.tlsMinVersion .Values.networking.tlsMaxVersion .Values.networking.tlsCipherSuites }}
          {{- include "kubeovn.componentTLSArgs" . | nindent 10 }}
          {{- end }}
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
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            {{- if index .Values "ovnCentral" "hcp" "enabled" }}
            - name: OVN_NB_ADDR
              value: "{{ include "kubeovn.ovnNbAddress" . }}"
            - name: OVN_SB_ADDR
              value: "{{ include "kubeovn.ovnSbAddress" . }}"
            {{- else if eq .Values.installMode "dataPlaneOnly" }}
            - name: OVN_NB_ADDR
              value: "{{ include "kubeovn.externalOvnNbAddress" . }}"
            - name: OVN_SB_ADDR
              value: "{{ include "kubeovn.externalOvnSbAddress" . }}"
            {{- else }}
            - name: OVN_DB_IPS
              value: "{{ include "kubeovn.ovnCentralNodeIPs" . }}"
            - name: KUBE_OVN_NB_PORT
              value: "{{ include "kubeovn.ovnNbPort" . }}"
            - name: KUBE_OVN_SB_PORT
              value: "{{ include "kubeovn.ovnSbPort" . }}"
            {{- end }}
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
              ephemeral-storage: {{ index .Values "kubeOvnController" "limits" "ephemeralStorage" }}
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
        {{- include "kubeovn.masterNodeAffinity" . | nindent 8 }}      
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      automountServiceAccountToken: true
      hostNetwork: true
      securityContext:
        seccompProfile:
          type: RuntimeDefault
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
             {{- if index .Values "ovnCentral" "hcp" "enabled" }}
            - name: OVN_NB_ADDR
              value: "{{ include "kubeovn.ovnNbAddress" . }}"
            - name: OVN_SB_ADDR
              value: "{{ include "kubeovn.ovnSbAddress" . }}"
            {{- else if eq .Values.installMode "dataPlaneOnly" }}
            - name: OVN_NB_ADDR
              value: "{{ include "kubeovn.externalOvnNbAddress" . }}"
            - name: OVN_SB_ADDR
              value: "{{ include "kubeovn.externalOvnSbAddress" . }}"
            {{- else }}
            - name: OVN_DB_IPS
              value: "{{ include "kubeovn.ovnCentralNodeIPs" . }}"
            - name: KUBE_OVN_NB_PORT
              value: "{{ include "kubeovn.ovnNbPort" . }}"
            - name: KUBE_OVN_SB_PORT
              value: "{{ include "kubeovn.ovnSbPort" . }}"
            {{- end }}
          resources:
            requests:
              cpu: 300m
              memory: 200Mi
            limits:
              cpu: 3
              memory: 1Gi
              ephemeral-storage: 1Gi
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
        {{- include "kubeovn.masterNodeAffinity" . | nindent 8 }}      
      priorityClassName: system-cluster-critical
      serviceAccountName: kube-ovn-app
      automountServiceAccountToken: true
      hostNetwork: true
      securityContext:
        seccompProfile:
          type: RuntimeDefault
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
          {{- if or .Values.networking.tlsMinVersion .Values.networking.tlsMaxVersion .Values.networking.tlsCipherSuites }}
          {{- include "kubeovn.componentTLSArgs" . | nindent 10 }}
          {{- end }}
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
            - name: NODE_NAME
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
              ephemeral-storage: {{ index .Values "kubeOvnMonitor" "limits" "ephemeralStorage" }}
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

	kubeovn_webhook_deployment = `kind: Deployment
apiVersion: apps/v1
metadata:
  name: kube-ovn-webhook
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-ovn-webhook
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 100%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: kube-ovn-webhook
    spec:
      tolerations:
        - operator: Exists
          effect: NoSchedule
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: kube-ovn-webhook
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical        
      serviceAccountName: ovn
      automountServiceAccountToken: true
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: kube-ovn-webhook
          image: {{ .Values.global.registry.address }}/{{ .Values.global.images.kubeovn.repository }}:{{ .Values.global.images.kubeovn.tag }}
          imagePullPolicy: {{ .Values.imagePullPolicy }}
          command:
            - /kube-ovn/kube-ovn-webhook
          args:
            - --port=8443
            - --health-probe-port=8080
            - --v=3
          env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: status.podIP
          volumeMounts:
            - mountPath: /tmp/k8s-webhook-server/serving-certs
              name: cert
              readOnly: true
          ports:
          - containerPort: 8443
            name: https
            protocol: TCP
          - containerPort: 8080
            name: health-probe
            protocol: TCP
          livenessProbe:
            failureThreshold: 3
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 60
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 1
          readinessProbe:
            failureThreshold: 3
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
            successThreshold: 1
            timeoutSeconds: 1
      volumes:
        - name: cert
          secret:
            defaultMode: 420
            secretName: webhook-certs
      nodeSelector:
        kubernetes.io/os: "linux"`

	DeploymentList = []string{ovn_central_deployment, kube_ovn_controller_deployment, ovn_ic_controller_deployment, kube_ovn_monitor_deployment, kubeovn_webhook_deployment}
)
