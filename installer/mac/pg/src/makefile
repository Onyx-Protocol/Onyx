# path configuration

PREFIX=$(PWD)/../build
SHAREDPREFIX=$(PREFIX)
PATH=$(PREFIX)/bin:/bin:/usr/bin:/opt/local/bin

POSTGRES_VERSION=9.5.4
POSTGRES_MAJOR_VERSION=9.5

JPEG_VERSION=8d
LIBEDIT_VERSION=20130611-3.1

#http://download.osgeo.org/gdal/
LIBJASPER_VERSION=1.900.1

# http://xmlsoft.org/news.html
LIBXML2_VERSION=2.9.3

# https://www.openssl.org
OPENSSL_VERSION=1.0.1t

# https://github.com/json-c/json-c/wiki
# https://s3.amazonaws.com/json-c_releases/releases/index.html
JSONC_VERSION=0.11


#compiler options
MACOSX_DEPLOYMENT_TARGET=10.7
CFLAGS:=$(CFLAGS) -mmacosx-version-min=10.7
CXXFLAGS:=$(CFLAGS) -mmacosx-version-min=10.7

export CFLAGS CXXFLAGS MACOSX_DEPLOYMENT_TARGET

# commands used for downloading and extracting sources
CURL=/usr/bin/curl -L10 --silent --show-error --remote-name
TAR=/usr/bin/tar xzf

all: postgresql
clean: clean-postgresql clean-openssl clean-libxml2 clean-libjpeg clean-libjasper clean-json-c

#########################
###### PostgreSQL #######
#########################

postgresql: $(PREFIX)/bin/psql
	ruby ../../tools/patch_loader_path.rb

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

postgresql-$(POSTGRES_VERSION)/configure: postgresql-$(POSTGRES_VERSION).tar.bz2
	$(TAR) "postgresql-$(POSTGRES_VERSION).tar.bz2"
	touch $@

postgresql-$(POSTGRES_VERSION).tar.bz2:	
	$(CURL) "https://ftp.postgresql.org/pub/source/v$(POSTGRES_VERSION)/postgresql-$(POSTGRES_VERSION).tar.bz2"

clean-postgresql:
	rm -Rf postgresql-$(POSTGRES_VERSION)
	
		
#########################
####### OpenSSL #########
#########################

openssl: $(SHAREDPREFIX)/lib/libssl.dylib

$(SHAREDPREFIX)/lib/libssl.dylib: openssl-${OPENSSL_VERSION}/Makefile
	make -C openssl-${OPENSSL_VERSION}
	make -C openssl-${OPENSSL_VERSION} install_sw

openssl-${OPENSSL_VERSION}/Makefile: openssl-${OPENSSL_VERSION}/Configure
	cd openssl-${OPENSSL_VERSION} && ./Configure --prefix="${SHAREDPREFIX}" darwin64-x86_64-cc zlib no-asm no-krb5 shared

openssl-${OPENSSL_VERSION}/Configure: openssl-${OPENSSL_VERSION}.tar.gz
	$(TAR) openssl-${OPENSSL_VERSION}.tar.gz
	touch $@

openssl-${OPENSSL_VERSION}.tar.gz:
	$(CURL) "https://www.openssl.org/source/openssl-${OPENSSL_VERSION}.tar.gz"
	
clean-openssl:
	rm -Rf "openssl-${OPENSSL_VERSION}"

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

libxml2-${LIBXML2_VERSION}/configure: libxml2-${LIBXML2_VERSION}.tar.gz
	$(TAR) libxml2-${LIBXML2_VERSION}.tar.gz
	touch $@
	
libxml2-${LIBXML2_VERSION}.tar.gz:
	$(CURL) "ftp://xmlsoft.org/libxml2/libxml2-${LIBXML2_VERSION}.tar.gz"
	
clean-libxml2:
	rm -Rf "libxml2-$(LIBXML2_VERSION)"

#########################
####### LibEdit #########
#########################

libedit: $(SHAREDPREFIX)/lib/libedit.dylib

$(SHAREDPREFIX)/lib/libedit.dylib: libedit-$(LIBEDIT_VERSION)/Makefile
	make -C "libedit-$(LIBEDIT_VERSION)" install

libedit-$(LIBEDIT_VERSION)/Makefile: libedit-$(LIBEDIT_VERSION)/configure
	cd libedit-$(LIBEDIT_VERSION) && ./configure --prefix="$(SHAREDPREFIX)"

libedit-$(LIBEDIT_VERSION)/configure: libedit-$(LIBEDIT_VERSION).tar.gz
	$(TAR) "libedit-${LIBEDIT_VERSION}.tar.gz"
	touch $@

libedit-$(LIBEDIT_VERSION).tar.gz:
	$(CURL) "http://www.thrysoee.dk/editline/libedit-$(LIBEDIT_VERSION).tar.gz"

clean-libedit:
	rm -Rf "libedit-$(LIBEDIT_VERSION)"


#########################
####### LibJasper #######
#########################

libjasper: $(SHAREDPREFIX)/lib/libjasper.dylib

$(SHAREDPREFIX)/lib/libjasper.dylib: jasper-${LIBJASPER_VERSION}.uuid/Makefile
	make -C jasper-${LIBJASPER_VERSION}.uuid install

jasper-${LIBJASPER_VERSION}.uuid/Makefile: jasper-${LIBJASPER_VERSION}.uuid/configure
	cd jasper-${LIBJASPER_VERSION}.uuid && ./configure --prefix="$(SHAREDPREFIX)" --disable-debug --disable-dependency-tracking --enable-shared --enable-dynamic


jasper-${LIBJASPER_VERSION}.uuid/configure: jasper-$(LIBJASPER_VERSION).uuid.tar.gz
	$(TAR) "jasper-$(LIBJASPER_VERSION).uuid.tar.gz"
	touch $@

jasper-$(LIBJASPER_VERSION).uuid.tar.gz:
	$(CURL) "http://download.osgeo.org/gdal/jasper-$(LIBJASPER_VERSION).uuid.tar.gz"

clean-libjasper:
	rm -Rf jasper-${LIBJASPER_VERSION}.uuid


#########################
#######  libjpeg  #######
#########################

libjpeg: $(SHAREDPREFIX)/lib/libjpeg.dylib

$(SHAREDPREFIX)/lib/libjpeg.dylib: jpeg-$(JPEG_VERSION)/Makefile
	make -C jpeg-$(JPEG_VERSION) install

jpeg-$(JPEG_VERSION)/Makefile: jpeg-$(JPEG_VERSION)/configure
	cd jpeg-$(JPEG_VERSION) && ./configure --prefix="$(SHAREDPREFIX)" --disable-dependency-tracking

jpeg-$(JPEG_VERSION)/configure: jpegsrc.v$(JPEG_VERSION).tar.gz
	$(TAR) jpegsrc.v$(JPEG_VERSION).tar.gz
	touch $@

jpegsrc.v$(JPEG_VERSION).tar.gz:
	$(CURL) "http://www.ijg.org/files/jpegsrc.v$(JPEG_VERSION).tar.gz"

clean-libjpeg:
	rm -Rf jpeg-$(JPEG_VERSION)



#########################
###### JSON-c ###########
#########################

$(SHAREDPREFIX)/include/json/json_object_iterator.h: json-c-$(JSONC_VERSION)/Makefile
	make -C json-c-$(JSONC_VERSION) install
	cp json-c-$(JSONC_VERSION)/json_object_iterator.h "$(SHAREDPREFIX)/include/json/json_object_iterator.h"


json-c-$(JSONC_VERSION)/Makefile: json-c-$(JSONC_VERSION)/configure
	cd json-c-$(JSONC_VERSION) && ./configure --prefix="$(SHAREDPREFIX)"

json-c-$(JSONC_VERSION)/configure: json-c-$(JSONC_VERSION).tar.gz
	$(TAR) json-c-$(JSONC_VERSION).tar.gz
	touch $@

json-c-$(JSONC_VERSION).tar.gz:
	$(CURL) "https://s3.amazonaws.com/json-c_releases/releases/json-c-$(JSONC_VERSION).tar.gz"

clean-json-c:
	rm -Rf json-c-$(JSONC_VERSION)



