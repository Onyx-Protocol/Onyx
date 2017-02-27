// Original from https://github.com/mxstbr/react-boilerplate/

/*eslint-env node*/

/**
 * WEBPACK DLL GENERATOR
 *
 * This profile is used to cache webpack's module
 * contexts for external library and framework type
 * dependencies which will usually not change often enough
 * to warrant building them from scratch every time we use
 * the webpack process.
 */

const { join } = require('path')
const webpack = require('webpack')
const pkg = require(join(process.cwd(), 'package.json'))

const outputPath = join(process.cwd(), 'node_modules/dashboard-dlls')

const config = require('./webpack.base')({
  context: process.cwd(),
  entry: {dependencies: Object.keys(pkg.dependencies)},
  devtool: 'eval',
  output: {
    filename: '[name].dll.js',
    path: outputPath,
    library: '[name]',
  },
  plugins: [
    new webpack.DllPlugin({
      name: '[name]',
      path: join(outputPath, 'manifest.json')
    }),
  ],
})

module.exports = config
