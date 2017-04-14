import React from 'react'
import FieldLabel from './FieldLabel/FieldLabel'
import pick from 'lodash/pick'
import ReactMarkdown from 'react-markdown'

const SELECT_FIELD_PROPS = [
  'value',
  'onBlur',
  'onChange',
  'onFocus',
]

class SelectField extends React.Component {
  render() {
    const options = this.props.options
    const emptyLabel = this.props.emptyLabel || 'Select one...'
    const valueKey = this.props.valueKey || 'value'
    const labelKey = this.props.labelKey || 'label'

    const fieldProps = pick(this.props.fieldProps, SELECT_FIELD_PROPS)
    const {touched, error} = this.props.fieldProps

    return(
      <div className='form-group'>
        {this.props.title && <FieldLabel>{this.props.title}</FieldLabel>}
        <select
          className='form-control' {...fieldProps}
          autoFocus={!!this.props.autoFocus}>
          {!this.props.skipEmpty && <option value=''>{emptyLabel}</option>}

          {options.map((option) =>
            <option value={option[valueKey]} key={option[valueKey]}>
              {option[labelKey]}
            </option>)}
        </select>

        {touched && error && <span className='text-danger'><strong>{error}</strong></span>}
        {this.props.hint && <span className='help-block'><ReactMarkdown source={this.props.hint} /></span>}
      </div>
    )
  }
}

export default SelectField
