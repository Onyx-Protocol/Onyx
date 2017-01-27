import React from 'react'
import snakeCase from 'lodash.snakecase'

const dashboardComponent = (WrappedComponent) => {
  class Wrapped extends React.Component {
    render() {
      const className = snakeCase(WrappedComponent.displayName)
      return(<WrappedComponent {...this.props} className={className}/>)
    }
  }
  return Wrapped
}

export default dashboardComponent
