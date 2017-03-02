FROM centos:6

ENV GOLANG_VERSION 1.7.5
ENV GOLANG_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_SHA256 2e4dd6c44f0693bef4e7b46cc701513d74c3cc44f2419bf519d7868b12931ac3
ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV JAVA_HOME /opt/jdk1.7.0_79
ENV JRE_HOME /opt/jdk1.7.0_79/jre
ENV MAVEN_HOME /usr/share/maven
ENV MAVEN_VERSION=3.3.9

RUN yum install -y autoconf automake bash bison bzip2 curl gcc gcc-c++ git go iconv-devel \
    libtool libyaml-devel libffi-devel make openssl-devel openssl patch readline \
    readline-develzlib zlib-develtch readline readline-devel zlib zlib-devel wget unzip \
    && curl -fsSL -o golang.tar.gz "$GOLANG_URL" \
    && echo "$GOLANG_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz \
    && export PATH=$PATH:/usr/local/go/bin \
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
    && export PATH=$PATH:/opt/jdk1.7.0_79/bin:/opt/jdk1.7.0_79/jre/bin \
    && cd /usr/share \
    && wget -q http://archive.apache.org/dist/maven/maven-3/$MAVEN_VERSION/binaries/apache-maven-$MAVEN_VERSION-bin.tar.gz -O - | tar xzf - \
    && mv /usr/share/apache-maven-$MAVEN_VERSION /usr/share/maven \
    && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
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
    && command curl -sSL https://rvm.io/mpapis.asc | gpg2 --import - \
    && curl -L get.rvm.io | bash -s stable \
    && source /etc/profile.d/rvm.sh \
    && /usr/local/rvm/bin/rvm install 2.0.0 \
    && /bin/bash --login \
    && /usr/local/rvm/bin/rvm use 2.0.0 --default

ADD bin /usr/bin
ENV CHAIN ${GOPATH:-$HOME/go}/src/chain
ENV PATH=$PATH:$GOPATH/bin:$GOROOT/bin:/opt/jdk1.7.0_79/bin:/opt/jdk1.7.0_79/jre/bin
RUN git clone https://github.com/GoASTScanner/gas $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && cd $(go env GOPATH)/src/github.com/GoASTScanner/gas \
    && git reset --hard d30c5cde3613e9ba0129febda849e4d4df1d57cd \
    && go install github.com/GoASTScanner/gas

ENV RUBY_MAJOR 2.0
ENV RUBY_VERSION 2.0.0-p594

# some of ruby's build scripts are written in ruby
# we purge this later to make sure our final image uses what we just built
RUN yum -y update && yum -y install ruby && yum clean all \
    && mkdir -p /usr/src/ruby \
    && curl -SL "http://cache.ruby-lang.org/pub/ruby/$RUBY_MAJOR/ruby-$RUBY_VERSION.tar.bz2" \
        | tar -xjC /usr/src/ruby --strip-components=1 \
    && cd /usr/src/ruby \
    && autoconf \
    && ./configure --disable-install-doc \
    && make -j"$(nproc)" \
    && yum remove -y ruby \
    && make install \
    && rm -r /usr/src/ruby

# skip installing gem documentation
RUN echo 'gem: --no-rdoc --no-ri' >> "$HOME/.gemrc"

# install things globally, for great justice
ENV GEM_HOME /usr/local/bundle
ENV PATH $GEM_HOME/bin:$PATH
RUN gem install bundler \
    && bundle config --global path "$GEM_HOME" \
    && bundle config --global bin "$GEM_HOME/bin"

# don't create ".bundle" in all our apps
ENV BUNDLE_APP_CONFIG $GEM_HOME
