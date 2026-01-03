# Form3 Kubernetes Operator
Kubernetes Operator for managing the lifecycle of a [Form3](https://form3.tech) Account.

## Prerequisites
- Go version v1.24.0+
- Docker version 17.03+.
- Access to a Kind cluster.
- Kubectl version v1.11.3+.

```sh
chmod +x run.sh
```

```sh
./run.sh
```

### Implementation

**Testing the Operator**
```sh
make test
```

**Deploy Kubernetes Manifests To A Kind Cluster, this provisions a Fake Form3 Account API**
```sh
cd manifests
kubectl apply -f accountapi.yaml
kubectl apply -f postgresql.yaml
kubectl apply -f vault.yaml
```

**Install the Operator**
```sh
export USERNAME=tomiwa97
make docker-build docker-push IMG=docker.io/tomiwa97/form3-operator:v1.0.0
kind load docker.io/tomiwa97/form3-operator:v1.0.0
make deploy IMG=docker.io/tomiwa97/memcached-operator:v1.0.0
```

**Fetching Operator CRD, Deployments, Pods, ClusterRoles, ClusterRolesBindings, Roles, RoleBindings**
```sh
kubectl get crds
kubectl get crds
```

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```
