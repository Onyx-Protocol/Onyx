import React from 'react'
import { TextField, ErrorBanner } from "../Common"
import styles from './Index.scss'

export default class Index extends React.Component {
  constructor(props) {
    super(props)
    this.state = { showFields: false }

    this.submitWithValidation = this.submitWithValidation.bind(this)
  }

  componentWillReceiveProps(nextProps) {
    console.log(nextProps)
    this.setState({ showFields: nextProps.fields.is_generator.value === 'false'})
  }

  submitWithValidation(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err.message}))
    })
  }

  render() {
    const {
      fields: {
        is_generator,
        generator_url,
        initial_block_hash
      },
      error,
      handleSubmit,
      submitting
    } = this.props

    let createNewIcon = require('../../images/config/create-new.svg')
    let joinExistingIcon = require('../../images/config/join-existing.svg')

    let fields
    if (this.state.showFields) {
      fields = [
        <TextField key="generator_url"
          title="Remote Generator URL"
          placeholder="https://:<password>@<host>"
          fieldProps={generator_url} />,
        <TextField key="initial_block_hash"
          title="Initial Block Hash"
          placeholder="T83340000000000"
          fieldProps={initial_block_hash} />
      ]
    }

    return (
      <form onSubmit={handleSubmit(this.submitWithValidation)}>
        <h2 className={styles.title}>Select how you would like to set up your blockchain</h2>
        <h3 className={styles.subtitle}>You can change your configuration at a later time</h3>
        <div className="row">
          <div className="col-sm-4">
            <label className={styles.choice_wrapper}>
              <input className={styles.choice_radio_button}
                    type="radio"
                    {...is_generator}
                    value='false'
                    checked={is_generator.value === 'false'} />
              <div className={styles.choice}>
                <img src={joinExistingIcon} />
                <span className={styles.choice_title}>Join existing blockchain network</span>

                <p>
                  Connect this Chain Core to an existing blockchain network
                </p>
              </div>
            </label>

            {fields}
          </div>

          <div className="col-sm-4">
            <label className={styles.choice_wrapper}>
              <input className={styles.choice_radio_button}
                    type="radio"
                    {...is_generator}
                    value='true'
                    checked={is_generator.value === 'true'} />
              <div className={styles.choice}>
                <img src={createNewIcon} />
                <span className={styles.choice_title}>Create new blockchain network</span>

                <p>
                  Start a new blockchain network with this Chain Core as the block generator.
                </p>
              </div>
            </label>
          </div>
        </div>

        <div className="row">
          <div className="col-sm-4">
            {error && <ErrorBanner
              title="There was a problem configuring your core:"
              message={error}/>}

            <button type="submit" className={`btn btn-primary btn-lg ${styles.submit}`} disabled={submitting}>
              <span className="glyphicon glyphicon-arrow-right" />
              &nbsp;Set up Chain Core
            </button>
          </div>
        </div>
      </form>
    )
  }
}
