import React from 'react'
import { Link } from 'react-router'
import styles from './Summary.scss'

const INOUT_TYPES = {
  issue: 'Issue',
  spend: 'Spend',
  control: 'Control',
  retire: 'Retire',
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
          control: 0
        }

        if (inout.type == 'spend') {
          account.spend += inout.amount
        } else if (inout.type == 'control' && inout.purpose == 'change') {
          account.spend -= inout.amount
        } else if (inout.type == 'control') {
          account.control += inout.amount
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
            asset: asset.alias ? asset.alias : <code className={styles.rawId}>{asset_id}</code>,
            assetId: asset_id,
          })
        }
      })


      Object.keys(asset).forEach((account_id) => {
        if (nonAccountTypes.includes(account_id)) return
        const account = asset[account_id]
        if (!account) return

        if (account_id == 'external') {
          account.alias= 'external'
          account_id = null
        }

        const accountTypes = ['spend', 'control']
        accountTypes.forEach((type) => {
          if (account[type] > 0) {
            items.push({
              type: INOUT_TYPES[type],
              rawAction: type,
              amount: account[type],
              asset: asset.alias ? asset.alias : <code className={styles.rawId}>{asset_id}</code>,
              assetId: asset_id,
              direction: type == 'spend' ? 'from' : 'to',
              account: account.alias ? account.alias : <code className={styles.rawId}>{account_id}</code>,
              accountId: account_id,
            })
          }
        })
      })
    })

    const ordering = ['issue', 'spend', 'control', 'retire']
    items.sort((a,b) => {
      return ordering.indexOf(a.rawAction) - ordering.indexOf(b.rawAction)
    })

    return(<table className={styles.main}>
      <tbody>
        {items.map((item, index) =>
          <tr key={index}>
            <td className={styles.colAction}>{item.type}</td>
            <td className={styles.colLabel}>amount</td>
            <td className={styles.colAmount}>
              <code className={styles.amount}>{item.amount}</code>
            </td>
            <td className={styles.colLabel}>asset</td>
            <td className={styles.colAccount}>
              <Link to={`/assets/${item.assetId}`}>
                {item.asset}
              </Link>
            </td>
            <td className={styles.colLabel}>{item.account && 'account'}</td>
            <td className={styles.colAccount}>
              {item.accountId && <Link to={`/accounts/${item.accountId}`}>
                {item.account}
              </Link>}
              {!item.accountId && item.account}
            </td>
          </tr>
        )}
      </tbody>
    </table>)
  }
}

export default Summary
