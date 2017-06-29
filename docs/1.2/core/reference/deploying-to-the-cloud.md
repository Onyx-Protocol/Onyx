# Deploying to the cloud

Using Docker and Docker Machine, you can deploy Chain Core Developer Edition to a cloud services provider like AWS or Digital Ocean in a matter of minutes.

[Docker Machine](https://docs.docker.com/machine/) is a Docker add-on that makes it easy to work with Docker containers on your preferred cloud services provider. It comes out-of-the-box with tools to provision compute instances to host your Docker containers, and lets you use Docker commands to control your remote containers.

## Step 1: Install Docker Machine

See the [Docker Machine docs](https://docs.docker.com/machine/install-machine/) for platform-specific installation instructions. Most recent distributions of Docker already include Docker Machine.

## Step 2: Provision a new Docker host instance

Docker Machine comes with provisioning tools that let you quickly spin up new Docker-ready compute instances on many popular providers, including AWS, Digital Ocean, Google Cloud, Microsoft Azure, and more.

We've included examples for AWS and Digital Ocean. See the [Docker Machine driver list](https://docs.docker.com/machine/drivers/) for instructions on how to use other hosting providers.

### AWS

First, if you haven't already, download and install the AWS command-line tools, and then configure them:

```
aws configure
```

This will install your AWS credentials to a standard location that will be accessible to the `docker-machine` command. Next, run Docker Machine to create a new ECW instance that will serve as your Docker Host:

```
docker-machine create      \
  --driver amazonec2       \
  --amazonec2-open-port 80 \
  my-example-machine
```

You can replace `my-example-machine` with any valid machine identifier.

This single command accomplishes a lot: it creates a new t2.micro EC2 instance to act as your Docker host. It places it in a security group that is accessible via SSH, the `docker` command, as well as port 80 (HTTP). It also creates a new SSH keypair for console access. The `docker create` command accepts a [variety of flags](https://docs.docker.com/machine/drivers/aws/) for customizing the EC2 instance.

### Digital Ocean

Visit the [Digital Ocean Control Panel](https://cloud.digitalocean.com/settings/api/tokens) and create an API token with read/write privileges. Then, on your local computer, execute the following:

```
docker-machine create                           \
  --driver digitalocean                         \
  --digitalocean-access-token=YOUR-ACCESS-TOKEN \
  my-example-machine
```

You can replace `my-example-machine` with any valid machine identifier.

This command will provision a new droplet using the `ubuntu-16-04-x64` image as a default.

## Step 3: Access your Docker host instance

To get the IP address of your newly-provisioned Docker host instance, use the following command:

```
docker-machine ip my-example-machine
```

For SSH access, use the following command:

```
docker-machine ssh my-example-machine
```

## Step 4: Setup your shell for remote Docker access

Once you've successfully provisioned your Dockerized cloud instance, run the following:

```
eval $(docker-machine env my-example-machine)
```

This exports some environment variables into your shell so that the `docker` command knows to talk to your remote Docker host.

## Step 5: Start Chain Core in a container

To start a new container, run the following:

```
docker run -d -p 80:1999 --name chain chaincore/developer
```

This creates a new container called `chain`, and installs and runs the [chaincore/developer](https://hub.docker.com/r/chaincore/developer/) Docker image in the background. Chain Core will be available at `http://<ip-of-your-cloud-instance>`.

## Step 6: Generate an access token

Since your Chain Core is running on a remote host, you'll need to use an access token to use the dashboard and SDK. Run the following to generate a new access token:

```
docker exec \
  chain     \
  /usr/bin/chain/corectl create-token mytoken client-readwrite
```

This will produce a token of the format `mytoken:5e542...`. Copy and save this value.

## Step 5: Visit the dashboard

Open a web browser and navigate to `http://<ip-of-your-cloud-instance>`. Paste the token from Step 4 into the prompt.
