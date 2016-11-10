import { cd, mkdir, echo, ShellString, cat, test } from 'shelljs'
import commandLineCommands from 'command-line-commands'
import commandLineArgs from 'command-line-args'

const validCommands = [ 'component' ]
const { command, argv } = commandLineCommands(validCommands)

function template(strings, ...keys) {
  return (function(...values) {
    var dict = values[values.length - 1] || {}
    var result = [strings[0]]
    keys.forEach(function(key, i) {
      var value = Number.isInteger(key) ? values[key] : dict[key]
      result.push(value, strings[i + 1])
    })
    return result.join('')
  })
}

const baseJsx =
template`import React from 'react'
import styles from './${0}.scss'

class ${0} extends React.Component {
  render() {
    return (
      <div className={styles.base}>
        <p>This is a ${0}</p>
      </div>
    )
  }
}

export default ${0}
`

const baseScss =
`.base {
  background: red;
}
`

switch (command) {
  case 'component': {

    let optionDefinitions = [
      { name: 'path', type: String, defaultOption: true }
    ]

    let options = commandLineArgs(optionDefinitions, argv)

    let path = options.path
    if (path === undefined) {
      echo('No component name specified.\nUsage: `npm run generate-component My/Name`')
      break
    }

    cd('src')
    mkdir('-p', path)

    let name = path.split('/').pop()
    ShellString(baseJsx(name)).to(path + `/${name}.jsx`)
    ShellString(baseScss).to(path + `/${name}.scss`)

    let indexPath = path.split('/').slice(0,-1).join('/') + '/index.js'
    if (test('-f', indexPath)) {
      let contents = cat(indexPath)

      let importString = `import ${name} from './${name}/${name}'`
      let exportString = `  ${name},\n`

      if (contents.indexOf(importString) < 0) {
        contents = `${importString}\n${contents}`
      }
      if (contents.indexOf(exportString) < 0) {
        let exportTarget = 'export {\n'
        let position = contents.indexOf(exportTarget)  + exportTarget.length

        contents = contents.slice(0, position)
          + exportString
          + contents.slice(position)
      }

      ShellString(contents).to(indexPath)
    }
    break
  }
  default: {
    echo(`Unknown command: ${command}`)
    break
  }
}
