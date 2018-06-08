# Config

## Introduction

Some aspects of the MySQL Operator can be configured via:

1. MySQL Operator Command Line Parameters.
2. MySQL Operator ConfigMap.

When applicable, a commandline parameter will override the equivalent config
map parameter.

Most of the time it should not be neccessary to supply any specific
configuration and the operator will use sensible defaults when required
values are not specified.


### Create a MySQLOperator deployment with volume mounted configuration.

In some cases, however, it may be desirable to configure aspects of the
controller. For example, during development you may wish to use a
different 'mysql-server' or 'mysql-agent' image.

The following Helm chart snippet does just that by configuring a
config map and volume mounting it to the known location:
_/etc/mysql-operator/mysql-operator-config.yaml_

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql-operator-config
  namespace: {{.Values.operator.namespace}}
  labels:
    app: mysql-operator
    release: {{ .Release.Name }}
    chart: {{ .Chart.Name }}-{{ .Chart.Version }}
data:
  mysql-operator-config.yaml: |
    images:
      mysqlServer: mysql/mysql-server
      mysqlAgent: iad.ocir.io/oracle/mysql-agent
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: mysql-operator
  namespace: {{.Values.operator.namespace}}
  labels:
    release: {{ .Release.Name }}
    chart: {{ .Chart.Name }}-{{ .Chart.Version }}
    app: mysql-operator
spec:
  members: 1
  selector:
    matchLabels:
      app: mysql-operator
  template:
    metadata:
      labels:
        app: mysql-operator
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
    spec:
      serviceAccountName: mysql-operator
      imagePullSecrets:
        - name: {{ .Values.docker.pullSecret }}
      volumes:
        - name: mysql-operator-config-volume
          configMap:
            name: mysql-operator-config
      containers:
      - name: mysql-operator-controller
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        image: iad.ocir.io/oracle/mysql-operator:{{ .Values.image.tag }}
        ports:
        - containerPort: 10254
        volumeMounts:
        - name: mysql-operator-config-volume
          mountPath: /etc/mysql-operator
        args:
          - --v=4
{{- if not .Values.operator.global }}
          - --namespace={{- .Values.operator.namespace }}
{{- end }}
```
