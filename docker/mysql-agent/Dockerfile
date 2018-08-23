FROM mysql/mysql-server:8.0.12

COPY bin/mysql-agent /

USER mysql

ENTRYPOINT ["/mysql-agent"]
