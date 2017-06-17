import { chainClient } from 'utility/environment'
import { pageSize } from 'utility/environment'
import { push, replace } from 'react-router-redux'
import { isEmpty } from 'lodash'

export default function(type, options = {}) {
  const listPath  = options.listPath || `/${type}s`
  const clientApi = () => options.clientApi ? options.clientApi() : chainClient()[`${type}s`]

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
      const promise = clientApi().query(params)

      promise.then(
        (resp) => dispatch(receive(resp))
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
      const latestResponse = list.cursor || null
      const refresh = requestOptions.refresh || false

      if (!refresh && latestResponse && latestResponse.lastPage) {
        return Promise.resolve({last: true})
      }

      let promise
      const filter = query.filter || ''

      if (!refresh && latestResponse) {
        let responsePage
        promise = latestResponse.nextPage()
          .then(resp => {
            responsePage = resp
            return dispatch(receive(responsePage))
          }).then(() =>
            responsePage
          )
      } else {
        const params = {}
        if (query.filter) params.filter = filter
        if (query.sumBy) params.sumBy = query.sumBy.split(',')

        promise = dispatch(fetchItems(params))
      }

      return promise.then((response) => {
        return dispatch({
          type: `APPEND_${type.toUpperCase()}_PAGE`,
          param: response,
          refresh: refresh,
        })
      }).catch(err => {
        if (options.defaultKey && !isEmpty(filter) && filter.indexOf('\'') < 0 && filter.indexOf('=') < 0) {
          dispatch(pushList({
            filter: `${options.defaultKey}='${filter}'`
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

      clientApi().delete(id)
        .then(() => dispatch({
          type: `DELETE_${type.toUpperCase()}`,
          id: id,
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
