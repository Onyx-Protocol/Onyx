import React from 'react'
import PageHeader from "../PageHeader/PageHeader"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      alias: "",
      type: "",
      unspents: false,
      filter: ""
    }
    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
  }

  handleChange() {
    let newState = {
      alias: this.refs.alias.value,
      filter: this.refs.filter.value,
      type: this.refs.type.value
    }
    this.setState(newState)
  }

  handleSubmit() {
    this.props.submitForm(this.state)
  }

  render() {
    return(
      <div className='form-container'>
        <PageHeader title="New Index" />

        <div className='form-group'>
          <label>Alias</label>
          <input
            ref="alias"
            className='form-control'
            type='text'
            placeholder='Alias'
            autoFocus="autofocus"
            value={this.state.alias}
            onChange={this.handleChange} />
        </div>
        <div className='form-group'>
          <label>Type</label>
          <select className='form-control'
            ref="type"
            value={this.state.type}
            onChange={this.handleChange}>
              <option value="transaction">Transaction</option>
              <option value="balance">Balance</option>
              <option value="asset">Asset</option>
          </select>
        </div>
        <div className='form-group'>
          <label>Filter</label>
          <input
            ref="filter"
            className='form-control'
            type='text'
            placeholder='Filter'
            value={this.state.filter}
            onChange={this.handleChange} />
        </div>

        <button className='btn btn-primary' onClick={this.handleSubmit}>Submit</button>
      </div>
    )
  }


}

export default Form
