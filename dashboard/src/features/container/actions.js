import { push } from 'react-router-redux'

export const showLogin =  () => push('/login')

export const showRoot = () => push('/transactions')

export const showConfiguration = () => push('/configuration')

//   return (dispatch, getState) => {
//     // Need a default here, since locationBeforeTransitions gets cleared
//     // during logout.
//     let pathname = (getState().routing.locationBeforeTransitions || {}).pathname
//
//     if (pathname !== 'configuration') {
//       dispatch(push('/configuration'))
//     }
//   }
// }
