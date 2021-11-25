{{- define "args" -}}
- -log.level=INFO
- -sync.address={{ .Values.sync.address }}
- -redis.enabled
- -redis.address={{ tpl .Values.redis.endpoint . }}
{{ if .Values.redis.password }}
- -redis.password={{ .Values.redis.password }}
{{ end }}
{{ if .Values.redis.useTLS }}
- -redis.tls
{{ end }}
{{ if .Values.redis.useTLSInsecure }}
- -redis.tls.insecure
{{ end }}
- -ssl.crt=/certs/CA.crt
- -ssl.key=/certs/CA.key
{{- end -}}

{{- define "data-volume" -}}
{{ if .Values.dataVolume.enabled }}
{{ toYaml .Values.dataVolume.spec}}
{{ else }}
emptyDir: {}
{{ end }}
{{- end -}}