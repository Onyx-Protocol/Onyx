FROM alpine

VOLUME /var/lib/postgresql/data
VOLUME /var/log/chain

RUN apk --no-cache add openjdk8-jre postgresql wget

COPY startup.sh /usr/bin/chain/startup.sh
COPY tests.sh /usr/bin/chain/tests.sh
COPY clear-blockchain.sql /var/lib/chain/clear-blockchain.sql
COPY schema.sql /var/lib/chain/schema.sql
COPY corectl /usr/bin/chain/corectl
COPY cored /usr/bin/chain/cored

RUN wget -O /usr/bin/chain/chain-core-qa.jar https://s3.amazonaws.com/chain-qa/chain-core-qa.jar

ENV PGDATA /var/lib/postgresql/data
ENV DB_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV DB1_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV DB2_URL=postgres://postgres:@localhost/core2?sslmode=disable
ENV DB3_URL=postgres://postgres:@localhost/core3?sslmode=disable
ENV CORE1_URL="http://localhost:8080"
ENV CORE2_URL="http://localhost:8081"
ENV CORE3_URL="http://localhost:8082"
ENV CORE2_LISTEN=":8081"
ENV CORE3_LISTEN=":8082"

ENTRYPOINT /usr/bin/chain/startup.sh
