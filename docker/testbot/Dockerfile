FROM golang:1.7.1-alpine
ENV CHAIN /go/src/chain
ENV DATABASE_URL postgres://postgres:@localhost/core?sslmode=disable
ENV PGDATA /var/lib/postgresql/data
VOLUME $CHAIN
VOLUME $PGDATA
VOLUME /usr/bin/chain
VOLUME /var/log/chain
COPY repositories /etc/apk/repositories
RUN apk --no-cache add maven@edge openjdk8 postgresql wget
ENTRYPOINT $CHAIN/docker/testbot/startup.sh
