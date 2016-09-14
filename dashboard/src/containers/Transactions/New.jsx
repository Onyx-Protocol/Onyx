import { reduxForm } from 'redux-form'
import actions from '../../actions'
import Form from '../../components/Transaction/Form'

const type = "transaction"
const formName = 'NewTransactionForm'

const mapStateToProps = () => ({})

const mapDispatchToProps = (dispatch) => {
  return {
    submitForm: (data) => dispatch(actions[type].submitForm(data))
  }
}

const config = {
  form: formName,
  fields: [
    'actions[].type',
    'actions[].account_alias',
    'actions[].asset_alias',
    'actions[].amount',
    'actions[].control_program',
    'actions[].transaction_id',
    'actions[].position',
    'actions[].reference_data',
    'reference_data',
  ],
  initialValues: {
    reference_data: '{}',
  },
}

export default reduxForm(config, mapStateToProps, mapDispatchToProps)(Form)
