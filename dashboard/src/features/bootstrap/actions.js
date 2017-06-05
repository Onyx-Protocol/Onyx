import { push } from 'react-router-redux'

export const showLogin =  () => push('/login')

export const showRoot = () => push('/transactions')

export const showConfiguration = () => push('/configuration')
