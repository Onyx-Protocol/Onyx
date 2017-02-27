/*eslint-env node*/

// TODO: this should be broken up into `dev` and `prod`
// configuration variants

var webpack = require('webpack')
var getConfig = require('hjs-webpack')
var path = require('path')

// Set base path to JS and CSS files when
// required by other files
let publicPath = '/'
let outPath = 'public'
if (process.env.NODE_ENV === 'production') {
  publicPath = '/dashboard/'
} else {
  outPath = 'node_modules/dashboard-dlls'
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
  out: outPath,

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
        publicPath: publicPath,
        head: process.env.NODE_ENV !== 'production' ? '<script data-dll="true" src="/dependencies.dll.js"></script>' : '',
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

// Customize loader configuration
let loaders = config.module.loaders

for (let item of loaders) {
  // Enable CSS module support
  if (item.loader && item.loader.indexOf('css-loader') > 0) {
    item.loader = item.loader.replace('css-loader','css-loader?module&importLoaders=1&localIdentName=[name]__[local]__[hash:base64:5]')
  }
  if ('.scss'.match(item.test) != null) {
    item.loader = item.loader.replace('sass-loader','sass-loader!sass-resources-loader')
  }

  // Enable babel-loader caching
  if (item.loader == 'babel-loader') {
    item.loader = 'babel-loader?cacheDirectory'
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

// Enable babel-polyfill
config.entry.push('babel-polyfill')

config.output.publicPath = publicPath

if (process.env.NODE_ENV !== 'production') {
  // Support source maps for Babel
  config.devtool = 'eval-cheap-module-source-map'

  // Use DLL
  config.plugins.push(new webpack.DllReferencePlugin({
    context: process.cwd(),
    manifest: require(path.resolve(process.cwd(), 'node_modules/dashboard-dlls/manifest.json')),
  }))
}

module.exports = config
