import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import { TextField, SelectField, ErrorBanner } from '../Common'
import { reduxForm } from 'redux-form'

const fields = [ 'alias', 'type', 'filter', 'sum_by[]' ]

const indexTypes = {
  transaction: "Transaction",
  balance: "Balance",
  asset: "Asset"
}

class Form extends React.Component {
  constructor(props) {
    super(props)

    this.state = { showSumBy: false }

    this.submitWithErrors = this.submitWithErrors.bind(this)
  }

  submitWithErrors(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err.message}))
    })
  }

  render() {
    const {
      fields: { alias, type, filter, sum_by },
      error,
      handleSubmit,
      submitting
    } = this.props

    let typeOnChange = event => {
      let showSumBy = type.onChange(event).value === 'balance'
      this.setState({ showSumBy: showSumBy })

      if (!showSumBy) {
        for (let i = 0; i < sum_by.length; i++) { sum_by.removeField() }
      } else {
        sum_by.addField()
      }
    }
    let typeProps = Object.assign({}, type, {onChange: typeOnChange})

    return(
      <div className='form-container'>
        <PageHeader title="New Index" />

        <form onSubmit={handleSubmit(this.submitWithErrors)}>
          <TextField title="Alias" fieldProps={alias} />
          <SelectField title="Type" emptyLabel="Select index type..." options={indexTypes} fieldProps={typeProps} />
          <TextField title="Filter" fieldProps={filter} />

          {this.state.showSumBy && <div className='form-group'>
            {sum_by.map((item, index) => <TextField title="Sum By" key={`sum-by-${index}`} fieldProps={item} />)}

            <button type="button" className="btn btn-link" onClick={sum_by.addField} >
              + Add sum field
            </button>

            {sum_by.length > 0 &&
              <button type="button" className="btn btn-link" onClick={() => sum_by.removeField()}>
                - Remove sum field
              </button>
            }
          </div>}

          {error && <ErrorBanner
            title="There was a problem creating your index:"
            message={error}/>}

          <button className='btn btn-primary'>Submit</button>
        </form>
      </div>
    )
  }


}

export default reduxForm({
  form: 'newIndexForm',
  fields
})(Form)
