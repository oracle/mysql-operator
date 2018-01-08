# Wordpress + MySQL Operator

An example application that makes use of the MySQL Operator.

Create the MySQL password

```
kubectl create secret generic wordpress-mysql-root-password --from-literal=password=password
```

Create the database

```
kubectl apply -f wordpress-database.yaml
```

Create the Wordpress deployment

```
kubectl apply -f wordpress-deployment.yaml
```
