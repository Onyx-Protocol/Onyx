FROM alpine

RUN apk --no-cache add postgresql ca-certificates
VOLUME /var/lib/postgresql/data
VOLUME /var/log/chain

COPY corectl /usr/bin/chain/corectl
COPY cored /usr/bin/chain/cored
COPY schema.sql /usr/lib/chain/schema.sql
COPY startup.sh /usr/bin/chain/startup.sh

ENV DATABASE_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV LOGFILE=/var/log/chain/cored.log
ENTRYPOINT /usr/bin/chain/startup.sh
EXPOSE 1999
