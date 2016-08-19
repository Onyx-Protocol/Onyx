import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Asset/Form'

const type = "asset"

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
