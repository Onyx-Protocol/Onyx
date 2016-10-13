import React from 'react'
import styles from './Summary.scss'

const INOUT_TYPES = {
  issue: 'Issued',
  control: 'Received',
  spend: 'Spent',
  receive: 'Received',
  retire: 'Retired',
}

class Summary extends React.Component {
  normalizeInouts(inouts) {
    const normalized = {}

    inouts.forEach(inout => {
      let asset = normalized[inout.asset_id]
      if (!asset) asset = normalized[inout.asset_id] = {
        alias: inout.asset_alias,
        issue: 0,
        retire: 0
      }

      if (['issue', 'retire'].includes(inout.type)) {
        asset[inout.type] += inout.amount
      } else {
        let accountKey = inout.account_id || 'external'
        let account = asset[accountKey]
        if (!account) account = asset[accountKey] = {
          alias: inout.account_alias,
          spend: 0,
          receive: 0
        }

        if (inout.type == 'spend') {
          account.spend += inout.amount
        } else if (inout.type == 'control' && inout.purpose == 'change') {
          account.spend -= inout.amount
        } else if (inout.type == 'control') {
          account.receive += inout.amount
        }
      }
    })

    return normalized
  }

  render() {
    const inouts = this.props.transaction.inputs.concat(this.props.transaction.outputs)
    const summary = this.normalizeInouts(inouts)
    const items = []

    Object.keys(summary).forEach((asset_id) => {
      const asset = summary[asset_id]
      const nonAccountTypes = ['issue','retire']

      nonAccountTypes.forEach((type) => {
        if (asset[type] > 0) {
          items.push({
            type: INOUT_TYPES[type],
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
              type: INOUT_TYPES[type],
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

    const ordering = ['issue', 'spend', 'receive', 'retire']
    items.sort((a,b) => {
      return ordering.indexOf(a.rawAction) - ordering.indexOf(b.rawAction)
    })

    return(<table className={styles.main}>
      <thead>
        <tr>
          <th>Type</th>
          <th>Amount</th>
          <th>Asset</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {items.map((item, index) =>
          <tr key={index} className={index % 2 == 0 ? '' : styles.odd}>
            <td className={styles.colAction}>{item.type}</td>
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
