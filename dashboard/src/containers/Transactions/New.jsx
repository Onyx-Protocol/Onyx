import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../features/transactions/components/New'

const type = 'transaction'

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
