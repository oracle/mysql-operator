# Using the MySQL Router

When connecting your applications to MySQL clusters created by the MySQL
Operator, we recommend that you use the [MySQL Router][1].

> The MySQL Router is part of InnoDB cluster, and is lightweight middleware that
> provides transparent routing between your application and back-end MySQL
> Servers. It can be used for a wide variety of use cases, such as providing
> high availability and scalability by effectively routing database traffic to
> appropriate back-end MySQL Servers. The pluggable architecture also enables
> developers to extend MySQL Router for custom use cases.

## Configuration

Typically the MySQL Router is deployed [alongside your application][2].

![MySQL Router][3]

## Demo

### Create a MySQL Cluster

We'll use WordPress as an example of how you might setup the MySQL Router for a
real application. The first thing we want to do is create our MySQL Cluster
using the operator.

Create this with:

```
kubectl apply -f examples/demo/wordpress-router/wordpress-database.yaml
```

### Create an application + router

We'll create a standard WordPress application but connect directly to the MySQL
Router which runs as a sidecar container alongside inside your Wordpress Pod.

Create this with:

```
kubectl apply -f examples/demo/wordpress-router/wordpress-deployment.yaml
```

### Verify

If you used a Service type=LoadBalancer to expose your WordPress deployment,
you can visit the load balancer IP address and verify your deployment
successfully connects to your MySQL cluster.

[1]: https://dev.mysql.com/doc/mysql-router/2.1/en/
[2]: https://dev.mysql.com/doc/mysql-router/2.1/en/mysql-router-general-using-deploying.html
[3]: https://dev.mysql.com/doc/mysql-router/2.1/en/images/mysql-router-positioning.png

