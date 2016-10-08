import chain from 'chain'
import { context, pageSize } from 'utility/environment'
import actionCreator from './actionCreator'
import { push } from 'react-router-redux'

export default function(type, options = {}) {
  const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)
  const listPath  = options.listPath || `/${type}s`

  const receivedItems = actionCreator(`RECEIVED_${type.toUpperCase()}_ITEMS`, param => ({ param }) )
  const appendPage = actionCreator(`APPEND_${type.toUpperCase()}_PAGE`, param => ({ param }) )
  const updateQuery = actionCreator(`UPDATE_${type.toUpperCase()}_QUERY`, param => ({ param }) )
  const didLoadAutocomplete = actionCreator(`DID_LOAD_${type.toUpperCase()}_AUTOCOMPLETE`)

  const deleteItemSuccess = actionCreator(`DELETE_${type.toUpperCase()}`, id => ({ id }))
  const deleteItem = (id) => {
    return (dispatch) => chain[className].delete(context(), id)
      .then(() => dispatch(deleteItemSuccess(id)))
      .catch(err => dispatch({type: 'ERROR', payload: err}))
  }

  const fetchItems = (params) => {
    const requiredParams = options.requiredParams || {}

    params = { ...params, ...requiredParams }

    return (dispatch) => {
      const promise = chain[className].query(context(), params)

      promise.then(
        (param) => dispatch(receivedItems(param))
      )

      return promise
    }
  }

  const fetchAll = function(stepCallback = () => {}) {
    return function(dispatch) {
      const fetchUntilLastPage = (next) => {
        return dispatch(fetchItems(next)).then((resp) => {
          stepCallback(resp)

          if (resp.last_page) {
            return resp
          } else {
            return fetchUntilLastPage(resp.next)
          }
        })
      }

      return fetchUntilLastPage({})
    }
  }

  const fetchQueryPage = function() {
    return function(dispatch, getState) {
      let latestResponse = getState()[type].listView.cursor
      let promise
      let filter = ''

      if (latestResponse && latestResponse.last_page) {
        return Promise.resolve({last: true})
      } else if (latestResponse.nextPage) {
        promise = latestResponse.nextPage(context())
      } else {
        let params = {}

        if (getState()[type].listView.query) {
          filter = getState()[type].listView.query
          params.filter = filter
        }

        if (getState()[type].listView.sumBy) {
          params.sum_by = getState()[type].listView.sumBy.split(',')
        }

        promise = dispatch(fetchItems(params))
      }

      return promise.then(
        (response) => dispatch(appendPage(response))
      ).catch(( err ) => {
        if (options.defaultKey && filter.indexOf('\'') < 0 && filter.indexOf('=') < 0) {
          dispatch(updateQuery(`${options.defaultKey}='${filter}'`))
          dispatch(fetchQueryPage())
        } else {
          return dispatch({type: 'ERROR', payload: err})
        }
      })
    }
  }

  const pushPage = (pageNumber) => push({
    pathname: listPath,
    query: {
      page: pageNumber
    }
  })

  const getPageSlice = (page, getState) => {
    const pageStart = page * pageSize
    return getState()[type].listView.itemIds.slice(pageStart, pageStart + pageSize)
  }

  return {
    appendPage,
    updateQuery,
    fetchItems,
    deleteItem,
    fetchAll,
    pushPage,
    fetchUntilPage: function(pageNumber) {
      return (dispatch, getState) => {
        const fullPage = () => {
          const currentPage = getPageSlice(pageNumber, getState)
          return currentPage.length == pageSize
        }

        if (fullPage()) return Promise.resolve({})

        const fillPageOrLast = () =>
          dispatch(fetchQueryPage()).then((resp) => {
            if (resp && resp.type == 'ERROR') return

            if (resp && resp.last) {
              return Promise.resolve(resp)
            } else if (!fullPage()) {
              return dispatch(fillPageOrLast)
            }
          })

        return dispatch(fillPageOrLast)
      }
    },
    didLoadAutocomplete,
  }
}
