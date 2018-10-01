# Enterprise edition tutorial
This tutorial will explain how to create a mysqlcluster that runs the enterprise version of mysql.

## Prerequisites

- The mysql-operator repository checked out locally.
- Access to a Docker registry that contains the enterprise version of mysql.

## 01 - Create the Operator
You will need to create the following:

1. Custom resources 
2. RBAC configuration <sup>*</sup>
3. The Operator 
4. The Agent ServiceAccount & RoleBinding

The creation of these resources can be achieved by following the [introductory tutorial][1]; return here before creating a MySQL Cluster.

## 02 - Create a secret with registry credentials
To be able to pull the mysql enterprise edition from docker it is necessary to provide credentials, these credentials must be supplied in the form of a Kubernetes secret.

- The name of the secret `myregistrykey` must match the name in the `imagepullsecrets` which we will specify in the cluster config in step 03.
- The secret must be created in the same namespace as the MySQL Cluster which we will make in step 03. It must also be in the same namespace as the RBAC permissions created in step 01.
- If you are pulling the mysql enterprise image from a different registry then the secret must contain the relevant credentials for that registry.

>For alternative ways to create Kubernetes secretes see their documentation on [creating secrets from docker configs](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod) or [creating secrets manually](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config).

Enter your credentials into the following command and execute it to create a Kubernetes secret that will enable pulling images from the Docker store. add  the `-n` flag to specify a namespace if you do not want to use the default namespace. 
```
kubectl create secret docker-registry myregistrykey \
--docker-server=https://index.docker.io/v1/ \
--docker-username= \
--docker-password= \
--docker-email=
```
## 03 - Create your MySQL Cluster
Finally, create your MySQL Cluster with the required specifications entered under `spec:` 

- The mysqlServer field should be the path to a registry containing the enterprise edition of MySQL.
- The imagePullSecret: name: Should be the name of a Kubernetes secret in the same namespace that contains your credentials for the docker registry.
- The version to be used must be specified, without this, a default version is used which is **not** guaranteed to match an available image of MySQL Enterprise.
- The namespace of the cluster  must match the namespace of the secret we created in step 02.
```
kubectl apply -f examples/cluster/cluster-enterprise-version.yaml
```
### Check that it is running
You can now run the following command to access the sql prompt in your MySQL Cluster, just replace `<NAMESPACE>` with the namespace you created your cluster in.
```
sh hack/mysql.sh <NAMESPACE>/mysql-0
```

><sup>*</sup>If you run into issues when creating RBAC roles see [Access controls](https://docs.cloud.oracle.com/iaas/Content/ContEng/Concepts/contengabouta]ccesscontrol.htm?) for more information.

[1]: docs/tutorial.md