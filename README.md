# Certificate-Manager

Certificate-Manager is a Kubernetes controller that manages the lifecycle of certificates. It is built using Kubebuilder and the Operator SDK.

## Table of Contents

- [Description](#description)
- [Features](#features)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Setting up K3D Cluster (Optional)](#setting-up-k3d-cluster-optional)
  - [Running the Controller Locally](#running-the-controller-locally)
    - [Uninstalling the Controller](#uninstalling-the-controller)
  - [Running on the Cluster via Helm](#running-on-the-cluster-via-helm)
    - [Uninstalling the Helm Chart](#uninstalling-the-helm-chart)
- [How it Works](#how-it-works)
- [Custom Resource Definition](#custom-resource-definition)
- [Uninstalling the Helm Chart](#uninstalling-the-helm-chart)
- [ASCIINEMA Demo](#asciinema-demo)

## Description

Certificate-Manager is a Kubernetes controller that watches for `Certificate` custom resources and creates a self-signed certificate based on the information provided in the custom resource spec. The controller then creates a secret with the certificate and key according to the name provided in the custom resource spec.

## Features

- Create a self-signed certificate
- Create a secret with the generated certificate and key
- Update the certificate and key in the secret when the certificate is updated
- Delete the secret when the certificate is deleted (Optional)
- Reload the deployments using the certificate when the certificate is updated (Optional)
- Rotate the certificate when the certificate is expired (Optional)

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) or [K3D](https://k3d.io) to create a local cluster for testing, or you can use a remote cluster.

**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e., whatever cluster `kubectl cluster-info` shows).

### Prerequisites

- Kubernetes cluster (local or remote)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Helm](https://helm.sh/docs/intro/install/)
- [K3D](https://k3d.io) (optional, for local cluster setup)

### Setting up K3D Cluster (Optional)

- Install K3D:

  ```sh
  wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
  ```

- Create a K3D cluster:

  ```sh
  k3d cluster create tnc --agents=3 --port=443:443@loadbalancer
  ```

Here, `tnc` is short for Three Node Cluster. You can name your cluster whatever you want. `--agents=3` specifies the number of agents/nodes in the cluster. `--port=443:443@loadbalancer` maps the host port 443 to the cluster port 443. This way you can access the cluster using `https://localhost`.

### Running the Controller Locally

>**Note**: The following steps assume you have a Kubernetes cluster running and `kubectl` configured to use it.

- Install the controller:

  ```sh
  make install
  ```

- Run the controller:

  ```sh
  make run
  ```

- Create a Certificate custom resource:

  ```sh
  kubectl apply -f config/samples/certificate_v1_certificate.yaml
  ```

- Check the secret created by the controller:

  ```sh
  kubectl get secret my-certificate-secret -o jsonpath='{.data.tls\.crt}' | base64 -d
  ```

#### Uninstalling the Controller

- To uninstall the controller, run:

  ```sh
  make uninstall
  ```

### Running on the Cluster via Helm

>**Note**: The following steps assume you have a Kubernetes cluster running and `kubectl` configured to use it.

- Create a namespace for the controller:

  ```sh
  kubectl create namespace certificate-manager
  ```

- Install the Helm chart:

  ```sh
  helm install certificate-manager helm/certificate-manager --namespace certificate-manager
  ```

- Create a Certificate custom resource:

  ```sh
  kubectl apply -f examples/certificate.yaml
  ```

- Check the secret created by the controller:

  ```sh
  kubectl get secret my-certificate-secret -o jsonpath='{.data.tls\.crt}' | base64 -d
  ```

#### Uninstalling the Helm Chart

- To uninstall the Helm chart, run:

  ```sh
  helm uninstall certificate-manager --namespace certificate-manager
  ```

## How it Works

This project aims to follow the Kubernetes Operator pattern.

It uses Controllers, which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

The controller watches for changes to the Certificate custom resource and takes the following actions:

1. When a Certificate resource is created, the controller creates a self-signed certificate and stores it in a secret.
1. When a Certificate resource is updated, the controller updates the certificate in the secret.
1. When a Certificate resource is deleted, the controller deletes the secret if the optional PurgeOnDelete field is set to true. Otherwise, the secret is left intact.
1. When a Certificate resource is updated, the controller reloads the deployments using the certificate if the optional ReloadOnChange field is set to true.
1. When a Certificate resource is expired, the controller rotates the certificate if the optional RotateOnExpiry field is set to true.

## Custom Resource Definition

The Certificate custom resource definition is defined in the `api/v1` directory.

```yaml
apiVersion: certs.k8c.io/v1
kind: Certificate
metadata:
  name: my-certificate
  namespace: default
spec:
  # the DNS name for which the certificate should be issued
  dnsName: example.k8c.io
  # the time until the certificate expires
  validity: 360d
  # a reference to the Secret object in which the certificate is stored
  secretRef:
    name: my-certificate-secret
  # optional: purgeOnDelete will delete the secret when the certificate CR is deleted
  purgeOnDelete: false
  # optional: reloadOnChange will reload the deployments using the secret when the certificate is updated
  reloadOnChange: false
  # optional: rotateOnExpiry will rotate the certificate before it expires
  rotateOnExpiry: false
```

## ASCIINEMA Demo

[![asciicast](https://asciinema.org/a/Tm4PiGFtchccYur7rkR3h6Sjv.svg)](https://asciinema.org/a/Tm4PiGFtchccYur7rkR3h6Sjv)
