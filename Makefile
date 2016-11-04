REPO = chain
SCHEMA_PATH = $(GOPATH)/src/$(REPO)/core/appdb/schema.sql
DB_DEV = core
DB_DEV_URL = postgres:///$(DB_DEV)?sslmode=disable
DB_OTHER = core2
DB_OTHER_URL = postgres:///$(DB_OTHER)?sslmode=disable
OTHER_PORT=1998
EXEC_NAME = cored
EXEC_PATH = $(GOPATH)/bin/$(EXEC_NAME)

run: build
	$(EXEC_PATH)

other: build
	LISTEN=:$(OTHER_PORT) DATABASE_URL=$(DB_OTHER_URL) $(EXEC_PATH)

dash:
	npm --prefix dashboard start

dash-other:
	PROXY_API_HOST=http://localhost:$(OTHER_PORT) npm --prefix dashboard start

docserver:
	go install chain/cmd/md2html
	md2html

build:
	go install -tags 'insecure_disable_https_redirect' $(REPO)/cmd/$(EXEC_NAME)

resetdb:
	-dropdb $(DB_DEV)
	createdb $(DB_DEV)
	go install $(REPO)/cmd/migratedb
	migratedb -d $(DB_DEV_URL)

resetdb-other:
	-dropdb $(DB_OTHER)
	createdb $(DB_OTHER)
	go install $(REPO)/cmd/migratedb
	migratedb -d $(DB_OTHER_URL)

reseed: resetdb
	go install $(REPO)/cmd/corectl
	DATABASE_URL=$(DB_DEV_URL) corectl config-generator

old-resetdb:
	-dropdb $(DB_DEV)
	createdb $(DB_DEV)
	migratedb -d postgres:///$(DB_DEV)?sslmode=disable

old-reseed: old-resetdb
	go run $(GOPATH)/src/$(REPO)/cmd/corectl/main.go boot hello@chain.com password | tee ~/src/chain-sdk-java/core_config.json

setup-generator:
	./DO_NOT_COMMIT_setup_generator

setup-follower: resetdb-other
	./DO_NOT_COMMIT_setup_follower
