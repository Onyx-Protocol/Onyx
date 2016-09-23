import { reduxForm } from 'redux-form'
import actions from '../actions'
import Index from '../components/Config/Index'

const type = 'core'
const formName = 'CoreConfigurationForm'

const mapStateToProps = () => ({})

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions[type].submitConfiguration(data))
})

const config = {
  form: formName,
  fields: [
    'is_generator',
    'generator_url',
    'initial_block_hash'
  ]
}

export default reduxForm(
  config,
  mapStateToProps,
  mapDispatchToProps
)(Index)
