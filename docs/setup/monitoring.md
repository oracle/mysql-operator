# Monitoring

The MySQL Operator provides basic operational metrics via Prometheus that can be used to monitor the state of
both the Operator and individual clusters.

## Getting started

First we need to install Prometheus and ensure that we are scraping metrics from the MySQL Operator.

```
# Install Prometheus
helm install stable/prometheus \
  --set alertmanager.persistentVolume.enabled=false \
  --set server.persistentVolume.enabled=false \
  --set rbac.create=true
```

By default, the Helm installation of Prometheus is setup to dynamically scrape pods and services annotated with prometheus.io/scrape=true

## View metrics in Prometheus

You can view your metrics by running the following:

```
export POD_NAME=$(kubectl get pods --namespace default -l "app=prometheus,component=server" \
    -o jsonpath="{.items[0].metadata.name}")

kubectl --namespace default port-forward $POD_NAME 9090
```

## Operator Metrics

The following custom metrics are exported by the MySQL Operator

#### Clusters

* Clusters created
* Clusters deleted
* Clusters total
