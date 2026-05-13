{{- define "edge-agent.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "edge-agent.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edge-agent.labels" -}}
app.kubernetes.io/name: {{ include "edge-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
