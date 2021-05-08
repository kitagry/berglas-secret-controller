## Berglas Secret Controller

[![Test](https://github.com/kitagry/berglas-secret-controller/actions/workflows/test.yaml/badge.svg)](https://github.com/kitagry/berglas-secret-controller/actions/workflows/test.yaml)
[![GitHub release](https://img.shields.io/github/v/tag/kitagry/berglas-secret-controller.svg?sort=semver)](https://github.com/kitagry/berglas-secret-controller/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Coverage Status](https://coveralls.io/repos/github/kitagry/berglas-secret-controller/badge.svg?branch=main)](https://coveralls.io/github/kitagry/berglas-secret-controller?branch=main)

### What is this?

This is CustomController of Kubernetes for berglas secret.
You can use berglas in Kubernetes to use [Custom Webhook](https://github.com/GoogleCloudPlatform/berglas/tree/main/examples/kubernetes).
But, this is a bit invconvinience, because you should grant all ServiceAccount permission of Deployment.
So, you should set ServiceAccount every time you create new service.
This Berglas Secret Controller can change all berglas secret once you install this.

### Usage

TODO

#### Use in local

1. build this repository

```bash
git clone https://github.com/kitagry/berglas-secret-controller
cd berglas-secret-controller
make
```

2. Create CRD in Kubernetes

```bash
make install
```

3. Run CustomController

```bash
make run
```

4. Create Custom Resource

Open new terminal window.

```bash
# Write ./config/samples/batch_v1alpha1_berglassecret.yaml by your favorite editor.
kubectl apply -f ./config/samples/batch_v1alpha1_berglassecret.yaml
```

5. Check the secret

```
kubectl get secret
kubectl describe secret <BeglasSecret name>
```
