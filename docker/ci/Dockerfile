FROM golang:1.7.3-alpine

RUN apk --no-cache add bash clang git g++ make nodejs openjdk8 python perl \
    ruby ruby-bigdecimal ruby-bundler ruby-io-console ruby-irb ruby-json \
    && MAVEN_VERSION=3.3.9 \
    && cd /usr/share \
    && wget -q http://archive.apache.org/dist/maven/maven-3/$MAVEN_VERSION/binaries/apache-maven-$MAVEN_VERSION-bin.tar.gz -O - | tar xzf - \
    && mv /usr/share/apache-maven-$MAVEN_VERSION /usr/share/maven \
    && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
    && git clone https://github.com/GoASTScanner/gas $GOPATH/src/github.com/GoASTScanner/gas \
    && cd $GOPATH/src/github.com/GoASTScanner/gas \
    && git reset --hard d30c5cde3613e9ba0129febda849e4d4df1d57cd \
    && go install github.com/GoASTScanner/gas

ADD bin /usr/bin
ENV CHAIN $GOPATH/src/chain
ENV JAVA_HOME /usr/lib/jvm/java-1.8-openjdk
ENV MAVEN_HOME /usr/share/maven
