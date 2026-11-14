{{/*
Renders a label or annotation: the value set in values.yaml, as a string, or the default from config/.
Usage: include "project-v4-with-plugins.valueOrDefault" (list <labels or annotations from the values> "<key>" "<default>")
*/}}
{{- define "project-v4-with-plugins.valueOrDefault" -}}
{{- $value := index (index . 0 | default dict) (index . 1) }}
{{- if kindIs "invalid" $value }}
{{- index . 2 }}
{{- else }}
{{- kindIs "string" $value | ternary $value (toJson $value) | toJson }}
{{- end }}
{{- end }}

{{/*
Renders labels or annotations set in values.yaml. Each value is a string, and a null value is left out.
Usage: include "project-v4-with-plugins.stringMap" <labels or annotations from the values>
*/}}
{{- define "project-v4-with-plugins.stringMap" -}}
{{- $strings := dict }}
{{- range $key, $value := . }}
{{- if not (kindIs "invalid" $value) }}
{{- $_ := set $strings $key (kindIs "string" $value | ternary $value (toJson $value)) }}
{{- end }}
{{- end }}
{{- if $strings }}
{{- toYaml $strings }}
{{- end }}
{{- end }}
