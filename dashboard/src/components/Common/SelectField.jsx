import React from 'react'

class SelectField extends React.Component {
  render() {
    const options = this.props.options
    const emptyLabel = this.props.emptyLabel || "Select one..."

    return(
      <div className='form-group'>
        <label>{this.props.title}</label>
        <select className='form-control'
          {...this.props.fieldProps}>
          <option>{emptyLabel}</option>
          {Object.keys(options).map((key) => <option value={key} key={key}>{options[key]}</option>) }
        </select>
      </div>
    )
  }
}

export default SelectField
