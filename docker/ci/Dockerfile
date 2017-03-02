FROM golang:1.7.4-alpine

RUN apk --no-cache add bash clang git g++ make nodejs openjdk8 python perl \
    ruby ruby-bigdecimal ruby-bundler ruby-io-console ruby-irb ruby-json openssl ca-certificates \
    && echo "http://dl-cdn.alpinelinux.org/alpine/v3.5/main" >> /etc/apk/repositories \
    && apk --no-cache add "protobuf>3.1.0" \
    && go get -u github.com/golang/protobuf/proto \
    && go get -u github.com/golang/protobuf/protoc-gen-go \
    && go get -u google.golang.org/grpc \
# TODO: remove third party glibc (as well as openssl and ca-certificates) package when protobuf works with musl (next release)
    && wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub \
    && wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.23-r3/glibc-2.23-r3.apk \
    && apk add glibc-2.23-r3.apk \
    && MAVEN_VERSION=3.3.9 \
    && cd /usr/share \
    && wget -q http://archive.apache.org/dist/maven/maven-3/$MAVEN_VERSION/binaries/apache-maven-$MAVEN_VERSION-bin.tar.gz -O - | tar xzf - \
    && mv /usr/share/apache-maven-$MAVEN_VERSION /usr/share/maven \
    && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
    && git clone https://github.com/GoASTScanner/gas $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && cd $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && git reset --hard d30c5cde3613e9ba0129febda849e4d4df1d57cd \
    && go install github.com/GoASTScanner/gas

ADD bin /usr/bin
ENV CHAIN ${GOPATH:-$HOME/go}/src/chain
ENV JAVA_HOME /usr/lib/jvm/java-1.8-openjdk
ENV MAVEN_HOME /usr/share/maven
