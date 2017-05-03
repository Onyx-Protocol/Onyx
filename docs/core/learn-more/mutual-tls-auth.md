# Chain Core - Mutual TLS Authentication

Chain Core 1.2 introduces support for mutual TLS authentication. This means both Chain Core and the client SDKs can authenticate each other using X.509 certificates and the TLS protocol. Previously, client authentication was facilitated through the use of access tokens and HTTP Basic Auth. While still supported, client access tokens are now deprecated.

## Background

This guide will illustrate how to configure Chain Core and the Chain SDKs to use mutual TLS authentication. It assumes you have access to the following files (encoded in PEM format):

- an X.509 certificate (and matching RSA private key) for the server
- the X.509 certificate of the root CA that issued the server's certificate
- an X.509 certificate (and matching RSA private key) for the client
- the X.509 certifcate of the root CA that issued the client's certificate

It is possible that the client and server certificates are issued by the same CA. It is also possible to be issued a certificate that acts as a client and server certificate. In both cases the same file can be used more than once.

## Configuration

Mutual TLS authentication can be broken down into two parts:

- server authentication
- client authentication

Server authentication will involve configuring `cored` with its own X.509 certificate and private key. These will be used to prove its identity to the SDK during client requests. To verify the server's certificate, the SDK must be configured with the certificate of the root CA that issued it.

Client authentication simply reverses the roles. The SDK is configured with its own certificate/key pair and `cored` is configured with the certificate of a root CA trusted by the client.

## Chain Core

### Server Authn

Place the server's certificate and private key in the following locations:

[sidenote]

Note: `$CHAIN_CORE_HOME` can be set as an environment variable. It defaults to `$HOME/.chaincore`.

[/sidenote]

- certificate: `$CHAIN_CORE_HOME/tls.crt`
- private key: `$CHAIN_CORE_HOME/tls.key`

### Client Authn

Set `ROOT_CA_CERTS` to the file path of the root CA certificate that issued the client's certificate.

## Java SDK

The Java SDK exposes public methods to configure the `Client` object for mutual TLS. Both methods are overloaded to accept a file path as a `String` or an `InputStream` of the file's contents.

### Server Authn

`Client#setTrustedCerts` configures the Client object with the certificate of the root CA that issued the server's certificate.

### Client Authn

`Client#setX509KeyPair` configures the Client object with the client's certificate and private key.

### Example

This example assumes the following env vars:

- `TLSCRT`: file path to the client certificate
- `TLSKEY`: file path to the RSA private key
- `ROOT_CA_CERTS`: file path to the root CA that issued the server's certificate

```java
Client client = new Client.Builder()
  .setTrustedCerts(System.getenv("ROOT_CA_CERTS"))
  .setX509KeyPair(System.getenv("TLSCRT"), System.getenv("TLSKEY"))
  .setURL("https://example.com")
  .build();
```

## Ruby SDK

The Ruby SDK accepts an `ssl_params` object as an attribute to the `opts` object used in the `Chain::Client` constructor.

### Server Authn

Set the `ca_file` attribute to the file path of the root CA that issued the server's certificate.

### Client Authn

Set the `cert` and `key` attributes to the client's certificate and private key, respectively.

### Example

This example assumes the following env vars:

- `TLSCRT`: file path to the client certificate
- `TLSKEY`: file path to the RSA private key
- `ROOT_CA_CERTS`: file path to the root CA that issued the server's certificate

```ruby
cert = OpenSSL::X509::Certificate.new(File.read(ENV['TLSCRT']))
key = OpenSSL::PKey::RSA.new(File.read(ENV['TLSKEY']))
ca_file = ENV['ROOT_CA_CERTS']
c = Chain::Client.new(url: "https://example.com", ssl_params: { cert: cert, key: key, ca_file: ca_file })
```

## Node SDK

The Node SDK accepts an `https.Agent` object as an attribute to the `opts` object used in the `Client` constructor.

### Server Authn

Set the `ca` attribute to the root CA that issued the server's certificate.

### Client Authn

Set the `cert` and `key` attributes to the client's certificate and private key, respectively.

### Example

This example assumes the following env vars:

- `TLSCRT`: file path to the client certificate
- `TLSKEY`: file path to the RSA private key
- `ROOT_CA_CERTS`: file path to the root CA that issued the server's certificate

```js
const https = require('https')
const fs = require('fs')
const chain = require('chain-sdk')

let agent = new https.Agent({
  ca: fs.readFileSync(process.env.ROOT_CA_CERTS),
  cert: fs.readFileSync(process.env.TLSCRT),
  key: fs.readFileSync(process.env.TLSKEY)
})

const client = new chain.Client({
  baseUrl: 'https://example.com',
  agent: agent
})
```
