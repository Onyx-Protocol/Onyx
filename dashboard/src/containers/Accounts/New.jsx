import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Account/Form'

const type = "account"

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
