# path configuration

PREFIX=$(PWD)/../build
TOOLS=$(PWD)/../../tools
SHAREDPREFIX=$(PREFIX)
PATH=$(PREFIX)/bin:/bin:/usr/bin:/opt/local/bin

POSTGRES_VERSION=9.5.4
POSTGRES_MAJOR_VERSION=9.5
POSTGRES_SHA256=cf5e571164ad66028ecd7dd8819e3765470d45bcd440d258b686be7e69c76ed0

# http://xmlsoft.org/news.html
LIBXML2_VERSION=2.9.3
LIBXML2_SHA256=4de9e31f46b44d34871c22f54bfc54398ef124d6f7cafb1f4a5958fbcd3ba12d

# https://www.openssl.org
OPENSSL_VERSION=1.0.1t
OPENSSL_SHA256=4a6ee491a2fdb22e519c76fdc2a628bb3cec12762cd456861d207996c8a07088

#compiler options
MACOSX_DEPLOYMENT_TARGET=10.7
CFLAGS:=$(CFLAGS) -mmacosx-version-min=10.7
CXXFLAGS:=$(CFLAGS) -mmacosx-version-min=10.7

export CFLAGS CXXFLAGS MACOSX_DEPLOYMENT_TARGET

# commands used for downloading and extracting sources
CURL=/usr/bin/curl -L10 --silent --show-error --remote-name
TAR=/usr/bin/tar xzf

all: postgresql
clean: clean-postgresql clean-openssl clean-libxml2

#########################
###### PostgreSQL #######
#########################

postgresql: $(PREFIX)/bin/psql
	ruby $(TOOLS)/patch_loader_path.rb

$(PREFIX)/bin/psql: postgresql-$(POSTGRES_VERSION)/GNUmakefile
	make -C "postgresql-$(POSTGRES_VERSION)" all contrib
	make -C "postgresql-$(POSTGRES_VERSION)" install
	
# setting PATH is to make sure we find the right xml2-config
# the --with-includes and --with-libraries options are necessary so
# that postgres will be compiled and linked against our own versions
# of libraries like openssl, instead of system provided versions
postgresql-$(POSTGRES_VERSION)/GNUmakefile: $(SHAREDPREFIX)/lib/libssl.dylib $(SHAREDPREFIX)/lib/libxml2.dylib postgresql-$(POSTGRES_VERSION)/configure 
	cd "postgresql-$(POSTGRES_VERSION)" && export PATH="$(PREFIX)/bin:$$PATH" && ./configure --prefix="$(PREFIX)" --with-includes="$(SHAREDPREFIX)/include" --with-libraries="$(SHAREDPREFIX)/lib" --enable-thread-safety --with-openssl --with-gssapi --with-bonjour --with-libxml --with-libxslt --with-perl --with-python --with-readline --with-uuid=e2fs
# patch makefile to fix python linker flags
	cd postgresql-$(POSTGRES_VERSION)/src; mv Makefile.global Makefile.global.old; sed 's@^python_libspec.*@python_libspec = '"`python-config --ldflags`"'@' Makefile.global.old >Makefile.global

postgresql-$(POSTGRES_VERSION)/configure: postgresql-$(POSTGRES_VERSION).tar.bz2-${POSTGRES_SHA256}
	$(TAR) "postgresql-$(POSTGRES_VERSION).tar.bz2-${POSTGRES_SHA256}"
	touch $@

postgresql-$(POSTGRES_VERSION).tar.bz2-${POSTGRES_SHA256}:
	$(CURL) "https://ftp.postgresql.org/pub/source/v$(POSTGRES_VERSION)/postgresql-${POSTGRES_VERSION}.tar.bz2"
	$(TOOLS)/renamesha256.sh postgresql-${POSTGRES_VERSION}.tar.bz2

clean-postgresql:
	rm -Rf postgresql-$(POSTGRES_VERSION)*
	
		
#########################
####### OpenSSL #########
#########################

openssl: $(SHAREDPREFIX)/lib/libssl.dylib

$(SHAREDPREFIX)/lib/libssl.dylib: openssl-${OPENSSL_VERSION}/Makefile
	make -C openssl-${OPENSSL_VERSION}
	make -C openssl-${OPENSSL_VERSION} install_sw

openssl-${OPENSSL_VERSION}/Makefile: openssl-${OPENSSL_VERSION}/Configure
	cd openssl-${OPENSSL_VERSION} && ./Configure --prefix="${SHAREDPREFIX}" darwin64-x86_64-cc zlib no-asm no-krb5 shared

openssl-${OPENSSL_VERSION}/Configure: openssl-${OPENSSL_VERSION}.tar.gz-${OPENSSL_SHA256}
	$(TAR) openssl-${OPENSSL_VERSION}.tar.gz-${OPENSSL_SHA256}
	touch $@

openssl-${OPENSSL_VERSION}.tar.gz-${OPENSSL_SHA256}:
	$(CURL) "https://www.openssl.org/source/openssl-${OPENSSL_VERSION}.tar.gz"
	$(TOOLS)/renamesha256.sh openssl-${OPENSSL_VERSION}.tar.gz
	
clean-openssl:
	rm -Rf openssl-${OPENSSL_VERSION}*

#########################
######## LibXML2 ########
#########################

libxml2: $(SHAREDPREFIX)/lib/libxml2.dylib

$(SHAREDPREFIX)/lib/libxml2.dylib: libxml2-${LIBXML2_VERSION}/Makefile
	make -C libxml2-${LIBXML2_VERSION} install-exec
	make -C libxml2-${LIBXML2_VERSION}/include install
	touch $(SHAREDPREFIX)/lib/libxml2.dylib

libxml2-${LIBXML2_VERSION}/Makefile: libxml2-${LIBXML2_VERSION}/configure
	cd libxml2-${LIBXML2_VERSION} && ./configure --prefix="$(SHAREDPREFIX)" --disable-dependency-tracking

libxml2-${LIBXML2_VERSION}/configure: libxml2-${LIBXML2_VERSION}.tar.gz-${LIBXML2_SHA256}
	$(TAR) libxml2-${LIBXML2_VERSION}.tar.gz-${LIBXML2_SHA256}
	touch $@
	
libxml2-${LIBXML2_VERSION}.tar.gz-${LIBXML2_SHA256}:
	$(CURL) "ftp://xmlsoft.org/libxml2/libxml2-${LIBXML2_VERSION}.tar.gz"
	$(TOOLS)/renamesha256.sh libxml2-${LIBXML2_VERSION}.tar.gz
	
clean-libxml2:
	rm -Rf libxml2-$(LIBXML2_VERSION)*


