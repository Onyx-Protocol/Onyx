import * as React from 'react'
import { connect } from 'react-redux'

require('../util/ivymode.js')
import app from '../../app'
import Ace from './ace'
import ErrorAlert from './errorAlert'
import Load from './load'
import Save from './save'
import Instructions from './instructions'
import { getTemplate } from '../selectors'
import { isError } from '../util'

const mapStateToProps = (state) => {
  return {
    template: getTemplate(state)
  }
}

const Editor = ({ template }) => {
  return (
    <div>
      <div className="panel panel-default">
        <div className="panel-heading clearfix">
          <h1 className="panel-title pull-left">Template Editor</h1>
          <table className="pull-right"><tbody><tr>
            <td style={{width: 300, paddingRight: 10}}><Load /></td>
            <Save />{/* Save returns a <td> element */}
          </tr></tbody></table>
        </div>
        <Ace source={template.source} />
      </div>
      { isError(template) ? <ErrorAlert errorMessage={template.message} /> : <Instructions />}
    </div>
  )
}

export default connect(
  mapStateToProps,
)(Editor)