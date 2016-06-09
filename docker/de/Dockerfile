FROM alpine

RUN apk add --update postgresql
VOLUME /var/lib/postgresql/data

RUN mkdir -p /usr/bin/chain
COPY cored /usr/bin/chain/cored
COPY startup.sh /usr/bin/chain/startup.sh

RUN mkdir -p /var/lib/chain
COPY schema.sql /var/lib/chain/schema.sql

ENV DB_URL=postgres://postgres:@localhost/core?sslmode=disable
ENTRYPOINT /usr/bin/chain/startup.sh
EXPOSE 8080
