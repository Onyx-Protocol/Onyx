import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Account/Form'

const type = "account"

const props = (state) => Object.assign({},
  mapStateToProps(type)(state),
  {
    // FIXME: load all keys
    mockhsmKeys: state.mockhsm.pages[0]
  }
)

export default connect(
  props,
  mapDispatchToProps(type),
  Form
)
