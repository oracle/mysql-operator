FROM mysql/mysql-server:5.7.20-1.1.2

COPY bin/linux_amd64/mysql-agent /

USER mysql

ENTRYPOINT ["/mysql-agent"]
