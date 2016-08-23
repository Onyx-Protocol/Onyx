import React from 'react'
import PageHeader from "../PageHeader/PageHeader"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      alias: ""
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange() {
    let newState = {
      alias: this.refs.alias.value
    }
    this.setState(newState)
  }

  handleSubmit() {
    this.props.submitForm(this.state)
  }

  render() {
    return(
      <div className='form-container'>
        <PageHeader title="New Mock HSM Key" />

        <div className='form-group'>
          <label>Alias</label>
          <input
            ref="alias"
            className='form-control'
            type='text'
            placeholder="Alias"
            autoFocus="autofocus"
            value={this.state.alias}
            onChange={this.handleChange} />
        </div>

        <button className='btn btn-primary' onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
