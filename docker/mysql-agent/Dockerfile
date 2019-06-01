FROM mysql/mysql-server:8.0.12

COPY bin/linux_amd64/mysql-agent /

# Set default value for 'MYSQL_USER', which is overridden with the UID at build time
ARG MYSQL_USER=mysql
USER ${MYSQL_USER}

ENTRYPOINT ["/mysql-agent"]
