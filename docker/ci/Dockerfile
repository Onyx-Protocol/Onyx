FROM centos:6

ENV GOLANG_VERSION 1.7.5
ENV GOLANG_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_SHA256 2e4dd6c44f0693bef4e7b46cc701513d74c3cc44f2419bf519d7868b12931ac3
ENV GOPATH /go
ENV JAVA_HOME /opt/jdk1.7.0_79
ENV JRE_HOME /opt/jdk1.7.0_79/jre

RUN yum install -y bash curl gcc gcc-c++ git go make openssl \
    && curl -fsSL -o golang.tar.gz "$GOLANG_URL" \
    && echo "$GOLANG_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz \
    && mkdir -p "$GOPATH/{src,bin}" \
    && chmod -R 777 "$GOPATH" \
    && curl -sL https://rpm.nodesource.com/setup_4.x | bash - \
    && yum install -y nodejs \
    && curl -L0 -o jdk-7u79-linux-x64.tar.gz -H \
    "Cookie: gpw_e24=http%3A%2F%2Fwww.oracle.com%2F; oraclelicense=accept-securebackup-cookie" \
    http://download.oracle.com/otn-pub/java/jdk/7u79-b15/jdk-7u79-linux-x64.tar.gz \
    && tar -C /opt -xzf jdk-7u79-linux-x64.tar.gz \
    && cd /opt/jdk1.7.0_79/ \
    && alternatives --install /usr/bin/java java /opt/jdk1.7.0_79/bin/java 2 \
    && alternatives --auto java \
    && alternatives --install /usr/bin/jar jar /opt/jdk1.7.0_79/bin/jar 2 \
    && alternatives --install /usr/bin/javac javac /opt/jdk1.7.0_79/bin/javac 2 \
    && alternatives --install /usr/bin/javaws javaws /opt/jdk1.7.0_79/bin/javaws 2 \
    && alternatives --set jar /opt/jdk1.7.0_79/bin/jar \
    && alternatives --set javac /opt/jdk1.7.0_79/bin/javac \
    && PATH=$PATH:/opt/jdk1.7.0_79/bin:/opt/jdk1.7.0_79/jre/bin

# RUN yum install -y bash clang git g++ make nodejs openjdk8 python perl \
#     ruby ruby-bigdecimal ruby-bundler ruby-io-console ruby-irb ruby-json \
#     openssl ca-certificates

# RUN apk --no-cache add bash clang git g++ make nodejs openjdk8 python perl \
#     ruby ruby-bigdecimal ruby-bundler ruby-io-console ruby-irb ruby-json openssl ca-certificates \
#     && echo "http://dl-cdn.alpinelinux.org/alpine/v3.5/main" >> /etc/apk/repositories \
#     && apk --no-cache add "protobuf>3.1.0" \
#     && go get -u github.com/golang/protobuf/proto \
#     && go get -u github.com/golang/protobuf/protoc-gen-go \
#     && go get -u google.golang.org/grpc \
# # TODO: remove third party glibc (as well as openssl and ca-certificates) package when protobuf works with musl (next release)
#     && wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub \
#     && wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.23-r3/glibc-2.23-r3.apk \
#     && apk add glibc-2.23-r3.apk \
#     && MAVEN_VERSION=3.3.9 \
#     && cd /usr/share \
#     && wget -q http://archive.apache.org/dist/maven/maven-3/$MAVEN_VERSION/binaries/apache-maven-$MAVEN_VERSION-bin.tar.gz -O - | tar xzf - \
#     && mv /usr/share/apache-maven-$MAVEN_VERSION /usr/share/maven \
#     && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
#     && git clone https://github.com/GoASTScanner/gas $(go env GOPATH)/src/github.com/GoASTScanner/gas \
#     && cd $(go env GOPATH)/src/github.com/GoASTScanner/gas \
#     && git reset --hard d30c5cde3613e9ba0129febda849e4d4df1d57cd \
#     && go install github.com/GoASTScanner/gas
#     && curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly-2017-02-25
#     && ~/.cargo/bin/cargo install rustfmt
#     && ~/.cargo/bin/cargo install clippy
#     && ~/.cargo/bin/cargo install cargo-audit

# ADD bin /usr/bin
# ENV CHAIN ${GOPATH:-$HOME/go}/src/chain
# ENV JAVA_HOME /usr/lib/jvm/java-1.8-openjdk
# ENV MAVEN_HOME /usr/share/maven
