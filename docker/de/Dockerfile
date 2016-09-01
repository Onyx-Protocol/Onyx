FROM alpine

RUN apk --no-cache add postgresql
VOLUME /var/lib/postgresql/data
VOLUME /var/log/chain

COPY cored /usr/bin/chain/cored
COPY schema.sql /var/lib/chain/schema.sql
COPY startup.sh /usr/bin/chain/startup.sh

ENV RUNTIME_PACKAGES openssl-dev tzdata
RUN apk --no-cache add $RUNTIME_PACKAGES

ENV CHAIN_API_URL=http://localhost:8080
ENV DATABASE_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV LOGFILE=/var/log/chain/cored.log
ENTRYPOINT /usr/bin/chain/startup.sh
EXPOSE 8080
