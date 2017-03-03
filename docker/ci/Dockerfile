FROM centos:6

ENV CHAIN="/go/src/chain" \
    BUNDLE_APP_CONFIG="/usr/local/bundle" \
    GEM_HOME="/usr/local/bundle" \
    GOLANG_VERSION="1.8" \
    GOLANG_URL="https://golang.org/dl/go1.8.linux-amd64.tar.gz" \
    GOLANG_SHA256="53ab94104ee3923e228a2cb2116e5e462ad3ebaeea06ff04463479d7f12d27ca" \
    GOPATH="/go" \
    GOROOT="/usr/local/go" \
    JAVA_HOME="/opt/jdk1.7.0_79" \
    JRE_HOME="/opt/jdk1.7.0_79/jre" \
    MAVEN_HOME="/usr/share/maven" \
    MAVEN_VERSION="3.3.9" \
    PATH="$PATH:/usr/local/bundle/bin:/go/bin:/usr/local/go/bin:/opt/jdk1.7.0_79/bin:/opt/jdk1.7.0_79/jre/bin" \
    RUBY_MAJOR="2.0" \
    RUBY_VERSION="2.0.0-p594"

RUN env && yum -y update \
    && yum install -y autoconf automake bash bison bzip2 curl gcc gcc-c++ epel-release git \
       libtool make openssl-devel openssl patch readline ruby wget unzip && yum clean all \
# Install Go 1.8
    && curl -fsSL -o golang.tar.gz "$GOLANG_URL" \
    && echo "$GOLANG_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz \
    && mkdir -p "$GOPATH"/{src,bin} \
    && chmod -R 777 "$GOPATH" \
# Install Node 4.8
    && curl -fsSL https://rpm.nodesource.com/setup_4.x | bash - \
    && yum install -y nodejs && yum clean all \
# Install Java 7
    && curl -L0 -o jdk-7u79-linux-x64.tar.gz -H \
       "Cookie: gpw_e24=http%3A%2F%2Fwww.oracle.com%2F; oraclelicense=accept-securebackup-cookie" \
       http://download.oracle.com/otn-pub/java/jdk/7u79-b15/jdk-7u79-linux-x64.tar.gz \
    && tar -C /opt -xzf jdk-7u79-linux-x64.tar.gz \
    && rm jdk-7u79-linux-x64.tar.gz \
    && cd /opt/jdk1.7.0_79/ \
    && alternatives --install /usr/bin/java java /opt/jdk1.7.0_79/bin/java 2 \
    && alternatives --auto java \
    && alternatives --install /usr/bin/jar jar /opt/jdk1.7.0_79/bin/jar 2 \
    && alternatives --install /usr/bin/javac javac /opt/jdk1.7.0_79/bin/javac 2 \
    && alternatives --install /usr/bin/javaws javaws /opt/jdk1.7.0_79/bin/javaws 2 \
    && alternatives --set jar /opt/jdk1.7.0_79/bin/jar \
    && alternatives --set javac /opt/jdk1.7.0_79/bin/javac \
# Install Maven 3.3.9
    && cd /usr/share \
    && wget -q http://archive.apache.org/dist/maven/maven-3/$MAVEN_VERSION/binaries/apache-maven-$MAVEN_VERSION-bin.tar.gz -O - | tar xzf - \
    && mv /usr/share/apache-maven-$MAVEN_VERSION /usr/share/maven \
    && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
# Build protobuf 3.1.0 compiler (necessary because Centos 6 include glibc is incompatible with provided binary)
    && git clone https://github.com/google/protobuf.git \
    && cd protobuf \
    && git checkout v3.1.0 \
    && ./autogen.sh \
    && ./configure \
    && make \
    && make install \
    && ldconfig \
    && go get -u github.com/golang/protobuf/proto \
    && go get -u github.com/golang/protobuf/protoc-gen-go \
    && go get -u google.golang.org/grpc \
# Install Ruby 2.0.0
    && mkdir -p /usr/src/ruby \
    && curl -SL "http://cache.ruby-lang.org/pub/ruby/$RUBY_MAJOR/ruby-$RUBY_VERSION.tar.bz2" \
       | tar -xjC /usr/src/ruby --strip-components=1 \
    && cd /usr/src/ruby \
    && autoconf \
    && ./configure --disable-install-doc \
    && make -j"$(nproc)" \
    && yum remove -y autoconf automake libtool gcc-c++ ruby \
    && make install \
    && rm -r /usr/src/ruby && cd \
    && echo 'gem: --no-rdoc --no-ri' >> "$HOME/.gemrc" \
    && mkdir -p "$GEM_HOME" \
    && gem install bundler \
    && bundle config --global path "$GEM_HOME" \
    && bundle config --global bin "$GEM_HOME/bin" \
# Install clang 3.4.2
    && yum install -y clang && yum clean all \
# Install gas
    && git clone https://github.com/GoASTScanner/gas $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && cd $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && git reset --hard d30c5cde3613e9ba0129febda849e4d4df1d57cd \
    && go install github.com/GoASTScanner/gas

ADD bin /usr/bin
