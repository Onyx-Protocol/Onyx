import React from 'react'
import PageHeader from '../PageHeader/PageHeader'
import { ErrorBanner } from '../Common'

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      alias: ''
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange(event) {
    this.setState({alias: event.target.value})
  }

  handleSubmit(event) {
    event.preventDefault()
    this.props.submitForm({alias: this.state.alias}).catch((err) => {
      this.setState({error: err})
    })
  }

  render() {
    return(
      <div className='form-container'>
        <PageHeader title='New Mock HSM Key' />

        <form onSubmit={this.handleSubmit}>
          <div className='form-group'>
            <label>Alias</label>
            <input
              className='form-control'
              type='text'
              placeholder='Alias'
              autoFocus='autofocus'
              value={this.state.alias}
              onChange={this.handleChange} />
          </div>

          {this.state.error &&
            <ErrorBanner
              title='Error creating key'
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
