FROM alpine

RUN apk --no-cache add postgresql
VOLUME /var/lib/postgresql/data

COPY cored /usr/bin/chain/cored
COPY corectl /usr/bin/chain/corectl
COPY dashboard /srv/chain/dashboard
COPY schema.sql /var/lib/chain/schema.sql
COPY startup.sh /usr/bin/chain/startup.sh

ENV BUILD_PACKAGES build-base git libffi-dev
ENV RUNTIME_PACKAGES ruby ruby-bigdecimal ruby-bundler ruby-dev ruby-io-console ruby-irb openssl-dev tzdata
RUN apk --no-cache add $BUILD_PACKAGES $RUNTIME_PACKAGES \
&& bundle install --gemfile=/srv/chain/dashboard/Gemfile \
&& apk del $BUILD_PACKAGES

ENV DB_URL=postgres://postgres:@localhost/core?sslmode=disable
ENV CHAIN_API_URL=http://localhost:8080
ENTRYPOINT /usr/bin/chain/startup.sh
EXPOSE 8080 8081
