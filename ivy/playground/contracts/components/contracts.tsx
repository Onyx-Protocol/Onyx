import * as React from 'react'
import DocumentTitle from 'react-document-title'
import Section from '../../app/components/section'
import { Contract } from '../types'
import { getIdList as getContractIds, getItem as getContract, getSpentIdList as getSpentContractIds } from '../selectors'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import { prefixRoute } from '../../util'

function shortenHash(hash: string) {
  if (hash.length < 43) {
    return hash
  } else {
    return hash.slice(0, 42) + "..."
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
      {props.spentContractIds.length ? <SpentContracts spentContractIds={props.spentContractIds} /> : <div />}
    </div>
  )
}

function UnspentContracts(props: { contractIds: string[] }) {
  return (
    <DocumentTitle title="Contracts">
      <Section name="Unspent Contracts" >
        <table className="table contracts-table">
          <thead>
            <tr>
              <th>Template</th>
              <th>Program</th>
              <th>Creation Transaction</th>
              <th className="table-spaceholder"></th>
            </tr>
          </thead>
          <tbody>
            { props.contractIds.map((id) => <ContractRow key={id} contractId={id} />) }
          </tbody>
        </table>
      </Section>
    </ DocumentTitle>
  )
}

function SpentContracts(props: { spentContractIds: string[] }) {
  return (
    <Section name="Spent Contracts" >
      <table className="table contracts-table">
        <thead>
          <tr>
            <th width="30%">Template</th>
            <th width="30%">Program</th>
            <th width="30%">Spending Transaction</th>
            <th className="table-spaceholder"></th>
          </tr>
        </thead>
        <tbody>
          { props.spentContractIds.map((id) => <SpentContractRow key={id} contractId={id} />) }
        </tbody>
      </table>
    </Section>
  )
}

function ContractRowUnconnected(props: { contractId: string, contract: Contract }) {
  const contract = props.contract
  return (
    <tr>
      <td>{ contract.template.name }</td>
      <td><pre className="codeblock program">{ shortenHash(contract.controlProgram) }</pre></td>
      <td><a href={"/dashboard/transactions/" + contract.id} target="_blank">{ shortenHash(contract.id) }</a></td>
      <td><SpendButton contractId={contract.id} /></td>
    </tr>
  )
}

const ContractRow = connect(
  (state, ownProps: { contractId: string }) => {  // mapStateToProps
    return { contract: getContract(state, ownProps.contractId) }
  }
)(ContractRowUnconnected)

function SpentContractRowUnconnected(props: { contractId: string, contract: Contract }) {
  const contract = props.contract
  return (
    <tr>
      <td>{ contract.template.name }</td>
      <td><pre className="codeblock program">{ shortenHash(contract.controlProgram) }</pre></td>
      <td><a href={"/dashboard/transactions/" + contract.id} target="_blank">{ shortenHash(contract.id) }</a></td>
      <td />
    </tr>
  )
}

const SpentContractRow = connect(
  (state, ownProps: { contractId: string }) => {
    return { contract: getContract(state, ownProps.contractId) }
  }
)(SpentContractRowUnconnected)

function SpendButton(props: {contractId: string} ) {
  return <Link to={prefixRoute("/spend/" + props.contractId)} ><button className="btn btn-primary pull-right">Spend</button></Link>
}
