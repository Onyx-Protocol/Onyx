import * as React from 'react'
import Section from '../../app/components/section'
import { Item as Contract } from '../types'
import { getIdList as getContractIds, getItem as getContract } from '../selectors'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'

function shortenHash(hash: string) {
  if (hash.length < 13) {
    return hash
  } else {
    return hash.slice(0, 15) + "..."
  }
}

const Contracts = connect(
  (state) => ({ contractIds: getContractIds(state), spentContractIds: [] })//getSpentContractIds(state) })
)(ContractsUnconnected)

export default Contracts

function ContractsUnconnected(props: {contractIds: string[], spentContractIds: string[] }) {
  return <div>
    <UnspentContracts contractIds={props.contractIds} />
    {props.spentContractIds.length ? <SpentContracts spentContractIds={props.spentContractIds} /> : <div />}
    </div>
}

function UnspentContracts(props: { contractIds: string[] }) {
  return <Section name="Unspent Contracts" >
    <table className="table">
      <thead><tr>
        <th>Contract Template</th>
        <th>Control Address</th>
        <th>Funding Transaction ID</th>
        <th></th>
      </tr></thead>
      <tbody>
      { props.contractIds.map((id) => <ContractRow key={id} contractId={id} />) }
      </tbody>
    </table>
  </Section>
}

function SpentContracts(props: { spentContractIds: string[] }) {
  return <Section name="Spent Contracts" >
  <table className="table">
    <thead><tr>
      <th>Contract Template</th>
      <th>ID</th>
      <th></th>
    </tr></thead>
    <tbody>
    { props.spentContractIds.map((id) => <SpentContractRow key={id} contractId={id} />) }
    </tbody>
  </table>
  </Section>
}

function ContractRowUnconnected(props: { contractId: string, contract: Contract }) {
  let contractId = props.contractId
  let contract = props.contract
  return <tr>
    <td>{ contract.template.name }</td>
    <td>{ shortenHash(contract.controlProgram) }</td>
    <td>{ shortenHash(contract.id) }</td>
    <td><SpendButton contractId={contractId} /></td>
  </tr>
}

const ContractRow = connect(
  (state, ownProps: { contractId: string }) => {  // mapStateToProps
    return { contract: getContract(state, ownProps.contractId) }
  }
)(ContractRowUnconnected)

function SpentContractRowUnconnected(props: { contractId: string, contract: Contract }) {
  let contractId = props.contractId
  let contract = props.contract
  return <tr>
    <td>{ contract.template.name }</td>
    <td>{ shortenHash(contract.id) }</td>
  </tr>
}

const SpentContractRow = connect(
  (state, ownProps: { contractId: string }) => { 
    return { contract: getContract(state, ownProps.contractId) }
  }
)(SpentContractRowUnconnected)

function SpendButton(props: {contractId: string} ) {
  return <Link to={"/spend/" + props.contractId} ><button className="btn btn-primary">Spend</button></Link>
}
