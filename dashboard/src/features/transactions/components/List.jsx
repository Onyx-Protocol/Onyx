import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem/ListItem'
import { Link } from 'react-router'

const type = 'transaction'

const actions = [
  <Link
    className='btn btn-link'
    key='consumers'
    to='transactions/consumers'
  >
    Consumers
  </Link>
]

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, { actions }),
  BaseList.mapDispatchToProps(type)
)
