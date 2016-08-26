import { reduxForm } from 'redux-form'
import actions from '../../actions'
import Form from '../../components/Transaction/Form'

const type = "transaction"
const formName = 'NewTransactionForm'

const mapStateToProps = () => ({})

const mapDispatchToProps = (dispatch) => {
  return {
    submitForm: (data) => {
      dispatch(actions[type].submitForm(data))
    },
  }
}

const config = {
  form: formName,
  fields: [
    'actions[].type',
    'actions[].params.account_alias',
    'actions[].params.asset_alias',
    'actions[].params.amount',
    'actions[].params.control_program',
    'actions[].params.transaction_id',
    'actions[].params.position',
    'actions[].reference_data'
  ]
}

export default reduxForm(config, mapStateToProps, mapDispatchToProps)(Form)
