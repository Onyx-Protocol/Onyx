import React from 'react'
import styles from './Summary.scss'

const ACTION_NAMES = {
  issue: 'Issued',
  control: 'Received',
  spend: 'Spent',
  receive: 'Received',
  retire: 'Retired',
}

class Summary extends React.Component {
  normalizeActions(actions) {
    const normalized = {}

    actions.forEach(action => {
      let asset = normalized[action.asset_id]
      if (!asset) asset = normalized[action.asset_id] = {
        alias: action.asset_alias,
        issue: 0,
        retire: 0
      }

      if (['issue', 'retire'].includes(action.action)) {
        asset[action.action] += action.amount
      } else {
        let accountKey = action.account_id || 'external'
        let account = asset[accountKey]
        if (!account) account = asset[accountKey] = {
          alias: action.account_alias,
          spend: 0,
          receive: 0
        }

        if (action.action == 'spend') {
          account.spend += action.amount
        } else if (action.action == 'control' && action.purpose == 'change') {
          account.spend -= action.amount
        } else if (action.action == 'control') {
          account.receive += action.amount
        }
      }
    })

    return normalized
  }

  render() {
    const actions = this.props.transaction.inputs.concat(this.props.transaction.outputs)
    const summary = this.normalizeActions(actions)
    const items = []

    Object.keys(summary).forEach((asset_id) => {
      const asset = summary[asset_id]
      const nonAccountTypes = ['issue','retire']

      nonAccountTypes.forEach((type) => {
        if (asset[type] > 0) {
          items.push({
            action: ACTION_NAMES[type],
            rawAction: type,
            amount: asset[type],
            asset: asset.alias ? asset.alias : <code className={styles.asset_id}>{asset_id}</code>,
          })
        }
      })


      Object.keys(asset).forEach((account_id) => {
        if (nonAccountTypes.includes(account_id)) return
        const account = asset[account_id]
        if (!account) return

        const accountTypes = ['spend', 'receive']
        accountTypes.forEach((type) => {
          if (account[type] > 0) {
            items.push({
              action: ACTION_NAMES[type],
              rawAction: type,
              amount: account[type],
              asset: asset.alias ? asset.alias : <code className={styles.asset_id}>{asset_id}</code>,
              direction: type == 'spend' ? 'from' : 'to',
              account: account.alias ? account.alias : account_id,
            })
          }
        })
      })
    })

    return(<table className={styles.main}>
      <thead>
        <tr>
          <th>Action</th>
          <th>Amount</th>
          <th>Asset</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {items.map((item, index) =>
          <tr key={index} className={index % 2 == 0 ? '' : styles.odd}>
            <td className={styles.colAction}>{item.action}</td>
            <td className={styles.colAmount}>
              <code className={`${styles.amount} ${styles[item.rawAction]}`}>{item.amount}</code>
            </td>
            <td>{item.asset}</td>
            <td className={styles.colAccount}>
              <span className={styles.direction}>{item.direction}</span>
              {item.account}
            </td>
          </tr>
        )}
      </tbody>
    </table>)
  }
}

export default Summary
