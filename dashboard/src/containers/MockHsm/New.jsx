import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/MockHsm/Form'

const type = "mockhsm"

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  Form
)
