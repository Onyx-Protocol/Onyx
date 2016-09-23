import React from 'react'
import styles from './JsonField.scss'
import AceEditor from 'react-ace'

import 'brace/mode/json'
import 'brace/theme/github'
import { parseNonblankJSON } from '../../../utility/string'

class JsonField extends React.Component {
  constructor(props) {
    super(props)
    this.state = {syntaxError: {}}
  }

  render() {
    const hint = this.props.hint || 'All JSON values supported'
    const fieldProps = this.props.fieldProps
    const displayProps = {
      mode: 'json',
      theme:'github',
      height: '100px',
      width: '100%',
      tabSize: 2,
      showGutter: false,
      highlightActiveLine: false,
      showPrintMargin: false,
      editorProps: {$blockScrolling: true}
    }

    const onLoad = (editor) => {
      const self = this

      editor.navigateFileStart()
      editor.navigateDown()
      editor.navigateRight(1)

      // Restore default browser tab-focusing behavior
      editor.commands.bindKey('Tab', null)
      editor.commands.bindKey('Shift-Tab', null)

      editor.getSession().on('changeAnnotation', function() {
        self.setState({syntaxError: editor.getSession().getAnnotations()[0]})
      })
    }

    const showError = fieldProps.touched && !fieldProps.active && fieldProps.error
    const syntaxError = this.state.syntaxError

    const editorStyles = [styles.editorWrapper]
    if (showError) { editorStyles.push(styles.editorError) }

    return (
      <div className='form-group'>
        {this.props.title && <label>{this.props.title}</label>}
        <div className={editorStyles.join(' ')}>
          <AceEditor
            {...fieldProps}
            {...displayProps}
            onLoad={onLoad}
          />
        </div>

        {showError &&
          <span className={`help-block ${styles.errorBlock}`}>
            {fieldProps.error}:
            {syntaxError && ` ${syntaxError.text} on row ${syntaxError.row + 1}`}
          </span>}
        {!showError && <span className='help-block'>{hint}</span>}
      </div>
    )
  }
}

JsonField.validator = value => {
  try {
    parseNonblankJSON(value)
  } catch (err) {
    return 'Error parsing JSON'
  }
  return null
}

export default JsonField
