import React from 'react'
import { BaseUpdate, FormContainer, FormSection, JsonField, NotFound } from 'features/shared/components'
import { reduxForm } from 'redux-form'

class Form extends React.Component {
  constructor(props) {
    super(props)

    this.submitWithErrors = this.submitWithErrors.bind(this)

    this.state = {}
  }

  submitWithErrors(data) {
    return this.props.submitForm(data, this.props.item.id).catch(err => {
      throw {_error: err}
    })
  }

  componentDidMount() {
    this.props.fetchItem(this.props.params.id).then(resp => {
      if (resp.items.length == 0) {
        this.setState({notFound: true})
      }
    })
  }

  render() {
    if (this.state.notFound) {
      return <NotFound />
    }
    const item = this.props.item

    if (!item) {
      return <div>Loading...</div>
    }

    const {
      fields: { tags },
      error,
      handleSubmit,
      submitting
    } = this.props

    const title = <span>
      {'Edit asset tags '}
      <code>{item.alias ? item.alias :item.id}</code>
    </span>

    // TODO: add link to documentation to further educate on tags and updating tags
    const tagChangeWarning = <p>
      Updated tags will only apply to new transactions. Existing transactions
      will reflect the tags that were present when the transaction was created.
    </p>

    const tagsString = Object.keys(item.tags).length === 0 ? '{\n\t\n}' : JSON.stringify(item.tags, null, 1)
    const tagLines = tagsString.split(/\r\n|\r|\n/).length
    let JsonFieldHeight

    if (tagLines < 5) {
      JsonFieldHeight = '80px'
    } else if (tagLines < 20) {
      JsonFieldHeight = `${tagLines * 17}px`
    } else {
      JsonFieldHeight = '340px'
    }

    return <FormContainer
      error={error}
      label={title}
      onSubmit={handleSubmit(this.submitWithErrors)}
      submitting={submitting} >

      <FormSection title='Asset Tags'>
        {tagChangeWarning}
        <JsonField
          height={JsonFieldHeight}
          fieldProps={tags} />
      </FormSection>
    </FormContainer>
  }
}

const mapStateToProps = (state, ownProps) => ({
  item: state.asset.items[ownProps.params.id]
})

const initialValues = (state, ownProps) => {
  const item = state.asset.items[ownProps.params.id]
  if (item) {
    const tags = Object.keys(item.tags).length === 0 ? '{\n\t\n}' : JSON.stringify(item.tags, null, 1)
    return {
      initialValues: {
        tags: tags
      }
    }
  }
  return {}
}

const updateForm = reduxForm({
  form: 'updateAssetForm',
  fields: ['tags'],
  validate: values => {
    const errors = {}

    const jsonFields = ['tags']
    jsonFields.forEach(key => {
      const fieldError = JsonField.validator(values[key])
      if (fieldError) { errors[key] = fieldError }
    })

    return errors
  }
}, initialValues)(Form)

export default BaseUpdate.connect(
  mapStateToProps,
  BaseUpdate.mapDispatchToProps('asset'),
  updateForm
)
