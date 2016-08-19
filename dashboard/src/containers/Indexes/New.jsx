import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Index/Form'

const type = "index"

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
