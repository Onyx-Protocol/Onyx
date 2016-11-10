// Original from https://github.com/mxstbr/react-boilerplate/

/*eslint-env node*/

// No need to build the DLL in production
if (process.env.NODE_ENV === 'production') {
  process.exit(0)
}

require('shelljs/global')

const path = require('path')
const fs = require('fs')
const exists = fs.existsSync
const writeFile = fs.writeFileSync

const pkg = require(path.join(process.cwd(), 'package.json'))
const outputPath = path.join(process.cwd(), 'node_modules/dashboard-dlls')
const dllManifestPath = path.join(outputPath, 'package.json')

/**
 * I use node_modules/react-boilerplate-dlls by default just because
 * it isn't going to be version controlled and babel wont try to parse it.
 */
mkdir('-p', outputPath)

echo('Building the Webpack DLL...')

/**
 * Create a manifest so npm install doesn't warn us
 */
if (!exists(dllManifestPath)) {
  writeFile(
    dllManifestPath,
    JSON.stringify({
      name: 'react-boilerplate-dlls',
      private: true,
      author: pkg.author,
      repository: pkg.repository,
      version: pkg.version,
    }),
    'utf8'
  )
}

// the BUILDING_DLL env var is set to avoid confusing the development environment
exec('webpack --display-chunks --display-error-details --color --config src/config/webpack.dll.js')
