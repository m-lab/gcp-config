# Google Cloud Deployment Manager

This directory contains YAML spec files suitable for creating [Google Cloud
Deployment Manager](https://cloud.google.com/deployment-manager) objects in
GCP. Each YAML file should contain only a single resource type (e.g., service
accounts, logs-based metrics, etc.)

## Deployment

Deployment of these resources can be done with something like:

```
gcloud deployment-manager deployments create <deployment-name> \
  --config <yaml-file> --project <project>
```

For example:

```
gcloud deployment-manager deployments create logs-based-metrics \
  --config logs-based-metrics.yaml --project mlab-sandbox
```
