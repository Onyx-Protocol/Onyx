import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import { ErrorBanner } from "../Common"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      form: {
        alias: "",
        xpubs: [],
        definition: "{}",
        quorum: 1,
        tags: "{}"
      }
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange() {
    this.setState({
      form: {
        alias: this.refs.alias.value,
        xpubs: this.refs.xpubs.value.split(","),
        quorum: parseInt(this.refs.quorum.value),
        definition: this.refs.definition.value,
        tags: this.refs.tags.value
      }
    })
  }

  handleSubmit(event) {
    event.preventDefault()

    let request = Object.assign({}, this.state.form)

    try {
      request.tags = JSON.parse(request.tags)
    } catch(err) {
      this.setState({error: "Tags must be a valid JSON object."})
      return
    }

    try {
      request.definition = JSON.parse(request.definition)
    } catch(err) {
      this.setState({error: "Definition must be a valid JSON object."})
      return
    }

    this.props.submitForm(request).catch(err => this.setState({error: err}))
  }

  render() {
    return(
      <div className='form-container'>
        <PageHeader title="New Asset" />

        <form onSubmit={this.handleSubmit}>
          <div className='form-group'>
            <label>Alias</label>
            <input
              ref="alias"
              className='form-control'
              type='text'
              placeholder="Alias"
              autoFocus="autofocus"
              value={this.state.form.alias}
              onChange={this.handleChange} />
          </div>
          <div className='form-group'>
            <label>Xpubs</label>
            <input
              ref="xpubs"
              className='form-control'
              type='text'
              placeholder="Xpubs (Quorum)"
              value={this.state.form.xpubs}
              onChange={this.handleChange} />
          </div>
          <div className='form-group'>
            <label>Quorum</label>
            <input
              ref="quorum"
              className='form-control'
              type='number'
              placeholder="Quorum"
              value={this.state.form.quorum}
              onChange={this.handleChange} />
          </div>
          <div className='form-group'>
            <label>Definition</label>
            <textarea
              ref="definition"
              className='form-control'
              value={this.state.form.definition}
              onChange={this.handleChange} />
          </div>
          <div className='form-group'>
            <label>Tags</label>
            <textarea
              ref="tags"
              className='form-control'
              value={this.state.form.tags}
              onChange={this.handleChange} />
          </div>

          {this.state.error &&
            <ErrorBanner
              title='Error creating asset'
              message={this.state.error.toString()}
            />
          }

          <button className='btn btn-primary'>Submit</button>
        </form>
      </div>
    )
  }


}

export default Form
