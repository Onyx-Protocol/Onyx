/*eslint-env node*/

var webpack = require('webpack')
var getConfig = require('hjs-webpack')
var path = require('path')

// Set base path to JS and CSS files when
// required by other files
let publicPath = '/'
if (process.env.NODE_ENV === 'production') {
  publicPath = '/dashboard/'
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
      context: '/api',
      options: {
        target: process.env.PROXY_API_HOST || 'http://localhost:1999',
        pathRewrite: {
          '^/api': ''
        }
      }
    }
  }
})

// Enable CSS modules
let loaders = config.module.loaders

for (let item of loaders) {
  if (item.loader) {
    item.loader = item.loader.replace('css-loader','css-loader?modules&importLoaders=1&localIdentName=[name]__[local]__[hash:base64:5]')
  }
  if ('.scss'.match(item.test) != null) {
    item.loader = item.loader.replace('sass-loader','sass-loader!sass-resources-loader')
  }
}

config.module.loaders = loaders
config.sassResources = './src/assets/styles/resources.scss'

// Configure node modules which may or
// may not be present in the browser.
config.node = {
  console: true,
  fs: 'empty',
  net: 'empty',
  tls: 'empty'
}

config.resolve = {
  root: path.resolve('./src'),
  extensions: [ '', '.js', '.jsx' ]
}

// module.noParse disables parsing for
// matched files. Used here to bypass
// issues with an AMD configured module.
config.module.noParse = /node_modules\/json-schema\/lib\/validate\.js/

// Import specified env vars in packaged source
config.plugins.push(new webpack.DefinePlugin({
  'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
  'process.env.API_URL': JSON.stringify(process.env.API_URL),
  'process.env.PROXY_API_HOST': JSON.stringify(process.env.PROXY_API_HOST),
  'process.env.TESTNET_INFO_URL': JSON.stringify(process.env.TESTNET_INFO_URL),
  'process.env.TESTNET_GENERATOR_URL': JSON.stringify(process.env.TESTNET_GENERATOR_URL),
}))

config.output.publicPath = publicPath

// Support source maps for Babel
config.devtool = 'source-map'

module.exports = config
