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
    'generator_access_token',
    'blockchain_id'
  ]
}

export default reduxForm(
  config,
  mapStateToProps,
  mapDispatchToProps
)(Index)
