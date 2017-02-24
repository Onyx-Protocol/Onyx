import { reduxForm } from 'redux-form'
import { ErrorBanner, SubmitIndicator, TextField } from 'features/shared/components'
import pick from 'lodash/pick'
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
    if (data.generatorUrl && !data.blockchainId) {
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
        generatorUrl,
        generatorAccessToken,
        blockchainId
      },
      handleSubmit,
      submitting,
    } = this.props

    const typeChange = (event) => {
      const value = type.onChange(event).value

      if (value != 'join') {
        generatorUrl.onChange('')
        generatorAccessToken.onChange('')
        blockchainId.onChange('')
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
                    value='new'
                    disabled={this.props.production} />
              <div className={`${styles.choice} ${styles.new} ` + (this.props.production ? styles.disabled : '')}>
                <span className={styles.choice_title}>Create new blockchain network</span>

                {this.props.production &&
                  <p>This core is compiled for production. Use <code>corectl</code> to configure as a generator.</p>
                }
                {!this.props.production &&
                  <p>Start a new blockchain network with this Chain Core as the block generator.</p>
                }
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
                fieldProps={generatorUrl} />
              <TextField
                title='Blockchain ID'
                placeholder='896a800000000000000'
                fieldProps={blockchainId} />
              <TextField
                title={[
                  'Network Access Token',
                  <a href='http://www.chain.com/docs/core/learn-more/authentication' target='_blank'>
                    <small className={styles.infoLink}>
                      <span className='glyphicon glyphicon-info-sign'></span>
                    </small>
                  </a>]}
                placeholder='token-id:9e5f139755366add8c76'
                fieldProps={generatorAccessToken} />

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

const mapStateToProps = state => ({
  production: state.core.production,
})

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.configuration.submitConfiguration(data))
})

const config = {
  form: 'coreConfigurationForm',
  fields: [
    'type',
    'generatorUrl',
    'generatorAccessToken',
    'blockchainId'
  ]
}

export default reduxForm(
  config,
  mapStateToProps,
  mapDispatchToProps
)(Index)
