import * as React from 'react'
import { connect } from 'react-redux'

import AceEditor from 'react-ace'
import * as Brace from 'brace'
import 'brace/theme/monokai'

import { setSource } from '../actions'

const mapStateToProps = undefined
const mapDispatchToProps = (dispatch) => {
  return {
    handleChange: (value) => {
      dispatch(setSource(value))
    }
  }
}

const Ace = ({ source, handleChange }) => {
  return (
    <div className="panel-body">
      <AceEditor
        mode="ivy"
        theme="monokai"
        onChange={handleChange}
        name="aceEditor"
        width="100%"
        fontSize={16}
        tabSize={2}
        maxLines={25}
        minLines={15}
        value={source}
        editorProps={{$blockScrolling: Infinity}}
        setOptions={{useSoftTabs: true, showPrintMargin: false}}
      />
    </div>
  )
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Ace)
