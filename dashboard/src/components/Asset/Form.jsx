import React from 'react';
import PageHeader from "../PageHeader"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      alias: "",
      xpubs: [],
      definition: "{}",
      quorum: 1,
      tags: "{}"
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange(event) {
    let newState = {
      alias: this.refs.alias.value,
      xpubs: this.refs.xpubs.value.split(","),
      quorum: parseInt(this.refs.quorum.value),
      definition: this.refs.definition.value,
      tags: this.refs.tags.value
    }
    this.setState(newState)
  }

  handleSubmit(event) {
    let request = Object.assign(this.state)
    request.tags = JSON.parse(request.tags)
    request.definition = JSON.parse(request.definition)
    this.props.submitForm(request)
  }

  render() {
    return(
      <div>
        <PageHeader title="New Asset" />

        <input
          ref="alias"
          className='form-control'
          type='text'
          placeholder="Alias"
          value={this.state.alias}
          onChange={this.handleChange} />
        <input
          ref="xpubs"
          className='form-control'
          type='text'
          placeholder="Xpubs (Quorum)"
          value={this.state.xpubs}
          onChange={this.handleChange} />
        <input
          ref="quorum"
          className='form-control'
          type='number'
          placeholder="Quorum"
          value={this.state.quorum}
          onChange={this.handleChange} />
        <textarea
          ref="definition"
          className='form-control'
          value={this.state.definition}
          onChange={this.handleChange} />
        <textarea
          ref="tags"
          className='form-control'
          value={this.state.tags}
          onChange={this.handleChange} />

        <button onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
