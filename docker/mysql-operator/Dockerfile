FROM oraclelinux:7.3

COPY bin/linux_amd64/mysql-operator /

# Add a non-root user and reference the UID
RUN adduser -rU -u 1000 mysql-operator
USER 1000

ENTRYPOINT ["/mysql-operator"]
