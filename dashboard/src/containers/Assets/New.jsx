import { mapStateToProps, mapDispatchToProps, connect } from '../Base/New'
import Form from '../../components/Asset/Form'

const type = "asset"

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
