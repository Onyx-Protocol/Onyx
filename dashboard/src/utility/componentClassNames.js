import React from 'react'
import classNames from 'classnames'

const componentClassNames = (owner, ...args) => {
  if (!React.Component.prototype.isPrototypeOf(owner)) {
    throw new Error('Component class must descend from React.Component')
  }

  return classNames(owner.constructor.name, args)
}

export default componentClassNames
