FROM centos:6

ENV CHAIN="/go/src/chain" \
    BUNDLE_APP_CONFIG="/usr/local/bundle" \
    GEM_HOME="/usr/local/bundle" \
    GOPATH="/go" \
    GOROOT="/usr/local/go" \
    JAVA_HOME="/opt/jdk1.7.0_79" \
    JRE_HOME="/opt/jdk1.7.0_79/jre" \
    MAVEN_HOME="/usr/share/maven" \
    PATH="$PATH:/usr/local/bundle/bin:/go/bin:/usr/local/go/bin:/opt/jdk1.7.0_79/bin:/opt/jdk1.7.0_79/jre/bin"

ADD bin /usr/bin

RUN yum -y update \
    && yum install -y autoconf automake bash curl gcc gcc-c++ git libtool make openssl-devel ruby wget unzip \
    && install-go && rm -f /usr/bin/install-go \
    && install-node && rm -f /usr/bin/install-node \
    && install-java && rm -f /usr/bin/install-java \
    && install-maven && rm -f /usr/bin/install-maven \
    && install-ruby && rm -f /usr/bin/install-ruby \
    && install-protobuf && rm -f /usr/bin/install-protobuf \
    && install-gas && rm -f /usr/bin/install-gas \
    && go get -u google.golang.org/grpc \
    && yum remove -y autoconf automake libtool gcc-c++ unzip \
    && yum clean all

