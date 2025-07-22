{{/*
Expand the name of the chart.
*/}}
{{- define "near-rt-ric.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "near-rt-ric.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "near-rt-ric.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "near-rt-ric.labels" -}}
helm.sh/chart: {{ include "near-rt-ric.chart" . }}
{{ include "near-rt-ric.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "near-rt-ric.selectorLabels" -}}
app.kubernetes.io/name: {{ include "near-rt-ric.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "near-rt-ric.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "near-rt-ric.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create image pull secrets
*/}}
{{- define "near-rt-ric.imagePullSecrets" -}}
{{- $pullSecrets := list }}
{{- if .Values.global.imagePullSecrets }}
{{- $pullSecrets = concat $pullSecrets .Values.global.imagePullSecrets }}
{{- end }}
{{- if .Values.image.pullSecrets }}
{{- $pullSecrets = concat $pullSecrets .Values.image.pullSecrets }}
{{- end }}
{{- if $pullSecrets }}
imagePullSecrets:
{{- range $pullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "near-rt-ric.image" -}}
{{- $registryName := .Values.image.registry -}}
{{- $repositoryName := .Values.image.repository -}}
{{- $tag := .Values.image.tag | toString -}}
{{- if .Values.global.imageRegistry }}
    {{- $registryName = .Values.global.imageRegistry -}}
{{- end -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end }}

{{/*
Return the proper interface image name
*/}}
{{- define "near-rt-ric.interfaceImage" -}}
{{- $registryName := .Values.image.registry -}}
{{- $repositoryName := .interface.image.repository -}}
{{- $tag := .interface.image.tag | toString -}}
{{- if .Values.global.imageRegistry }}
    {{- $registryName = .Values.global.imageRegistry -}}
{{- end -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end }}

{{/*
Return the proper Storage Class
*/}}
{{- define "near-rt-ric.storageClass" -}}
{{- $storageClass := .Values.persistence.storageClass -}}
{{- if .Values.global.storageClass }}
    {{- $storageClass = .Values.global.storageClass -}}
{{- end -}}
{{- if $storageClass }}
  storageClassName: {{ $storageClass | quote }}
{{- end -}}
{{- end }}

{{/*
Generate certificates for near-rt-ric
*/}}
{{- define "near-rt-ric.gen-certs" -}}
{{- $altNames := list ( printf "%s.%s" (include "near-rt-ric.fullname" .) .Release.Namespace ) ( printf "%s.%s.svc" (include "near-rt-ric.fullname" .) .Release.Namespace ) -}}
{{- $ca := genCA "near-rt-ric-ca" 365 -}}
{{- $cert := genSignedCert ( include "near-rt-ric.fullname" . ) nil $altNames 365 $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end }}

{{/*
Return whether to create a secret for certificates
*/}}
{{- define "near-rt-ric.createTLSSecret" -}}
{{- if and .Values.ingress.enabled .Values.ingress.tls -}}
{{- range .Values.ingress.tls -}}
{{- if not .secretName -}}
{{- printf "true" -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Common environment variables
*/}}
{{- define "near-rt-ric.commonEnvVars" -}}
- name: LOG_LEVEL
  value: {{ .Values.config.logLevel | quote }}
- name: ENVIRONMENT
  value: {{ .Values.config.environment | quote }}
- name: NAMESPACE
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
{{- if .Values.postgresql.enabled }}
- name: DB_HOST
  value: {{ .Release.Name }}-postgresql
- name: DB_NAME
  value: {{ .Values.postgresql.auth.database }}
- name: DB_USER
  value: {{ .Values.postgresql.auth.username }}
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Release.Name }}-postgresql
      key: password
{{- end }}
{{- if .Values.redis.enabled }}
- name: REDIS_ADDR
  value: {{ .Release.Name }}-redis-master:6379
{{- if .Values.redis.auth.enabled }}
- name: REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Release.Name }}-redis
      key: redis-password
{{- end }}
{{- end }}
{{- if .Values.kafka.enabled }}
- name: KAFKA_BROKERS
  value: {{ .Release.Name }}-kafka:9092
{{- end }}
{{- if .Values.influxdb.enabled }}
- name: INFLUX_URL
  value: http://{{ .Release.Name }}-influxdb:8086
- name: INFLUX_ORG
  value: {{ .Values.influxdb.auth.org }}
- name: INFLUX_BUCKET
  value: {{ .Values.influxdb.auth.bucket }}
{{- end }}
{{- range .Values.extraEnvVars }}
- name: {{ .name }}
  value: {{ .value | quote }}
{{- end }}
{{- end }}

{{/*
Common volume mounts
*/}}
{{- define "near-rt-ric.commonVolumeMounts" -}}
- name: config
  mountPath: /app/configs
  readOnly: true
- name: tmp
  mountPath: /tmp
{{- if .Values.persistence.enabled }}
- name: data
  mountPath: /app/data
{{- end }}
{{- range .Values.extraVolumeMounts }}
- name: {{ .name }}
  mountPath: {{ .mountPath }}
  {{- if .subPath }}
  subPath: {{ .subPath }}
  {{- end }}
  {{- if .readOnly }}
  readOnly: {{ .readOnly }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Common volumes
*/}}
{{- define "near-rt-ric.commonVolumes" -}}
- name: config
  configMap:
    name: {{ include "near-rt-ric.fullname" . }}-config
- name: tmp
  emptyDir: {}
{{- if .Values.persistence.enabled }}
- name: data
  persistentVolumeClaim:
    claimName: {{ include "near-rt-ric.fullname" . }}-data
{{- end }}
{{- range .Values.extraVolumes }}
- name: {{ .name }}
  {{- if .configMap }}
  configMap:
    name: {{ .configMap.name }}
    {{- if .configMap.items }}
    items:
    {{- range .configMap.items }}
    - key: {{ .key }}
      path: {{ .path }}
    {{- end }}
    {{- end }}
  {{- else if .secret }}
  secret:
    secretName: {{ .secret.secretName }}
    {{- if .secret.items }}
    items:
    {{- range .secret.items }}
    - key: {{ .key }}
      path: {{ .path }}
    {{- end }}
    {{- end }}
  {{- else if .emptyDir }}
  emptyDir: {}
  {{- else if .hostPath }}
  hostPath:
    path: {{ .hostPath.path }}
    {{- if .hostPath.type }}
    type: {{ .hostPath.type }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Resource limits and requests
*/}}
{{- define "near-rt-ric.resources" -}}
{{- if .resources }}
resources:
  {{- if .resources.limits }}
  limits:
    {{- if .resources.limits.cpu }}
    cpu: {{ .resources.limits.cpu }}
    {{- end }}
    {{- if .resources.limits.memory }}
    memory: {{ .resources.limits.memory }}
    {{- end }}
  {{- end }}
  {{- if .resources.requests }}
  requests:
    {{- if .resources.requests.cpu }}
    cpu: {{ .resources.requests.cpu }}
    {{- end }}
    {{- if .resources.requests.memory }}
    memory: {{ .resources.requests.memory }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}