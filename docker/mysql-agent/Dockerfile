FROM mysql/mysql-server:8.0.12

COPY bin/linux_amd64/mysql-agent /

# Reference the UID for the 'mysql' user
USER 27

ENTRYPOINT ["/mysql-agent"]
