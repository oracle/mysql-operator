# Enterprise edition tutorial
This tutorial will explain how to create a mysqlcluster that runs the enterprise version of mysql.

## Prerequisites
- A Kubernetes Cluster running on Kubernetes 1.7.0+.
- The mysql-operator repository checked out locally.
- Access to a Docker registry that contains the enterprise version of mysql.

##Create the Operator
This file bundles the creation of various resources:

1. Custom resources 
2. RBAC configuration <sup>*</sup>
3. The Operator 
4. The Agent

Section 3 of the file pulls the sql enterprise image from the docker store: `store/oracle/mysql-enterprise-server`. If you wish to pull the image from somewhere else you will need to swap out this address.
```
kubectl apply -f examples/example-enterprise-deployment.yaml
```

##Create a secret with registry credentials
To be able to pull the mysql enterprise edition from docker it is necessary to provide credentials, these credentials must be supplied in the form of a Kubernetes secret.

- The name of the secret `myregistrykey` must match the name in the `imagepullsecrets` which is found in Section 3 of the `example-enterprise-deployment.yaml`.
- The secret must be created in the same namespace as the mysqlcluster which we will make in the next step.
- If you are pulling the mysql enterprise image from a different registry then the secret must contain the relevant credentials for that registry.

>For alternative ways to create Kubernetes secretes see their documentation on [creating secrets from docker configs](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod) or [creating secrets manually](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config).

Enter your credentials into the following command and execute it to create a Kubernetes secret that will enable pulling images from the Docker store.
```
kubectl create secret docker-registry myregistrykey \
--docker-server=https://index.docker.io/v1/ \
--docker-username= \
--docker-password= \
--docker-email=
```
##Create your mysqlcluster
Finally, create the mysqlcluster. 

- The version to be used has been specified in the file. Without this a default version is used which is **not** guaranteed to match an available image of mysql enterprise.
- The lowest version supported by the mysql operator is **8.0.11**
- The namespace of the cluster  must match the namespace of the secret we created in the previous step. This file omits namespace in the metadata so the cluster will be created in the default namespace.
```
kubectl apply -f examples/cluster/cluster-with-3-members-enterprise-version.yaml
```
You can now run the following command to see the newly created mysql cluster
```
kubectl describe mysqlcluster mysql
```

## Clean up

To remove the mysqlcluster and each of the components created in this tutorial, execute the following:
```
kubectl delete -f examples/cluster/cluster-with-3-members-enterprise-version.yaml
kubectl delete secret myregistrykey
kubectl delete -f examples/example-enterprise-deployment.yaml 
```

><sup>*</sup>If you run into issues when creating RBAC roles see [Access controls](https://docs.cloud.oracle.com/iaas/Content/ContEng/Concepts/contengabouta]ccesscontrol.htm?) for more information.