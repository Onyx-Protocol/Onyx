import { connect } from 'react-redux'
import AutocompleteField, {mapStateToProps, mapDispatchToProps} from './AutocompleteField'

const type = 'account'

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type)
)(AutocompleteField)
