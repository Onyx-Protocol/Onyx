
build:
	go install chain/cmd/cored
	dropdb --if-exists core0
	rm -rf $(HOME)/.cored{0,1,2}
	createdb core0

run0:
	LISTEN=127.0.0.1:1999 \
	DIR=$(HOME)/.cored0 \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run1:
	LISTEN=127.0.0.1:1998 \
	DIR=$(HOME)/.cored1 \
	BOOTURL=http://127.0.0.1:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored

run2:
	LISTEN=127.0.0.1:1997 \
	DIR=$(HOME)/.cored2 \
	BOOTURL=http://127.0.0.1:1999/ \
	DATABASE_URL=postgres:///core0?sslmode=disable \
	cored
