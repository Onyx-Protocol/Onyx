import chain from '../chain'
import { context, pageSize } from '../utility/environment'
import actionCreator from './actionCreator'

export default function(type, options = {}) {
  const incrementPage = actionCreator(`INCREMENT_${type.toUpperCase()}_PAGE`)
  const decrementPage = actionCreator(`DECREMENT_${type.toUpperCase()}_PAGE`)
  const appendPage = actionCreator(`APPEND_${type.toUpperCase()}_PAGE`, param => ({ param }) )
  const updateQuery = actionCreator(`UPDATE_${type.toUpperCase()}_QUERY`, param => ({ param }) )

  const getNextPageSlice = function(getState) {
    const pageStart = (getState()[type].listView.pageIndex + 1) * pageSize
    return getState()[type].listView.itemIds.slice(pageStart, pageStart + pageSize)
  }

  const fetchPage = function() {
    const className = options.className || type.charAt(0).toUpperCase() + type.slice(1)

    return function(dispatch, getState) {
      let latestResponse = getState()[type].listView.cursor
      let promise, filter

      if (latestResponse && latestResponse.last_page) {
        return new Promise.resolve()
      } else if (latestResponse.nextPage) {
        promise = latestResponse.nextPage(context)
      } else {
        let params = {}

        if (getState()[type].listView.query) {
          filter = getState()[type].listView.query
          params.filter = filter
        }

        if (getState()[type].listView.sumBy) {
          params.sum_by = getState()[type].listView.sumBy.split(',')
        } else if (options.defaultSumBy) {
          params.sum_by = options.defaultSumBy()
        }

        promise = chain[className].query(context, params)
      }

      return promise.then(
        (param) => dispatch(appendPage(param))
      ).catch((err) => {
        console.log(err)
        if (options.defaultKey && filter.indexOf('\'') < 0 && filter.indexOf('=') < 0) {
          dispatch(updateQuery(`${options.defaultKey}='${filter}'`))
        }
      })
    }
  }

  return {
    appendPage: appendPage,
    updateQuery: updateQuery,
    incrementPage: function() {
      return function(dispatch, getState) {
        const nextPage = getNextPageSlice(getState)

        if (nextPage.length < pageSize) {
          let fetchPromise = dispatch(fetchPage())

          if (nextPage.length != 0) {
            dispatch(incrementPage())
          } else if (getState()[type].listView.pageIndex != 0) {
            fetchPromise.then(() => {
              dispatch(incrementPage())
            })
          }

          return fetchPromise
        } else {
          return dispatch(incrementPage())
        }
      }
    },
    decrementPage: decrementPage
  }
}
