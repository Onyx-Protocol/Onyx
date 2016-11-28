FROM alpine:3.4

RUN apk --no-cache add ca-certificates postgresql
VOLUME /var/lib/postgresql/data
VOLUME /var/log/chain

COPY corectl /usr/bin/chain/corectl
COPY cored /usr/bin/chain/cored
COPY startup.sh /usr/bin/chain/startup.sh

ENV DATABASE_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV LOGFILE=/var/log/chain/cored.log
ENTRYPOINT /usr/bin/chain/startup.sh
EXPOSE 1999
