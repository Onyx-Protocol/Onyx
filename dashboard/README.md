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

#### Tests

```
npm test
```

#### Running the development server

To connect to Chain Core in development, we use a proxy server to provide a
simpler local experience without running into CORs issues.

To start the server in proxy mode, you can use the following example command:

```
API_URL=http://localhost:3000/api PROXY_API_HOST=http://localhost:8080 npm start
```

Then navigate to http://localhost:3000

_NOTE: the `/api` suffix on the `API_URL` variable is required for properly
scoping proxied calls._

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
