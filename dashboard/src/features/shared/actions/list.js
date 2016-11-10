import chain from '_chain'
import { context, pageSize } from 'utility/environment'
import { push, replace } from 'react-router-redux'

export default function(type, options = {}) {
  const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)
  const listPath  = options.listPath || `/${type}s`

  const receive = (param) => ({
    type: `RECEIVED_${type.toUpperCase()}_ITEMS`,
    param,
  })

  // Dispatch a single request for the specified query, and persist the
  // results to the default item store
  const fetchItems = (params) => {
    const requiredParams = options.requiredParams || {}

    params = { ...params, ...requiredParams }

    return (dispatch) => {
      const promise = chain[className].query(context(), params)

      promise.then(
        (param) => dispatch(receive(param))
      )

      return promise
    }
  }

  // Fetch all items up to the specified page, and persist the results to
  // the filter-specific store
  const fetchPage = (query, pageNumber = 1, options = {}) => {
    const getPageSlice = (list, page) => {
      const pageStart = page * pageSize
      return (list.itemIds || []).slice(pageStart, pageStart + pageSize)
    }

    const listId =  query.filter || ''
    pageNumber = parseInt(pageNumber || 1)

    return (dispatch, getState) => {
      const getFilterStore = () => getState()[type].queries[listId] || {}

      const fullPage = () => {
        // Return early to load all pages if -1 is passed
        if (pageNumber == -1) return

        const list = getFilterStore()
        const currentPage = getPageSlice(list, pageNumber)
        return currentPage.length == pageSize
      }

      if (!options.refresh && fullPage()) return Promise.resolve({})

      const fetchNextPage = () =>
        dispatch(_load(query, getFilterStore(), options)).then((resp) => {
          if (!resp || resp.type == 'ERROR') return

          if (resp && resp.last) {
            return Promise.resolve(resp)
          } else if (!fullPage()) {
            options.refresh = false
            return dispatch(fetchNextPage)
          }
        })

      return dispatch(fetchNextPage)
    }
  }

  // Fetch and persist all records of the current object type
  const fetchAll = () => {
    return fetchPage('', -1)
  }

  const _load = function(query = {}, list = {}, requestOptions) {
    return function(dispatch) {
      let latestResponse = list.cursor || {}
      let promise
      let refresh = requestOptions.refresh || false
      let filter = query.filter || ''

      if (!refresh && latestResponse && latestResponse.last_page) {
        return Promise.resolve({last: true})
      } else if (!refresh && latestResponse.nextPage) {
        promise = latestResponse.nextPage(context())
        promise.then(resp => dispatch(receive(resp)))
      } else {
        let params = {}

        if (query.filter) params.filter = filter
        if (query.sum_by) params.sum_by = query.sum_by.split(',')

        promise = dispatch(fetchItems(params))
      }

      return promise.then((response) => {
        return dispatch({
          type: `APPEND_${type.toUpperCase()}_PAGE`,
          param: response,
          refresh: refresh,
        })
      }).catch(err => {
        if (options.defaultKey && filter.indexOf('\'') < 0 && filter.indexOf('=') < 0) {
          dispatch(pushList({
            filter: `${options.defaultKey}='${query.filter}'`
          }, null, {replace: true}))
        } else {
          return dispatch({type: 'ERROR', payload: err})
        }
      })
    }
  }

  const deleteItem = (id, confirmMessage, deleteMessage) => {
    return (dispatch) => {
      if (!window.confirm(confirmMessage)) {
        return
      }

      chain[className].delete(context(), id)
        .then(() => dispatch({
          type: `DELETE_${type.toUpperCase()}`,
          id: id,
        })).then(() => dispatch({
          type: `DELETED_${type.toUpperCase()}`,
          message: deleteMessage,
        })).catch(err => dispatch({
          type: 'ERROR', payload: err
        }))
    }
  }

  const pushList = (query = {}, pageNumber, options = {}) => {
    if (pageNumber) {
      query = {
        ...query,
        page: pageNumber,
      }
    }

    const location = {
      pathname: listPath,
      query
    }

    if (options.replace) return replace(location)
    return push(location)
  }

  return {
    fetchItems,
    fetchPage,
    fetchAll,
    deleteItem,
    pushList,
    didLoadAutocomplete: {
      type: `DID_LOAD_${type.toUpperCase()}_AUTOCOMPLETE`
    },
  }
}
