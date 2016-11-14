import { reduxForm } from 'redux-form'
import { TextField } from 'components/Common'
import { ErrorBanner, SubmitIndicator } from 'features/shared/components'
import pick from 'lodash.pick'
import actions from 'actions'
import React from 'react'
import styles from './Index.scss'

class Index extends React.Component {
  constructor(props) {
    super(props)

    this.submitWithValidation = this.submitWithValidation.bind(this)
  }

  showNewFields() {
    return this.props.fields.type.value === 'new'
  }

  showJoinFields() {
    return this.props.fields.type.value === 'join'
  }

  showTestnetFields() {
    return this.props.fields.type.value === 'testnet'
  }

  submitWithValidation(data) {
    if (data.generator_url && !data.blockchain_id) {
      return new Promise((_, reject) => reject({
        _error: 'You must specify a blockchain ID to connect to a network'
      }))
    }

    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({type: err}))
    })
  }

  render() {
    const {
      fields: {
        type,
        generator_url,
        generator_access_token,
        blockchain_id
      },
      handleSubmit,
      submitting,
    } = this.props

    const typeChange = (event) => {
      const value = type.onChange(event).value

      if (value != 'join') {
        generator_url.onChange('')
        generator_access_token.onChange('')
        blockchain_id.onChange('')
      }
    }

    const typeProps = {
      ...pick(type, ['name', 'value', 'checked', 'onBlur', 'onFocus']),
      onChange: typeChange
    }

    let configSubmit = [
      (type.error && <ErrorBanner
        key='configError'
        title='There was a problem configuring your core'
        error={type.error}
      />),
      <button
        key='configSubmit'
        type='submit'
        className={`btn btn-primary btn-lg ${styles.submit}`}
        disabled={submitting}>
          &nbsp;{this.showNewFields() ? 'Create' : 'Join'} network
      </button>
    ]

    if (submitting) {
      configSubmit.push(<SubmitIndicator
        text={this.showNewFields() ? 'Creating network...' : 'Joining network...'}
      />)
    }

    return (
      <form onSubmit={handleSubmit(this.submitWithValidation)}>
        <h2 className={styles.title}>Configure Chain Core</h2>

        <div className={styles.choices}>
          <div className={styles.choice_wrapper}>
            <label>
              <input className={styles.choice_radio_button}
                    type='radio'
                    {...typeProps}
                    value='new' />
              <div className={`${styles.choice} ${styles.new}`}>
                <span className={styles.choice_title}>Create new blockchain network</span>

                <p>
                  Start a new blockchain network with this Chain Core as the block generator.
                </p>
              </div>
            </label>
          </div>

          <div className={styles.choice_wrapper}>
            <label>
              <input className={styles.choice_radio_button}
                    type='radio'
                    {...typeProps}
                    value='join' />
              <div className={`${styles.choice} ${styles.join}`}>
                <span className={styles.choice_title}>Join existing blockchain network</span>

                <p>
                  Connect this Chain Core to an existing blockchain network.
                </p>
              </div>
            </label>
          </div>

          <div className={styles.choice_wrapper}>
            <label>
              <input className={styles.choice_radio_button}
                    type='radio'
                    {...typeProps}
                    value='testnet' />
              <div className={`${styles.choice} ${styles.testnet}`}>
                  <span className={styles.choice_title}>Join the Chain Testnet</span>

                  <p>
                    Connect this Chain Core to the Chain Testnet. <strong>Data will be reset every week.</strong>
                  </p>
              </div>
            </label>
          </div>
        </div>

        <div className={styles.choices}>
          <div>
            {this.showNewFields() && <span className={styles.submitWrapper}>{configSubmit}</span>}
          </div>

          <div>
            {this.showJoinFields() && <div className={styles.joinFields}>
              <TextField
                title='Block Generator URL'
                placeholder='https://<block-generator-host>'
                fieldProps={generator_url} />
              <TextField
                title='Network Access Token'
                placeholder='token-id:9e5f139755366add8c76'
                fieldProps={generator_access_token} />
              <TextField
                title='Blockchain ID'
                placeholder='896a800000000000000'
                fieldProps={blockchain_id} />

              {configSubmit}
            </div>}
          </div>

          <div>
            {this.showTestnetFields() &&
              <span className={styles.submitWrapper}>{configSubmit}</span>
            }
          </div>
        </div>
      </form>
    )
  }
}

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.configuration.submitConfiguration(data))
})

const config = {
  form: 'coreConfigurationForm',
  fields: [
    'type',
    'generator_url',
    'generator_access_token',
    'blockchain_id'
  ]
}

export default reduxForm(
  config,
  () => {},
  mapDispatchToProps
)(Index)
