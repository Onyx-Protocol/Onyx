import classNames from 'classnames'
import snakeCase from 'lodash/snakecase'

const dashboardClasses = (owner, ...args) => {
  const coreName = owner ? snakeCase(owner.constructor.name) : false
  return classNames(coreName, args)
}

export default dashboardClasses
