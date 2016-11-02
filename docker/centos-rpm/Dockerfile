FROM centos:7

RUN yum install -y gcc make rpm-build ruby-devel \
    && gem install fpm

COPY corectl /usr/bin/corectl
COPY cored /usr/bin/cored
COPY init /etc/init.d/cored
COPY before-install /before-install
COPY after-install /after-install
COPY after-remove /after-remove
COPY startup.sh /startup.sh
ENTRYPOINT /startup.sh
