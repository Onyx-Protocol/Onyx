const webpack = require('webpack')
const path = require('path')
const { CheckerPlugin } = require('awesome-typescript-loader')

module.exports = {
  resolve: {
    extensions: ['', '.js', '.json', '.pegjs', '.jsx', '.ts', '.tsx'],
    alias: {
      "ivy-compiler": path.resolve(__dirname, 'ivy-compiler/src/index.ts'),
      "chain-sdk": path.resolve(__dirname, '../sdk/node/src/index')
    }
  },
  resolveLoader: {
    root: path.join(__dirname, 'node_modules')
  },
  entry: {
    playground: path.resolve(__dirname, 'playground/entry'),
  },
  output: {
    path: path.resolve(__dirname, 'playground/build'),
    filename: 'playground.bundle.js',
    publicPath: "/assets/"
  },
  module: {
    loaders: [
      { test: /\.js$/, loader: 'babel', exclude: /node_modules/},
      { test: /\.pegjs$/, loader: 'pegjs-loader'},
      { test: /\.json$/, loader: 'json'},
      { test: /\.tsx?$/, loaders: ['babel', 'awesome-typescript-loader']},
    ]
  },
  devServer: {
    historyApiFallback: true,

    // Proxy API requests to local core server
    proxy: {
      '/api': {
        target: process.env.PROXY_API_HOST || 'http://localhost:1999/',
        pathRewrite: {
          '^/api': ''
        }
      }
    }
  },
  plugins: [
      new CheckerPlugin()
  ]
}
