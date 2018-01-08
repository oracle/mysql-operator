# Upgrade

It is possible to upgrade/downgrade the mysql-operator and mysql-agent containers by updating the image to the required version.

```bash
kubectl patch \
    -n mysql-operator \
    deployment/mysql-operator \
    -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"mysql-operator-controller\",\"image\":\"registry.oracledx.com/skeppare/mysql-operator:${OPERATOR_VERSION}\"}]}}}}"
```


