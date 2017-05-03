import * as React from 'react'
import { connect } from 'react-redux'
import { getIdList as getTemplateIds } from '../../templates/selectors'
import { selectTemplate } from '../actions'
import { getSelectedTemplateId } from '../selectors'
import { AppState } from '../../app/types'

function SelectTemplateUnconnected(props: { templateIDs: string[], selectedTemplate: string, selectTemplate: (string)=>void }) {
  let options = props.templateIDs.map(templateID => {
    return <option key={templateID} value={templateID}>{templateID}</option>
  })
  return <select value={props.selectedTemplate} className="form-control" name="selectTemplate" onChange={(e)=>props.selectTemplate(e.target.value)}>
    {options}
  </select>
}

function mapStateToTemplateProps(state: AppState) {
  return { 
    templateIDs: getTemplateIds(state),
    selectedTemplate: getSelectedTemplateId(state)
  }
}

function mapDispatchToSelectTemplateProps(dispatch) {
  return {
    selectTemplate: (id) => {
      dispatch(selectTemplate(id))
    }
  }
}

const SelectTemplate = connect(
  mapStateToTemplateProps,
  mapDispatchToSelectTemplateProps
)(SelectTemplateUnconnected)

export default SelectTemplate