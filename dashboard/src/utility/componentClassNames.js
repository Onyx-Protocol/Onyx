import React from 'react'
import classNames from 'classnames'

const componentClassNames = (owner, ...args) => {
  if (!React.Component.prototype.isPrototypeOf(owner))
    throw new Error('Component must descend from React.Component')

  const coreName = 'component-' + owner.constructor.name
  return classNames(coreName, args)
}

export default componentClassNames
