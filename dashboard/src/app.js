/*eslint-env node*/

import 'bootstrap-loader'
import React from 'react'
import { render } from 'react-dom'
import Dashboard from 'Dashboard'
import configureStore from 'configureStore'

// Set favicon
let faviconPath = require('!!file?name=favicon.ico!../static/images/favicon.png')
let favicon = document.createElement('link')
favicon.type = 'image/png'
favicon.rel = 'shortcut icon'
favicon.href = faviconPath
document.getElementsByTagName('head')[0].appendChild(favicon)

// Start app
export const store = configureStore()
render(
	<Dashboard store={store}/>,
	document.getElementById('root')
)
