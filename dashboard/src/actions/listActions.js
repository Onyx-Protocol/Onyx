import chain from '../chain'
import { context } from '../utility/environment'
import actionCreator from './actionCreator'

export default function(type, options = {}) {
  const incrementPage = actionCreator(`INCREMENT_${type.toUpperCase()}_PAGE`)
  const decrementPage = actionCreator(`DECREMENT_${type.toUpperCase()}_PAGE`)
  const appendPage = actionCreator(`APPEND_${type.toUpperCase()}_PAGE`, param => ({ param }) )
  const resetPage = actionCreator(`RESET_${type.toUpperCase()}_PAGE`)
  const updateQuery = actionCreator(`UPDATE_${type.toUpperCase()}_QUERY`, param => ({ param }) )

  const submitQuery = function(query) {
    return function(dispatch, getState) {
      dispatch(updateQuery(query))
      dispatch(resetPage())
      dispatch(fetchPage()) // FIXME: do this in fewer steps?
    }
  }

  const fetchPage = function() {
    const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)
    return function(dispatch, getState) {
      let pageCount = getState()[type].pages.length
      let latestPage = getState()[type].pages[pageCount - 1]
      let promise, filter

      if (latestPage) {
        if (!latestPage.last_page) {
          promise = latestPage.next(context)
        } else {
          return
        }
      } else {
        let params = {}
        if (getState()[type].currentQuery) {
          filter = getState()[type].currentQuery
          params.filter = filter
        }
        if (getState()[type].sumBy) {
          params.sum_by = getState()[type].sumBy.split(",")
        } else if (options.defaultSumBy) {
          params.sum_by = options.defaultSumBy()
        }
        promise = chain[className].query(context, params)
      }

      promise.then((param) => {
        if (param.items.length == 0) {
          return
        }

        dispatch(appendPage(param))
      }).catch((err) => {
        console.log(err)
        if (options.defaultKey && filter.indexOf(" ") < 0 && filter.indexOf("=") < 0) {
          dispatch(submitQuery(`${options.defaultKey}='${filter}'`))
        }
      })
    }
  }

  return {
    incrementPage: incrementPage,
    decrementPage: decrementPage,
    appendPage: appendPage,
    resetPage: resetPage,
    updateQuery: updateQuery,
    displayNextPage: function() {
      return function(dispatch, getState) {
        let currentPage = getState()[type].currentPage
        if (currentPage + 1 >= getState()[type].pages.length) {
          dispatch(fetchPage())
        } else {
          dispatch(incrementPage())
        }
      }
    },
    submitQuery: submitQuery
  }
}
