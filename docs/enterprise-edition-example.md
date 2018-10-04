# Enterprise edition tutorial
This tutorial will explain how to create a MySQL cluster that runs the enterprise version of MySQL.

## Prerequisites

- The MySQL operator repository checked out locally.
- Access to a Docker registry that contains the enterprise version of MySQL.

## 01 - Create the Operator
You will need to create the following:

1. Custom resources 
2. RBAC configuration <sup>*</sup>
3. The Operator 
4. The Agent ServiceAccount & RoleBinding

The creation of these resources can be achieved by following the [introductory tutorial][1]; return here before creating a MySQL cluster.

## 02 - Create a secret with registry credentials
To be able to pull the MySQL Enterprise Edition from Docker it is necessary to provide credentials, these credentials must be supplied in the form of a Kubernetes secret.

- Remember the name of the secret *myregistrykey* as this will need to be used in step 03 when creating the cluster. 
- If you are pulling the MySQL Enterprise image from a different registry than the one in the example then the secret must contain the relevant credentials for that registry.

>For alternative ways to create Kubernetes secrets see their documentation on [creating secrets from Docker configs](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod) or [creating secrets manually](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config).

Enter your credentials into the following command and execute it to create a Kubernetes secret that will enable pulling images from the Docker store.
```
kubectl create secret docker-registry myregistrykey \
--docker-server=https://index.docker.io/v1/ \
--docker-username= \
--docker-password= \
--docker-email=
```
## 03 - Create your MySQL Cluster
Finally, create your MySQL Cluster with the required specifications entered under `spec:` 

- The `repository:` field should be the path to a Docker registry containing the enterprise edition of MySQL. If this is omitted, the default is taken from the MySQL operator field `defaultMysqlServer:` which you can also specify.
- The `imagePullSecrets`: field allows you to specify a list of Kubernetes secret names. These secret(s) should contain your credentials for the Docker registry.
- The version to be used should be specified, without this, a default version is used which is **not** guaranteed to match an available image of MySQL Enterprise.
- The namespace of the cluster  must match the namespace of the RBAC permissions created in step 01.
```
kubectl apply -f examples/cluster/cluster-enterprise-version.yaml
```
### Check that it is running
You can now run the following command to access the SQL prompt in your MySQL Cluster, just replace `<NAMESPACE>` with the namespace you created your cluster in.
```
sh hack/mysql.sh <NAMESPACE>/mysql-0
```

><sup>*</sup>If you run into issues when creating RBAC roles see [Access controls](https://docs.cloud.oracle.com/iaas/Content/ContEng/Concepts/contengabouta]ccesscontrol.htm?) for more information.

[1]: docs/tutorial.md
