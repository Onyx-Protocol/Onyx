# The following commands may be useful for developing and debugging
# our Raft packages. 

build:
	go install  -tags 'localhost_auth http_ok' chain/cmd/cored
	dropdb --if-exists core0
	rm -rf $(HOME)/.chaincore{0,1,2}
	createdb core0

run0:
	LISTEN=localhost:1999 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore0 \
	ROOT_CA_CERTS=$(HOME)/.chaincore0/root_ca_certs \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run1:
	LISTEN=localhost:1998 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore1 \
	ROOT_CA_CERTS=$(HOME)/.chaincore1/root_ca_certs \
	BOOTURL=https://localhost:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run2:
	LISTEN=localhost:1997 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore2 \
	ROOT_CA_CERTS=$(HOME)/.chaincore2/root_ca_certs \
	BOOTURL=https://localhost:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored
