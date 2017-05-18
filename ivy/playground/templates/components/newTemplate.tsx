import * as React from 'react'
import { connect } from 'react-redux'

const NewTemplate = () => {
  return (
    <div className="dropdown">
      <button className="btn btn-primary dropdown-toggle" type="button" id="dropdownMenu1" data-toggle="dropdown" aria-haspopup="true" aria-expanded="true">
        <span className="glyphicon glyphicon-plus"></span>
        New
      </button>
    </div>
  )
}

export default connect()(NewTemplate)
