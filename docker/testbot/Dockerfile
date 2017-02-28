FROM golang:1.8-alpine
ENV CHAIN /go/src/chain
ENV DATABASE_URL postgres://testbot:@localhost/core?sslmode=disable
ENV JAVA_HOME /usr/lib/jvm/java-1.8-openjdk
COPY repositories /etc/apk/repositories
RUN apk --no-cache add git maven@edge openjdk8 postgresql && \
    adduser -S testbot
COPY startup.sh /startup.sh
CMD /startup.sh
USER testbot
