import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import { ErrorBanner } from "../Common"
import ActionItem from './FormActionItem'

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {fields: []}

    this.submitWithValidation = this.submitWithValidation.bind(this)
    this.openReferenceData = this.openReferenceData.bind(this)
    this.addActionItem = this.addActionItem.bind(this)
    this.removeActionItem = this.removeActionItem.bind(this)

    this.addActionItem()
    this.addActionItem()
  }

  addActionItem() {
    this.props.fields.actions.addField({reference_data: '{}'})
  }

  removeActionItem() {
    this.props.fields.actions.removeField()
  }

  submitWithValidation(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err.message}))
    })
  }

  openReferenceData() {
    this.setState({referenceDataOpen: true})
  }

  render() {
    const {
      fields: { actions, reference_data },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <div className="form-container">
        <PageHeader title="New Transaction" />

        <form onSubmit={handleSubmit(this.submitWithValidation)} >
          <div className='form-group'>

            {!actions.length && <div className='well'>Add actions to build a transaction</div>}

            {actions.map((action, index) => <ActionItem key={index} index={index} fieldProps={action} />)}

            <button type="button" className="btn btn-link" onClick={this.addActionItem} >
              + Add Action
            </button>

            {actions.length > 0 &&
              <button type="button" className="btn btn-link" onClick={this.removeActionItem}>
                - Remove Action
              </button>
            }
          </div>

          <hr />

          {this.state.referenceDataOpen &&
            <TextareaField title='Transaction-level reference data' fieldProps={reference_data} />
          }
          {!this.state.referenceDataOpen &&
            <button type="button" className="btn btn-link" onClick={this.openReferenceData}>
              Add transaction-level reference data
            </button>
          }

          <hr />

          <p>
            Submitting builds a transaction template, signs the template with
             the Mock HSM, and submits the fully signed template to the blockchain.
          </p>

          {error && <ErrorBanner
            title="There was a problem submitting your transaction:"
            message={error}/>}

          <button type="submit" className="btn btn-primary" disabled={submitting}>
            Submit Transaction
          </button>
        </form>
      </div>
    )
  }
}

export default Form
