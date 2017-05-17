package main

import (
	"os"
	"text/template"
)

// Postgres might be installed in a weird place.
// Look for it and return its location if we find it.
// Otherwise it'll have to be in $PATH.
func pgbin() string {
	for _, s := range []string{
		"/usr/lib/postgresql/9.6/bin",
		"/usr/lib/postgresql/9.5/bin",
		"/Applications/Postgres.app/Contents/Versions/9.6/bin",
		"/Applications/Postgres.app/Contents/Versions/9.5/bin",
		"/usr/pgsql-9.6/bin",
	} {
		if fi, err := os.Lstat(s); err == nil && fi.IsDir() {
			return s
		}
	}
	return ""
}

// based on a mix of RDS's default.postgres9.4 parameter group, the Flynn
// defaults, and my own notes from running local postgreses
var configTemplate = template.Must(template.New("postgresql.conf").Parse(`
port = {{.port}}
listen_addresses = 'localhost'
max_connections = 100
shared_buffers = 128MB
dynamic_shared_memory_type = posix
unix_socket_directories = '{{.sockdir}}'

# replication
# logging
datestyle = 'iso, mdy'
timezone = 'UTC'
lc_messages = 'en_US.UTF-8'     # locale for system error message
lc_monetary = 'en_US.UTF-8'     # locale for monetary formatting
lc_numeric = 'en_US.UTF-8'      # locale for number formatting
lc_time = 'en_US.UTF-8'       # locale for time formatting

# default configuration for text search
default_text_search_config = 'pg_catalog.english'

log_destination = 'csvlog'
log_directory = '{{.logdir}}'
log_filename = 'integtest-queries.log'
log_file_mode = 0644
log_min_duration_statement = {{.logdur}}
`))

var hbaTemplate = template.Must(template.New("pg_hba.conf").Parse(`
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             all                                     trust
# IPv4 local connections:
host    all             all             127.0.0.0/8             trust
# IPv6 local connections:
host    all             all             ::1/128                 trust
`))
