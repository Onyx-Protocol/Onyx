# Chain Node.js SDK

## Usage

### Get the package

The Chain Node SDK is available [via npm](https://www.npmjs.com/package/chain-sdk). Make sure to use the most recent version whose major and minor components (`major.minor.x`) match your version of Chain Core. Node 4 or greater is required.

To install, add the `chain-sdk` NPM module to your `package.json`, using a tilde range (`~`) and specifying the patch version:

```
{
  "dependencies": {
    "chain-sdk": "~1.2.0-rc.2"
  }
}
```

### In your code

```
const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
```

## Asynchronous Operation

There are two options for interacting with the SDK asynchronously: promises and callbacks.

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

## Using external signers

To connect to an HSM other than the built-in MockHSM, you must create a new `Connection` object:

```
const myHsmConnection = new chain.Connection('https://myhost.dev/mockhsm', 'tokenname:tokenvalue')
signer.addKey(myKey, myHsmConnection)
```

## Testing

To run integration tests, run an instance of Chain Core on localhost:1999.
Then run:

```
npm test
```
