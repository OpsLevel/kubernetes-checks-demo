# Kubernetes Checks Demo

This example repository is a Kubernetes agent that watches resources in your cluster and publishes their yaml to an OpsLevel payload check. This runs in your k8s cluster.

The intent of this repo was created as an example of how to get kubernetes data into OpsLevel payload checks as described in the blog post [Validating Kubernetes Best Practices](https://www.opslevel.com/blog/validating-kubernetes-best-practices/)

### Getting Started

```
Opslevel Example Kubernetes Agent

Usage:
  opslevel-agent [command]

Available Commands:
  config      Commands for working with the opslevel configuration
  help        Help about any command
  run         

Flags:
  -c, --config string       (default "./opslevel.yaml")
  -h, --help               help for opslevel-agent
      --logFormat string   overrides environment variable 'OL_LOGFORMAT' (options ["JSON", "TEXT"]) (default "JSON")
      --logLevel string    overrides environment variable 'OL_LOGLEVEL' (options ["ERROR", "WARN", "INFO", "DEBUG"]) (default "INFO")
```

To run the agent

```
opslevel-agent run -c ./opslevel.yaml
```

To view a sample configuration file

```
opslevel-agent config sample
```

### Integration

Build the docker container and publish it to your artifact repository.

Configure an OpsLevel Payload Check https://www.opslevel.com/docs/checks/payload-checks/

The last thing the agent needs is a configuration file and here is a sample

```
integrationid: "" #opslevel payload integration id
payloadcheckid: "" #payload check identifier / check reference id
resync: 3600 #1hr
deployments: true
statefulsets: false
daemonsets: false
jobs: false
cronjobs: false
services: false
ingress: false
configmaps: false
secrets: false
```

you can test load the configuration file like this

```
docker run -v $(pwd)/opslevel.yaml:/opslevel.yaml <docker image you built> -c /opslevel.yaml config view  
```

### Deploy to Kubernetes

Configure the ConfigMap and Deploy it to your kubernetes cluster.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: opslevel-payload-check
  namespace: default
data:
  opslevel.yaml: |-
    integrationId: ""
    payloadCheckId: ""
    resync: 3600
    deployments: true
    statefulsets: false
    daemonsets: false
    jobs: false
    cronjobs: false
    services: false
    ingress: false
    configmaps: false
    secrets: false
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opslevel-payload-check
  namespace: default
  labels:
    app: opslevel-payload-check
spec:
  replicas: 1
  selector:
    matchLabels:
      app: opslevel-payload-check
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
  template:
    metadata:
      labels:
        app: opslevel-payload-check
    spec:
      containers:
        - name: web
          image: <replace me with location where you published the image>
          imagePullPolicy: Always
          args:
            - run
            - --config=/etc/agent/config/opslevel.yaml
            - 2>&1
          volumeMounts:
            - name: opslevel-payload-check
              mountPath: /etc/agent/config
              readOnly: true
      volumes:
        - name: opslevel-payload-check
          configMap:
            name: opslevel-payload-check

```
