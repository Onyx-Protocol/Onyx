import React from 'react';
import PageHeader from "../PageHeader/PageHeader"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      actions: '[]'
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange(event) {
    let newState = {
      actions: this.refs.actions.value
    }
    this.setState(newState)
  }

  handleSubmit(event) {
    let request = Object.assign(this.state)
    request.actions = JSON.parse(request.actions)
    this.props.submitForm(request)
  }

  render() {
    return(
      <div>
        <PageHeader title="New Transaction" />

        <textarea
          ref="actions"
          className='form-control'
          value={this.state.actions}
          onChange={this.handleChange} />

        <button onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
