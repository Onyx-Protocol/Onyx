import * as React from 'react'
import DocumentTitle from 'react-document-title'
import Section from '../../app/components/section'
import { Contract } from '../types'
import { getIdList as getContractIds, getItem as getContract, getSpentIdList as getSpentContractIds } from '../selectors'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import { prefixRoute } from '../../util'
import { getItemMap as getAssetMap } from '../../assets/selectors'

function shortenHash(hash: string) {
  if (hash.length < 9) {
    return hash
  } else {
    return hash.slice(0, 8) + "..."
  }
}

const Contracts = connect(
  (state) => ({ contractIds: getContractIds(state), spentContractIds: getSpentContractIds(state)})
)(ContractsUnconnected)

export default Contracts

function ContractsUnconnected(props: {contractIds: string[], spentContractIds: string[] }) {
  return (
    <div>
      <UnspentContracts contractIds={props.contractIds} />
      <SpentContracts spentContractIds={props.spentContractIds} />
    </div>
  )
}

function UnspentContracts(props: { contractIds: string[] }) {
  let content = <div>No Unspent Contracts</div>
  if (props.contractIds.length > 0) {
    content = (
      <table className="table contracts-table">
        <thead>
          <tr>
            <th>Asset</th>
            <th>Amount</th>
            <th>Contract Template</th>
            <th>Lock Transaction</th>
            <th className="table-spaceholder"></th>
          </tr>
        </thead>
        <tbody>
          { props.contractIds.map((id) => <ContractRow key={id} contractId={id} />) }
        </tbody>
      </table>
    )
  }
  return (
    <DocumentTitle title="Locked Value">
      <Section name="Locked Value">
        {content}
      </Section>
    </ DocumentTitle>
  )
}

function ContractRowUnconnected(props: { asset, contractId: string, contract: Contract }) {
  const contract = props.contract
  return (
    <tr>
      <td>{ props.asset.alias }</td>
      <td>{ contract.amount }</td>
      <td>{ contract.template.name }</td>
      <td><a href={"/dashboard/transactions/" + contract.id} target="_blank">{ shortenHash(contract.id) }</a></td>
      <td><SpendButton contractId={contract.id} /></td>
    </tr>
  )
}

const ContractRow = connect(
  (state, ownProps: { contractId: string }) => {  // mapStateToProps
    const contract = getContract(state, ownProps.contractId)
    const assetMap = getAssetMap(state)
    return {
      asset: assetMap[contract.assetId],
      contract
    }
  }
)(ContractRowUnconnected)

function SpentContracts(props: { spentContractIds: string[] }) {
  let content = <div>No History</div>
  if (props.spentContractIds.length > 0) {
    content = (
      <table className="table contracts-table">
        <thead>
          <tr>
            <th>Asset</th>
            <th>Amount</th>
            <th>Contract Template</th>
            <th>Lock Transaction</th>
            <th>Unlock Transaction</th>
          </tr>
        </thead>
        <tbody>
          { props.spentContractIds.map((id) => <SpentContractRow key={id} contractId={id} />) }
        </tbody>
      </table>
    )
  }
  return (
    <Section name="History">
      {content}
    </Section>
  )
}

function SpentContractRowUnconnected(props: { asset, contractId: string, contract: Contract }) {
  const contract = props.contract
  return (
    <tr>
      <td>{ props.asset.alias }</td>
      <td>{ contract.amount }</td>
      <td>{ contract.template.name }</td>
<<<<<<< Updated upstream
      <td><code>{ shortenHash(contract.controlProgram) }</code></td>
      <td><a href={"/dashboard/transactions/" + contract.id} target="_blank">{ shortenHash(contract.id) }</a></td>
=======
      <td><a href={"/dashboard/transactions/" + contract.id} target="_blank">{ shortenHash(contract.id) }</a></td>
      <td><a href={"/dashboard/transactions/" + contract.spendTxid} target="_blank">{ shortenHash(contract.spendTxid) }</a></td>
>>>>>>> Stashed changes
      <td />
    </tr>
  )
}

const SpentContractRow = connect(
  (state, ownProps: { contractId: string }) => {
    const contract = getContract(state, ownProps.contractId)
    const assetMap = getAssetMap(state)
    return {
      asset: assetMap[contract.assetId],
      contract
    }
  }
)(SpentContractRowUnconnected)

function SpendButton(props: {contractId: string} ) {
  return <Link to={prefixRoute("/unlock/" + props.contractId)} ><button className="btn btn-primary pull-right">Unlock</button></Link>
}
