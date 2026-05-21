{{/*
Chart name, truncated to 63 chars (DNS-1123 label limit).
*/}}
{{- define "pingapp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name. If `release name` already contains the chart name,
use the release name alone; otherwise prefix it.
*/}}
{{- define "pingapp.fullname" -}}
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
Chart label — `<name>-<version>`, with "+" replaced (chart labels can't contain it).
*/}}
{{- define "pingapp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels — attached to every resource the chart owns.
*/}}
{{- define "pingapp.labels" -}}
helm.sh/chart: {{ include "pingapp.chart" . }}
{{ include "pingapp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: pingapp
{{- end }}

{{/*
Selector labels — the stable subset used by Service selectors and Deployment
matchLabels. Must NOT include version/chart so a rolling upgrade doesn't churn
the selector.
*/}}
{{- define "pingapp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pingapp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
