# The following commands may be useful for developing and debugging
# our Raft packages. 

build:
	go install  -tags 'localhost_auth http_ok' chain/cmd/cored
	dropdb --if-exists core0
	rm -rf $(HOME)/.chaincore{0,1,2}
	createdb core0

run0:
	LISTEN=127.0.0.1:1999 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore0 \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run1:
	LISTEN=127.0.0.1:1998 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore1 \
	BOOTURL=http://127.0.0.1:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run2:
	LISTEN=127.0.0.1:1997 \
	CHAIN_CORE_HOME=$(HOME)/.chaincore2 \
	BOOTURL=http://127.0.0.1:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored
