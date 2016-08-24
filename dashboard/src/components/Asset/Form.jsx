import React from 'react'
import PageHeader from "../PageHeader/PageHeader"

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

  handleChange() {
    let newState = {
      alias: this.refs.alias.value,
      xpubs: this.refs.xpubs.value.split(","),
      quorum: parseInt(this.refs.quorum.value),
      definition: this.refs.definition.value,
      tags: this.refs.tags.value
    }
    this.setState(newState)
  }

  handleSubmit() {
    let request = Object.assign(this.state)
    request.tags = JSON.parse(request.tags)
    request.definition = JSON.parse(request.definition)
    this.props.submitForm(request)
  }

  render() {
    return(
      <div className='form-container'>
        <PageHeader title="New Asset" />

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
        <div className='form-group'>
          <label>Xpubs</label>
          <input
            ref="xpubs"
            className='form-control'
            type='text'
            placeholder="Xpubs (Quorum)"
            value={this.state.xpubs}
            onChange={this.handleChange} />
        </div>
        <div className='form-group'>
          <label>Quorum</label>
          <input
            ref="quorum"
            className='form-control'
            type='number'
            placeholder="Quorum"
            value={this.state.quorum}
            onChange={this.handleChange} />
        </div>
        <div className='form-group'>
          <label>Definition</label>
          <textarea
            ref="definition"
            className='form-control'
            value={this.state.definition}
            onChange={this.handleChange} />
        </div>
        <div className='form-group'>
          <label>Tags</label>
          <textarea
            ref="tags"
            className='form-control'
            value={this.state.tags}
            onChange={this.handleChange} />
        </div>

        <button className='btn btn-primary' onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
