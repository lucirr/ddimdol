{{- define "portal-api.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "portal-api.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "portal-api.labels" -}}
app.kubernetes.io/name: {{ include "portal-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end }}
