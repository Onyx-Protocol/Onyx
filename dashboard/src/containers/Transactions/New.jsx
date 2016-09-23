import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Transaction/Form'

const type = 'transaction'

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
