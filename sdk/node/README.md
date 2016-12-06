# Chain Node.js SDK

## Usage

### Get the package

The Node SDK is available [via NPM](). Make sure to use the most recent version whose major and minor components (`major.minor.x`) match your version of Chain Core. Node 6 or later is required.

For most applications, you can simply add Chain to your `package.json` with the following command:

```
npm install --save chain-sdk
```

### In your code

```
const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()
```

## Testing

To run integration tests, run an instance of Chain Core on localhost:1999. Then run:

```
npm test
```
