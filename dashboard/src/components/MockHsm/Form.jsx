import React from 'react';
import PageHeader from "../PageHeader"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      alias: ""
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange(event) {
    let newState = {
      alias: this.refs.alias.value
    }
    this.setState(newState)
  }

  handleSubmit(event) {
    this.props.submitForm(this.state)
  }

  render() {
    return(
      <div>
        <PageHeader title="New Key" />

        <input
          ref="alias"
          className='form-control'
          type='text'
          placeholder="Alias"
          value={this.state.alias}
          onChange={this.handleChange} />

        <button onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
