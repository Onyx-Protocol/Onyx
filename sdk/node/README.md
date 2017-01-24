# Chain Node.js SDK

## Usage

### Get the package

The Chain Node SDK is available [via npm](https://www.npmjs.com/package/chain-sdk). Make sure to use the most recent
version whose major and minor components (`major.minor.x`) match your version
of Chain Core. Node 4 or greater is required.

For most applications, you can simply add Chain to your `package.json` with
the following command:

```
npm install --save chain-sdk@1.0.1
```

### In your code

```
const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
```

## Asynchronous Operation

There are two options for interacting with the SDK asynchronously: promises and
callbacks.

With promises:

```
client.transactions.query({}).then(data => {
  // operate on data
  console.log(data)
})
```

With callbacks:

```
let callback = (err, data) => {
  // operate on data
  console.log(data)
}

client.transactions.query({}, callback)
```

## Testing

To run integration tests, run an instance of Chain Core on localhost:1999.
Then run:

```
npm test
```
