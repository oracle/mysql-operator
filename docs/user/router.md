# Using the MySQL Router

When connecting your applications to MySQL clusters created by the MySQL Operator, we recommend that you use the [MySQL Router][1].

> The MySQL Router is part of InnoDB cluster, and is lightweight middleware that provides transparent routing between your application and back-end MySQL Servers. It can be used for a wide variety of use cases, such as providing high availability and scalability by effectively routing database traffic to appropriate back-end MySQL Servers. The pluggable architecture also enables developers to extend MySQL Router for custom use cases.

## Configuration

Typically the MySQL Router is deployed [alongside your application][2].

![MySQL Router][3]

## Demo

### Create a MySQL Cluster

We'll use WordPress as an example of how you might setup the MySQL Router for a a real application. The first thing we want to do is create our MySQL Cluster using the operator.

```yaml
apiVersion: "mysql.oracle.com/v1"
kind: MySQLCluster
metadata:
  name: mysql-wordpress
spec:
  replicas: 3
  secretRef:
    name: wordpress-mysql-root-password
```

Create this with

```
kubectl apply -f examples/demo/wordpress-router/wordpress-database.yaml
```

### Create an application + router

We'll create a standard WordPress application but connect directly to the MySQL Router which runs as a sidecar container alongside inside your Wordpress pod.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: wordpress-router
  labels:
    app: wordpress-router
spec:
  ports:
    - port: 80
  selector:
    app: wordpress-router
  type: LoadBalancer
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: wordpress-router
  labels:
    app: wordpress-router
spec:
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: wordpress-router
    spec:
      containers:
      - name: mysqlrouter
        image: pulsepointinc/mysql-router:2.1.3
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: wordpress-mysql-root-password
              key: password
        command:
        - "/bin/bash"
        - "-cx"
        - |
          echo "Bootstraping the router"
          echo $MYSQL_ROOT_PASSWORD | mysqlrouter --bootstrap mysql-wordpress-0.mysql-wordpress:3306 --user=root

          echo "Updating resolv.conf"
          search=$(grep ^search /etc/resolv.conf)
          echo "$search mysql-wordpress.default.svc.cluster.local" >> /etc/resolv.conf

          echo "Running the router"
          mysqlrouter --user=root
      - name: wordpress
        image: wordpress:4.8.0-apache
        env:
        - name: WORDPRESS_DB_HOST
          value: 127.0.0.1:6446
        - name: WORDPRESS_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: wordpress-mysql-root-password
              key: password
        ports:
        - containerPort: 80
```

Create this with

```
kubectl apply -f examples/demo/wordpress-router/wordpress-deployment.yaml
```

### Verify

If you used a LoadBalancer service to expose your WordPress deployment, you can visit the load balancer IP address and verify your deployment successfully connects to your MySQL cluster.

[1]: https://dev.mysql.com/doc/mysql-router/2.1/en/
[2]: https://dev.mysql.com/doc/mysql-router/2.1/en/mysql-router-general-using-deploying.html
[3]: https://dev.mysql.com/doc/mysql-router/2.1/en/images/mysql-router-positioning.png

