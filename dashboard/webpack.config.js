/*eslint-env node*/

var webpack = require('webpack')
var getConfig = require('hjs-webpack')

// Set base path to JS and CSS files when
// required by other files
let publicPath = "/"
if (process.env.NODE_ENV === "production") {
  publicPath = "/dashboard/"
}

// Creates a webpack config object. The
// object can be extended by accessing
// its properties.
var config = getConfig({
  // entry point for the app
  in: 'src/app.js',

  // Name or full path of output directory
  // commonly named `www` or `public`. This
  // is where your fully static site should
  // end up for simple deployment.
  out: 'public',

  output: {
    hash: true
  },

  // This will destroy and re-create your
  // `out` folder before building so you always
  // get a fresh folder. Usually you want this
  // but since it's destructive we make it
  // false by default
  clearBeforeBuild: true,

  html: function (context) {
    return {
      'index.html': context.defaultTemplate({
        publicPath: publicPath
      })
    }
  },

  // Proxy API requests to local core server
  devServer: {
    proxy: {
      context: "/api",
      options: {
        target: process.env.PROXY_API_HOST || "http://localhost:8080",
        pathRewrite: {
          "^/api": ""
        }
      }
    }
  }
})

// Enable CSS modules
let loaders = config.module.loaders

for (let item of loaders) {
  if (item.loader) {
    item.loader = item.loader.replace("css-loader","css-loader?modules&importLoaders=1&localIdentName=[name]__[local]__[hash:base64:5]")
  }
  if (".scss".match(item.test) != null) {
    item.loader = item.loader.replace("sass-loader","sass-loader!sass-resources-loader")
  }
}
config.module.loaders = loaders
config.sassResources = './src/styles/resources.scss'

// Configure node modules which may or
// may not be present in the browser.
config.node = {
  console: true,
  fs: 'empty',
  net: 'empty',
  tls: 'empty'
}

// module.noParse disables parsing for
// matched files. Used here to bypass
// issues with an AMD configured module.
config.module.noParse = /node_modules\/json-schema\/lib\/validate\.js/

// Import only specified env vars in packaged source
config.plugins.push(new webpack.EnvironmentPlugin([
  'NODE_ENV',
  'API_URL',
  'PROXY_API_HOST'
]))

config.output.publicPath = publicPath

module.exports = config
