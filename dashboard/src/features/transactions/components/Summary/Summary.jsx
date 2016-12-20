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
      let assetId = inout.asset_id
      if (inout.readable != 'yes') {
        assetId = inout.asset_id_commitment
      }

      let asset = normalized[assetId]
      if (!asset) asset = normalized[assetId] = {
        alias: inout.asset_alias,
        assetCommitment: inout.asset_id_commitment,
        issue: {amount :0},
        retire: {amount: 0},
        accounts: {}
      }

      if (['issue', 'retire'].includes(inout.type)) {
        asset[inout.type].amount += (inout.amount || 0)
        if (inout.readable != 'yes') {
          asset[inout.type].hidden = true
          asset[inout.type].amountCommitment = inout.amount_commitment
        }
      } else {
        let accountKey = inout.account_id || 'external'
        let account = asset.accounts[accountKey]
        if (!account) account = asset.accounts[accountKey] = {
          id: inout.account_id,
          alias: inout.account_alias || 'external',
          spend: {amount: 0},
          control: {amount: 0}
        }

        if (inout.type == 'spend') {
          account.spend.amount += (inout.amount || 0)
        } else if (inout.type == 'control' && inout.purpose == 'change') {
          account.spend.amount -= (inout.amount || 0)
        } else if (inout.type == 'control') {
          account.control.amount += (inout.amount || 0)
        }
        if (inout.readable != 'yes') {
          account[inout.type].hidden = true
          account[inout.type].amountCommitment = inout.amount_commitment
        }
      }
    })

    return normalized
  }

  render() {
    const inouts = this.props.transaction.inputs.concat(this.props.transaction.outputs)
    const summary = this.normalizeInouts(inouts)
    const items = []

    Object.keys(summary).forEach(assetId => {
      const asset = summary[assetId]

      const actions = ['issue','retire']
      actions.forEach((type) => {
        if (asset[type].hidden) {
          console.log(asset);
          console.log(asset[type]);
          items.push({
            type: INOUT_TYPES[type],
            assetCommitment: asset.assetCommitment,
            amountCommitment: asset[type].amountCommitment,
            hidden: true
          })
        } else if (asset[type].amount > 0) {
          items.push({
            type: INOUT_TYPES[type],
            rawAction: type,
            amount: asset[type].amount,
            asset: asset.alias ? asset.alias : <code className={styles.rawId}>{assetId}</code>,
            assetId: assetId,
          })
        }
      })

      Object.values(asset.accounts).forEach(account => {
        const accountTypes = ['spend', 'control']
        accountTypes.forEach((type) => {
          if (!account[type]) return

          if (account[type].hidden) {
            items.push({
              type: INOUT_TYPES[type],
              assetCommitment: asset.assetCommitment,
              amountCommitment: account[type].amountCommitment,
              hidden: true
            })
          } else if (account[type].amount > 0) {
            items.push({
              type: INOUT_TYPES[type],
              rawAction: type,
              amount: account[type].amount,
              asset: asset.alias ? asset.alias : <code className={styles.rawId}>{assetId}</code>,
              assetId: assetId,
              direction: type == 'spend' ? 'from' : 'to',
              account: account.alias ? account.alias : <code className={styles.rawId}>{account_id}</code>,
              accountId: account.id,
            })
          }
        })
      })
    })

    const ordering = ['issue', 'spend', 'control', 'retire']
    items.sort((a,b) => {
      return ordering.indexOf(a.rawAction) - ordering.indexOf(b.rawAction)
    })

    if (items.length == 0) {
      return null
    }

    const confidentialIcon = (value) => (<span className={styles.confidential}>
      <a href='#' onClick={(e) => e.preventDefault()}>
        <span className={`${styles.icon} glyphicon glyphicon-lock`} />
        confidential

        <code className={styles.confidentialValue}>
          {value}
        </code>
      </a>
    </span>)

    return(<table className={styles.main}>
      <tbody>
        {items.map((item, index) => {
          console.log(item);
          return (
          <tr key={index}>
            <td className={styles.colAction}>{item.type}</td>
            <td className={styles.colLabel}>amount</td>
            <td className={styles.colAmount}>
              {item.hidden
                ? confidentialIcon(item.amountCommitment)
                : <code className={styles.amount}>{item.amount}</code>}
            </td>
            <td className={styles.colLabel}>asset</td>
            <td className={styles.colAccount}>
              {item.hidden
                ? confidentialIcon(item.assetCommitment)
                : <Link to={`/assets/${item.assetId}`}>
                  {item.asset}
                </Link>}
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
        )}
      </tbody>
    </table>)
  }
}

export default Summary
