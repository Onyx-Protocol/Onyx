# Chain Core Dashboard

## Development

#### Setup

Install Node.js:

```
brew install node
```

Install dependencies:

```
npm install
```

Start the development server with

```
npm start
```

By default, the development server uses the following environment variables
with default values to connect to a local Chain Core instance:

```
API_URL=http://localhost:3000/api
PROXY_API_HOST=http://localhost:1999
```

#### Style Guide

We use `eslint` to maintain a consistent code style. To check the source
directory with `eslint`, run:

```
npm run lint src
```

#### Tests

The Chain Core Dashboard has a series of integration tests that can be run
against a running core. First, start Chain Core and Dashboard on their default
ports of 1999 and 3000 respective. Then, start tests with the command:

```
npm test
```

There an extended test suite that can be run with:

```
npm run testExtended
```

(Note: The extended test suite can take significantly longer to run, as test
  files cannot be run in parallel).

### React + Redux

#### ES6

Babel is used to transpile the latest ES6 syntax into a format understood by
both Node.js and browsers. To get an ES6-compatible REPL (or run a one-off script)
you can use the `babel-node` command:

`$(npm bin)/babel-node`

#### Redux Actions

To inspect and debug Redux actions, we recommend the "Redux DevTools" Chrome
extension:

https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd


#### Creating new components

To generate a new component with a connected stylesheet, use the following
command:

```
npm run generate-component Common/MyComponent
```

The above command will create two new files in the `src/components` directory:

```
src/components/Common/MyComponent/MyComponent.jsx
src/components/Common/MyComponent/MyComponent.scss
```

with `MyComponent.scss` imported as a stylesheet into `MyComponent.jsx`.

Additionally, if there is an `index.js` file in `src/components/Common`, it
will appropriately add the newly created component to the index exports.


## Production

In production environments, Chain Core Dashboard is served from within `cored`. The contents
of the application are packaged into a single Go source file that maps generated
filenames to file contents.

To deploy an updated dashboard to production:

1. Package the dashboard in production mode using `webpack` with:

    ```sh
    $ npm run build
    ```

2. Bundle the packaged output into an updated `dashboard.go`:

    ```sh
    $ go install ./cmd/gobundle
    $ gobundle -package dashboard dashboard/public > generated/dashboard/dashboard.go
    $ gofmt -w generated/dashboard/dashboard.go
    ```

3. Commit the resulting `dashboard.go`, then rebuild and start `cored`

    ```sh
    $ go install ./cmd/cored
    $ cored
    ```

    Dashboard will be served at the root path from the `cored` server.

## Generating NOTICE content

The `NOTICE` file documents licenses for software packages that make it into our binary distribution of Chain Core. To generate this file, we use the process documented below:

1. For the basis of the `NOTICE` file, we use a Webpack plugin called [license-webpack-plugin](https://www.npmjs.com/package/license-webpack-plugin). This causes an artifact called `public/3rdpartylicenses.txt` to be generated during production builds. It contains a list of modules that ultimately make it into the final Webpack bundle, alongside the license type and any license file content in those modules, including copyright information.
2. We append the Facebook supplemental patent grant and list any Facebook dependencies, such as `react`, `react-dom`, and `prop-types`.
