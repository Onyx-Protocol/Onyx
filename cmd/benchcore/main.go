package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

var (
	flagD = flag.Bool("d", false, "delete instances from previous runs")

	appName      = "benchcore"
	testRunID    = appName + randString()
	ami          = "ami-f71883e0" // Ubuntu LTS 16.04
	instanceType = "m3.large"
	subnetID     = "subnet-80560fd9"
	key          = os.Getenv("USER")
	user         = os.Getenv("USER")
	schemaPath   = os.Getenv("CHAIN") + "/core/schema.sql"

	awsConfig = &aws.Config{Region: aws.String("us-east-1")}
	ec2client = ec2.New(awsConfig)
	elbclient = elb.New(awsConfig)

	keyring   = sshAgent(os.Getenv("SSH_AUTH_SOCK"))
	sshConfig = &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(keyring.Signers)},
	}

	killInstanceIDs []*string // instances to terminate on exit
	deleteELBNames  []*string // elbs to delete on exit
)

func sshAgent(socket string) agent.Agent {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	return agent.NewClient(conn)
}

type instance struct {
	id       string
	addr     string
	privAddr string
}

func main() {
	log.SetPrefix(appName + ": ")
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	flag.Parse()

	if *flagD {
		doDelete()
		return
	}

	schema, err := ioutil.ReadFile(schemaPath)
	must(err)

	var (
		db     instance
		client instance
		coreds = make([]instance, 3)
	)

	var wg sync.WaitGroup
	wg.Add(1 + len(coreds) + 1)
	log.Println("starting EC2 instances")
	go makeEC2("pg", &db, &wg)
	for i := range coreds {
		go makeEC2("cored", &coreds[i], &wg)
	}
	go makeEC2("client", &client, &wg)
	killInstanceIDs = append(killInstanceIDs, &db.id, &client.id)
	for i := range coreds {
		killInstanceIDs = append(killInstanceIDs, &coreds[i].id)
	}

	coredBin := mustBuildCored()
	corectlBin := mustBuildCorectl()
	jarFile := mustBuildJAR()

	log.Println("waiting for EC2 instances to open port 22")
	wg.Wait()

	log.Println("init database")
	must(scp(db.addr, schema, "schema.sql", 0644))
	must(scp(db.addr, corectlBin, "corectl", 0755))
	mustRunOn(db.addr, initdbsh)

	dbURL := "postgres://benchcore:benchcorepass@" + db.privAddr + "/core?sslmode=disable"

	log.Println("init cored hosts")
	for _, inst := range coreds {
		must(scp(inst.addr, coredBin, "cored", 0755))
		go mustRunOn(inst.addr, coredsh, "dbURL", dbURL)
	}

	log.Println("init client")
	var elbHost, elbName string
	must(makeELB(coreds, &elbHost, &elbName))
	deleteELBNames = append(deleteELBNames, &elbName)
	coreURL := "http://" + elbHost
	log.Println("core URL:", coreURL)
	must(scp(client.addr, jarFile, "test.jar", 0644))
	mustRunOn(client.addr, clientsh, "coreURL", coreURL, "elbHost", elbHost)
	log.Println("SUCCESS")
	cleanup()
}

func doDelete() {
	desc, err := ec2client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Application"), Values: []*string{aws.String("benchcore")}},
			{Name: aws.String("tag:User"), Values: []*string{&user}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, res := range desc.Reservations {
		for _, inst := range res.Instances {
			killInstanceIDs = append(killInstanceIDs, inst.InstanceID)
		}
	}

	lbs, err := elbclient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		log.Fatal(err)
	}
	for _, desc := range lbs.LoadBalancerDescriptions {
		if strings.HasPrefix(*desc.LoadBalancerName, appName+"-"+user+"-") {
			deleteELBNames = append(deleteELBNames, desc.LoadBalancerName)
		}
	}

	cleanup()
}

func mustBuildCored() []byte {
	log.Println("building cored")

	env := []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	}

	date := time.Now().UTC().Format(time.RFC3339)
	cmd := exec.Command("go", "build",
		"-tags", "insecure_disable_https_redirect",
		"-ldflags", "-X main.buildDate="+date,
		"-o", "/dev/stdout",
		"chain/cmd/cored",
	)
	cmd.Env = mergeEnvLists(env, os.Environ())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	must(err)
	log.Printf("cored executable: %d bytes", len(out))
	return out
}

func mustBuildCorectl() []byte {
	log.Println("building corectl")

	env := []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	}

	cmd := exec.Command("go", "build", "-o", "/dev/stdout", "chain/cmd/corectl")
	cmd.Env = mergeEnvLists(env, os.Environ())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	must(err)
	log.Printf("corectl executable: %d bytes", len(out))
	return out
}

func mustBuildJAR() []byte { return nil }

func cleanup() {
	if len(killInstanceIDs) > 0 {
		_, err := ec2client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIDs: killInstanceIDs})
		if err != nil {
			log.Println(err)
		}
	}

	for _, name := range deleteELBNames {
		_, err := elbclient.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
			LoadBalancerName: name,
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func scp(host string, data []byte, dest string, mode int) error {
	log.Printf("scp %d bytes to %s", len(data), dest)
	var client *ssh.Client
	retry(func() (err error) {
		client, err = ssh.Dial("tcp", host+":22", sshConfig)
		return
	})
	s, err := client.NewSession()
	if err != nil {
		return err
	}
	w, err := s.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "C%04o %d %s\n", mode, len(data), dest)
		w.Write(data)
		w.Write([]byte{0})
	}()

	return s.Run("/usr/bin/scp -tr .")
}

func mustRunOn(host, sh string, keyval ...string) {
	if len(keyval)%2 != 0 {
		log.Fatal("odd params", keyval)
	}
	log.Println("run on", host)
	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		log.Fatal(err)
	}
	s, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	s.Stdout = os.Stdout
	s.Stderr = os.Stderr
	for i := 0; i < len(keyval); i += 2 {
		sh = strings.Replace(sh, "{{"+keyval[i]+"}}", keyval[i+1], -1)
	}
	err = s.Run(sh)
	if err != nil {
		log.Fatal(err)
	}
}

func makeEC2(role string, inst *instance, wg *sync.WaitGroup) {
	defer wg.Done()
	runtoken := randString()
	var n int64 = 1

	var resv *ec2.Reservation
	retry(func() (err error) {
		resv, err = ec2client.RunInstances(&ec2.RunInstancesInput{
			ClientToken:  &runtoken,
			ImageID:      &ami,
			InstanceType: &instanceType,
			KeyName:      &key,
			MinCount:     &n,
			MaxCount:     &n,
			SubnetID:     &subnetID,
		})
		return err
	})

	inst.id = *resv.Instances[0].InstanceID

	retry(func() error {
		_, err := ec2client.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{&inst.id},
			Tags: []*ec2.Tag{
				{Key: aws.String("Application"), Value: &appName},
				{Key: aws.String("User"), Value: &user},
				{Key: aws.String("Run"), Value: &testRunID},
				{Key: aws.String("Role"), Value: &role},
			},
		})
		return err
	})

	var desc *ec2.DescribeInstancesOutput
	retry(func() (err error) {
		desc, err = ec2client.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIDs: []*string{&inst.id},
		})
		if err != nil {
			return err
		}
		info := desc.Reservations[0].Instances[0]
		state := info.State
		const (
			running = 16 // see ec2.InstanceState
			pending = 0
		)
		if *state.Code&0xff == pending {
			return errRetry
		} else if *state.Code&0xff != running {
			reason := ""
			if x := info.StateReason; x != nil {
				reason = *x.Message
			}
			return fmt.Errorf("instance %s state %s (%s)", inst.id, *state.Name, reason)
		}
		inst.privAddr = *info.PrivateIPAddress
		inst.addr = *info.PublicIPAddress
		return nil
	})

	retry(func() error {
		conn, err := net.Dial("tcp", inst.addr+":22")
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "refused") {
			return errRetry
		} else if err != nil {
			return err
		}
		conn.Close()
		return nil
	})

}

func makeELB(ec2Instances []instance, addr, name *string) error {
	*name = appName + "-" + user + "-" + randString()[:10]
	resp, err := elbclient.CreateLoadBalancer(&elb.CreateLoadBalancerInput{
		LoadBalancerName: name,
		Subnets:          []*string{&subnetID},
		Scheme:           aws.String("internal"),
		Listeners: []*elb.Listener{{
			InstancePort:     aws.Int64(8080),
			InstanceProtocol: aws.String("HTTP"),
			LoadBalancerPort: aws.Int64(80),
			Protocol:         aws.String("HTTP"),
		}},
	})
	if err != nil {
		return err
	}
	*addr = *resp.DNSName

	var instances []*elb.Instance
	for _, inst := range ec2Instances {
		instances = append(instances, &elb.Instance{InstanceID: &inst.id})
	}
	_, err = elbclient.RegisterInstancesWithLoadBalancer(&elb.RegisterInstancesWithLoadBalancerInput{
		LoadBalancerName: name,
		Instances:        instances,
	})
	return err
}

var errRetry = errors.New("retry")

// retry f until it returns nil.
// wait 500ms in between attempts.
// log err unless it is errRetry.
// after 5 failures, it will call log.Fatal.
// returning errRetry doesn't count as a failure.
func retry(f func() error) {
	for n := 0; n < 5; {
		err := f()
		if err != nil && err != errRetry {
			log.Println("retrying:", err)
			n++
		}
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return
	}
	log.Fatal("too many retries")
}

func randString() string {
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalln(err)
	}
	return hex.EncodeToString(b)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
// This always returns a newly allocated slice.
func mergeEnvLists(in, out []string) []string {
	out = append([]string(nil), out...)
NextVar:
	for _, inkv := range in {
		k := strings.SplitAfterN(inkv, "=", 2)[0]
		for i, outkv := range out {
			if strings.HasPrefix(outkv, k) {
				out[i] = inkv
				continue NextVar
			}
		}
		out = append(out, inkv)
	}
	return out
}

const initdbsh = `#!/bin/bash
set -eo pipefail

sudo bash <<EOFSUDO
set -eo pipefail
apt-get update -qq
apt-get install -y -qq postgresql-9.5 postgresql-client-9.5

cat <<EOF >/etc/postgresql/9.5/main/postgresql.conf
data_directory = '/var/lib/postgresql/9.5/main'
hba_file = '/etc/postgresql/9.5/main/pg_hba.conf'
ident_file = '/etc/postgresql/9.5/main/pg_ident.conf'
external_pid_file = '/var/run/postgresql/9.5-main.pid'
listen_addresses = '*'
port = 5432
max_connections = 100
unix_socket_directories = '/var/run/postgresql'
ssl = true
ssl_cert_file = '/etc/ssl/certs/ssl-cert-snakeoil.pem'
ssl_key_file = '/etc/ssl/private/ssl-cert-snakeoil.key'
shared_buffers = 128MB
dynamic_shared_memory_type = posix
log_timezone = 'UTC'
stats_temp_directory = '/var/run/postgresql/9.5-main.pg_stat_tmp'
datestyle = 'iso, mdy'
timezone = 'UTC'
lc_messages = 'en_US.UTF-8'
lc_monetary = 'en_US.UTF-8'
lc_numeric = 'en_US.UTF-8'
lc_time = 'en_US.UTF-8'
default_text_search_config = 'pg_catalog.english'
EOF

cat <<EOF >/etc/postgresql/9.5/main/pg_hba.conf
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local   all             postgres                                peer
local   all             all                                     peer
host    all             all             0.0.0.0/0               md5
host    all             all             ::0/0                   md5
EOF

/etc/init.d/postgresql restart
EOFSUDO

sudo -u postgres bash <<EOFPOSTGRES
set -eo pipefail
/usr/lib/postgresql/9.5/bin/createdb core
/usr/lib/postgresql/9.5/bin/psql \
	--quiet \
	-c "CREATE USER benchcore WITH PASSWORD 'benchcorepass' SUPERUSER" \
	core
/usr/lib/postgresql/9.5/bin/psql --quiet -f $HOME/schema.sql core
EOFPOSTGRES

export DATABASE_URL='postgres://benchcore:benchcorepass@localhost/core'
$HOME/corectl config-generator
`

const coredsh = `#!/bin/bash
set -eo pipefail
export DATABASE_URL='{{dbURL}}'
./cored
`

const clientsh = `#!/bin/bash
set -eo pipefail


sudo bash <<EOFSUDO
set -eo pipefail

(
	echo 'debconf shared/accepted-oracle-license-v1-1 select true'
	echo 'debconf shared/accepted-oracle-license-v1-1 seen true'
) | debconf-set-selections

mkdir -p /var/cache/oracle-jdk8-installer
cat <<EOF >/var/cache/oracle-jdk8-installer/wgetrc
noclobber = off
dir_prefix = .
dirstruct = off
progress = dot:giga
verbose = off
quiet = on
tries = 5
EOF

add-apt-repository ppa:webupd8team/java
apt-get update -qq
apt-get install -y -qq oracle-java8-installer

EOFSUDO

export JAVA_HOME=/usr/lib/jvm/java-8-oracle

export CHAIN_API_URL='{{coreURL}}'
echo waiting for elb dns to resolve
while ! host {{elbHost}}
do sleep 5 # sigh
done
curl -sv "$CHAIN_API_URL"/debug/vars
#wget --quiet -O t.jar https://s3.amazonaws.com/chain-qa/chain-core-qa.jar
#java -ea -cp t.jar com.chain.qa.singlecore.Main
`
