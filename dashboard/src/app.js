/*eslint-env node*/

import 'bootstrap-loader'
import React from 'react'
import { render } from 'react-dom'
import Root from 'Root'
import configureStore from 'configureStore'

// Set favicon
let faviconPath = require('!!file?name=favicon.ico!./assets/images/favicon.png')
let favicon = document.createElement('link')
favicon.type = 'image/png'
favicon.rel = 'shortcut icon'
favicon.href = faviconPath
document.getElementsByTagName('head')[0].appendChild(favicon)

// Start app
const store = configureStore()
render(
	<Root store={store}/>,
	document.getElementById('root')
)
