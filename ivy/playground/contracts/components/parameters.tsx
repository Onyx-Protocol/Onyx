// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// ivy imports
import { Item as Asset } from '../../assets/types'
import { Item as Account } from '../../accounts/types'
import { getItemMap as getAssetMap, getItemList as getAssets } from '../../assets/selectors'
import { getBalanceMap, getItemList as getAccounts, getBalanceSelector } from '../../accounts/selectors'
import { getClauseValueId, getState as getContractsState } from '../../contracts/selectors'
import { getShowLockInputErrors, getParameterIds, getInputMap, getContractValueId } from '../../templates/selectors'
import { getRequiredAssetAmount, getSpendContract } from '../../contracts/selectors'
import { seed } from '../../app/actions'

import RadioSelect from '../../app/components/radioSelect'
import { Input, InputContext, ParameterInput, NumberInput, BooleanInput, StringInput,
         ProvideStringInput, GenerateStringInput, HashInput,
         TimeInput, TimestampTimeInput,
         PublicKeyInput, GeneratePublicKeyInput, ProvidePublicKeyInput, GeneratePrivateKeyInput, GenerateHashInput,
         ProvideHashInput, InputType, ComplexInput, SignatureInput, GenerateSignatureInput,
         ProvideSignatureInput, ProvidePrivateKeyInput,
         ValueInput, AccountAliasInput, AssetAliasInput, AssetInput, AmountInput,
         ProgramInput, ChoosePublicKeyInput, KeyData, typeToString } from '../../inputs/types'

import { validateInput, computeDataForInput, getChild,
         getParameterIdentifier, getInputContext } from '../../inputs/data'

// internal imports
import { updateInput, updateClauseInput } from '../actions'
import { getShowUnlockInputErrors, getSpendInputMap, getClauseParameterIds } from '../selectors'

function getChildWidget(input: ComplexInput) {
  return getWidget(getChild(input))
}

function ParameterWidget(props: { input: ParameterInput, handleChange: (e)=>undefined }) {
  // handle the fact that clause arguments look like spend.sig rather than sig
  const parameterName = getParameterIdentifier(props.input)
  const valueType = typeToString(props.input.valueType)
  return (
    <div key={props.input.name}>
      <label>{parameterName}: <span className='type-label'>{valueType}</span></label>
      {getChildWidget(props.input)}
    </div>
  )
}

function GenerateStringWidget(props: { id: string,
                                       input: GenerateStringInput,
                                       errorClass: string,
                                       handleChange: (e)=>undefined}) {
  return (
    <div>
      <div className={"input-group " + props.errorClass}>
        <div className="input-group-addon">Length</div>
        <input type="text" className="form-control" style={{width: 200}} key={props.input.name} value={props.input.value} onChange={props.handleChange} />
      </div>
      <ComputedValue computeFor={props.id} />
    </div>
  )
}

function NumberWidget(props: { input: NumberInput | AmountInput,
                               handleChange: (e)=>undefined }) {
  return <input type="text" className="form-control" style={{width: 200}} key={props.input.name} value={props.input.value} onChange={props.handleChange} />
}

function TimestampTimeWidget(props: { input: TimeInput,
                                      errorClass: string,
                                      handleChange: (e)=>undefined }) {
  return (
    <div className={"form-group " + props.errorClass}>
      <input type="datetime-local" placeholder="yyyy-mm-ddThh:mm:ss" key={props.input.name} className="form-control" value={props.input.value} onChange={props.handleChange} />
    </div>
  )
}

function AmountWidget(props: { input: AmountInput,
                               errorClass: string,
                               handleChange: (e)=>undefined }) {
  return (
    <div className={"form-group " + props.errorClass}>
      <div className="input-group">
        <div className="input-group-addon">Amount</div>
        <NumberWidget input={props.input} handleChange={props.handleChange} />
      </div>
    </div>
  )
}

function BooleanWidget(props: { input: BooleanInput, handleChange: (e)=>undefined }) {
  return <input type="checkbox" key={props.input.name} checked={(props.input.value === "true")} onChange={props.handleChange} />
}

function StringWidget(props: { input: StringInput, handleChange: (e)=>undefined }) {
  const options = [{label: "Generate String", value: "generateStringInput"},
                   {label: "Provide String (Hex)", value: "provideStringInput"}]
  const handleChange = (s: string) => undefined
  return (
    <div>
      <RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
      {getChildWidget(props.input)}
    </div>
  )
}

function TextWidget(props: { input: ProvideStringInput | ProvideHashInput |
                                    ProvidePublicKeyInput | ProvideSignatureInput |
                                    ProvidePrivateKeyInput,
                             errorClass: string,
                             handleChange: (e)=>undefined }) {
  return (
    <div className={"form-group " + props.errorClass}>
      <input type="text" key={props.input.name} className="form-control string-input" value={props.input.value} onChange={props.handleChange} />
    </div>
  )
}

function HashWidget(props: { input: HashInput, handleChange: (e)=>undefined }) {
  const options = [{label: "Generate Hash", value: "generateHashInput"},
                   {label: "Provide Hash", value: "provideHashInput"}]
  const handleChange = (s: string) => undefined
  return (
    <div>
      <RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
      {getChildWidget(props.input)}
    </div>
  )
}

function GenerateHashWidget(props: { id: string,
                                     input: GenerateHashInput,
                                     handleChange: (e)=>undefined}) {
  return (
    <div>
      <ComputedValue computeFor={props.id} />
      <div className="nested">
        <div className="description">{props.input.hashFunction} of:</div>
        <label className="type-label">{typeToString(props.input.inputType)}</label>
        {getChildWidget(props.input)}
      </div>
    </div>
  )
}

function PublicKeyWidget(props: { input: PublicKeyInput,
                                  handleChange: (e)=>undefined }) {
  const options = [{label: "Generate Public Key", value: "accountInput"},
                   {label: "Provide Public Key", value: "provideStringInput"}]
  const handleChange = (s: string) => undefined
  return (
    <div>
      <RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
      {getChildWidget(props.input)}
    </div>
  )
}

function GeneratePublicKeyWidget(props: { id: string,
                                          input: GeneratePublicKeyInput,
                                          handleChange: (e)=>undefined}) {
  const options = [{label: "Generate Private Key", value: "generatePrivateKeyInput"},
                   {label: "Provide Private Key", value: "providePrivateKeyInput"}]
  return (
    <div>
      <ComputedValue computeFor={props.id} />
      <div className="nested">
        <div className="description">derived from:</div>
        <label className="type-label">PrivateKey</label>
        <RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
        {getChildWidget(props.input)}
      </div>
    </div>
  )
}

function GenerateSignatureWidget(props: { input: GenerateSignatureInput,
                                          handleChange: (e)=>undefined,
                                          computedValue: string }) {
  return (
    <div>{props.computedValue ? <pre>{props.computedValue}</pre> : <span />}
      <div className="nested">
        <div className="description">signed using:</div>
        <label className="type-label">PrivateKey</label>
        {getChildWidget(props.input)}
      </div>
    </div>
  )
}

function SignatureWidget(props: { input: SignatureInput,
                                  handleChange: (e)=>undefined,
                                  computedValue: string }) {
  return <div>{getChildWidget(props.input)}</div>
}

function GeneratePrivateKeyWidget(props: { input: GeneratePrivateKeyInput, handleChange: (e)=>undefined }) {
  return <div><pre>{props.input.value}</pre></div>
}

function TimeWidget(props: { input: TimeInput, handleChange: (e)=>undefined }) {
  return <div>{getChildWidget(props.input)}</div>
}

const EmptyCoreAlert = connect(
  (state) => ({ balanceMap: getBalanceMap(state) }),
  (dispatch) => ({
    handleClick(e) {
      e.preventDefault()
      dispatch(seed())
    }
  })
)(EmptyCoreAlertUnconnected)

function EmptyCoreAlertUnconnected({ handleClick, balanceMap }) {
  if (Object.keys(balanceMap).length === 0) {
    return (
      <div style={{width: '300px'}}className="alert alert-warning" role="alert">
        <a style={{color: 'inherit'}} className="alert-link" href='#' onClick={handleClick}>Seed Chain Core</a> w/ Accounts & Assets
      </div>
    )
  }
  return <small/>
}

const InsufficientFundsAlert = connect(
  (state, ownProps: { namePrefix: string }) => ({ balance: getBalanceSelector(ownProps.namePrefix)(state), inputMap: getInputMap(state), contracts: getContractsState(state) })
)(InsufficientFundsAlertUnconnected)

function InsufficientFundsAlertUnconnected({ namePrefix, balance, inputMap, contracts }) {
  let amountInput
  if (namePrefix.startsWith("contract")) {
    amountInput = inputMap[namePrefix + ".amountInput"]
  } else if (namePrefix.startsWith("clause")) {
    // THIS IS A HACK
    const spendInputMap = contracts.contractMap[contracts.spendContractId].spendInputMap
    amountInput = spendInputMap[namePrefix + ".valueInput.amountInput"]
  }
  let jsx = <small/>
  if (balance !== undefined && amountInput && amountInput.value) {
    if (balance < amountInput.value) {
      jsx = (
        <div style={{width: '300px'}}className="alert alert-danger" role="alert">
          Insufficient Funds
        </div>
      )
    }
  }
  return jsx
}

const BalanceWidget = connect(
  (state, ownProps: { namePrefix: string }) => ({ balance: getBalanceSelector(ownProps.namePrefix)(state) })
)(BalanceWidgetUnconnected)

function BalanceWidgetUnconnected({ namePrefix, balance }) {
  let jsx = <small/>
  if (balance !== undefined) {
    jsx = <small className="value-balance">{balance} available</small>
  }
  return jsx
}

function ValueWidget(props: { input: ValueInput, handleChange: (e)=>undefined }) {
  return (
    <div>
      <EmptyCoreAlert />
      <InsufficientFundsAlert namePrefix={props.input.name} />
      {getWidget(props.input.name + ".accountInput")}
      {getWidget(props.input.name + ".assetInput")}
      {getWidget(props.input.name + ".amountInput")}
      <BalanceWidget namePrefix={props.input.name} />
    </div>
  )
}

function ProgramWidget(props: { input: ProgramInput, handleChange: (e)=>undefined }) {
  return <div>{getChildWidget(props.input)}</div>
}

const AccountAliasWidget = connect(
  (state) => ({ accounts: getAccounts(state) })
)(AccountAliasWidgetUnconnected)

function AccountAliasWidgetUnconnected(props: { input: AccountAliasInput,
                                                errorClass: string,
                                                handleChange: (e)=>undefined,
                                                accounts: Account[]}) {
  const options = props.accounts.map(account => <option key={account.id} value={account.id}>{account.alias}</option>)
  if (options.length === 0) {
    options.push(<option key="" value="">No Accounts Available</option>)
  } else {
    options.unshift(<option key="" value="">Select Account</option>)
  }
  return (
    <div className={"form-group " + props.errorClass}>
      <div className="input-group">
        <div className="input-group-addon">Account</div>
        <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
          {options}
        </select>
      </div>
    </div>
  )
}

const AssetAliasWidget = connect(
  (state) => ({ assets: getAssets(state) })
)(AssetAliasWidgetUnconnected)

function AssetAliasWidgetUnconnected(props: { input: AssetAliasInput,
                                              errorClass: string,
                                              handleChange: (e)=>undefined,
                                              assets: Asset[]}) {
  const options = props.assets.map(asset => <option key={asset.id} value={asset.id}>{asset.alias}</option>)
  if (options.length === 0) {
    options.push(<option key="" value="">No Assets Available</option>)
  } else {
    options.unshift(<option key="" value="">Select Asset</option>)
  }
  return (
    <div className={"form-group " + props.errorClass}>
      <div className="input-group">
        <div className="input-group-addon">Asset</div>
        <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
          {options}
        </select>
      </div>
    </div>
  )
}

function ChoosePublicKeyWidget(props: { input: ChoosePublicKeyInput,
                                        errorClass: string,
                                        handleChange: (e)=>undefined }) {
  if (props.input.keyMap === undefined) {
    throw 'keyMap is undefined'
  }

  const options : any[] = []
  const map: {[s: string]: KeyData} = props.input.keyMap
  for (const key in map) {
    options.push(<option key={key} value={key}>{key}</option>)
  }
  options.unshift(<option key="" value="">Select Public Key</option>)

  return (
    <div className={"form-group " + props.errorClass}>
      <div className="input-group">
        <div className="input-group-addon">Public Key</div>
        <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
          {options}
        </select>
      </div>
    </div>
  )
}

function getWidgetType(type: InputType): ((props: { input: Input, handleChange: (e)=>undefined }) => JSX.Element) {
  switch (type) {
    case "numberInput": return NumberWidget
    case "booleanInput": return BooleanWidget
    case "stringInput": return StringWidget
    case "generateStringInput": return GenerateStringWidget
    case "provideStringInput": return TextWidget
    case "publicKeyInput": return PublicKeyWidget
    case "signatureInput": return SignatureWidget
    case "generateSignatureInput": return GenerateSignatureWidget
    case "generatePublicKeyInput": return GeneratePublicKeyWidget
    case "generatePrivateKeyInput": return GeneratePrivateKeyWidget
    case "providePublicKeyInput": return TextWidget
    case "providePrivateKeyInput": return TextWidget
    case "provideSignatureInput": return TextWidget
    case "hashInput": return HashWidget
    case "provideHashInput": return TextWidget
    case "generateHashInput": return GenerateHashWidget
    case "timeInput": return TimeWidget
    case "timestampTimeInput": return TimestampTimeWidget
    case "programInput": return ProgramWidget
    case "valueInput": return ValueWidget
    case "accountInput": return AccountAliasWidget
    case "assetInput": return AssetAliasWidget
    case "amountInput": return AmountWidget
    case "assetInput": return AssetAliasWidget
    case "amountInput": return AmountWidget
    case "amountInput": return AmountWidget
    case "programInput": return ProgramWidget
    case "choosePublicKeyInput": return ChoosePublicKeyWidget
    default: return ParameterWidget
  }
}

function mapToInputProps(showError: boolean, inputsById: {[s: string]: Input}, id: string) {
  const input = inputsById[id]
  if (input === undefined) {
    throw "bad input ID: " + id
  }

  let errorClass = ''
  const hasInputError = !validateInput(input)
  if (showError && hasInputError) {
    errorClass = 'has-error'
  }
  if (input.type === "generateSignatureInput") {
    return {
      input,
      errorClass,
      computedValue: "",
    }
  }

  return {
    input,
    errorClass
  }
}

function mapStateToContractInputProps(state, ownProps: { id: string }) {
  const inputMap = getInputMap(state)
  if (inputMap === undefined) {
    throw "inputMap should not be undefined when contract inputs are being rendered"
  }
  const showError = getShowLockInputErrors(state)
  return mapToInputProps(showError, inputMap, ownProps.id)
}

function mapDispatchToContractInputProps(dispatch, ownProps: { id: string }) {
  return {
    handleChange: (e) => {
      dispatch(updateInput(ownProps.id, e.target.value.toString()))
    }
  }
}

function mapStateToSpendInputProps(state, ownProps: { id: string }) {
  const inputsById = getSpendInputMap(state)
  const showError = getShowUnlockInputErrors(state)
  return mapToInputProps(showError, inputsById, ownProps.id)
}

function mapDispatchToSpendInputProps(dispatch, ownProps: { id: string} ) {
  return {
    handleChange: (e) => {
      dispatch(updateClauseInput(ownProps.id, e.target.value.toString()))
    }
  }
}

function mapToComputedProps(state, ownProps: { computeFor: string} ) {
  let inputsById = getInputMap(state)
  if (inputsById === undefined) throw "inputMap should not be undefined when contract inputs are being rendered"
  let input = inputsById[ownProps.computeFor]
  if (input === undefined) throw "bad input ID: " + ownProps.computeFor
  if (input.type === "generateHashInput" ||
      input.type === "generateStringInput") {
    try {
      let computedValue = computeDataForInput(ownProps.computeFor, inputsById)
      return {
        value: computedValue
      }
    } catch(e) {
      return {}
    }
  }
}

const ComputedValue = connect(
  mapToComputedProps,
)(ComputedValueUnconnected)

function ComputedValueUnconnected(props: { value: string }) {
  return props.value? <pre>{props.value}</pre> : <span />
}

export function getWidget(id: string): JSX.Element {
  let inputContext = id.split(".").shift() as InputContext
  let type = id.split(".").pop() as InputType
  let widgetTypeConnected
  if (inputContext === "contractParameters" || inputContext === "contractValue") {
    widgetTypeConnected = connect(
      mapStateToContractInputProps,
      mapDispatchToContractInputProps
    )(getWidgetType(type))
  } else {
    widgetTypeConnected = connect(
      mapStateToSpendInputProps,
      mapDispatchToSpendInputProps
    )(getWidgetType(type))
  }
  return (
    <div className="widget-wrapper" key={"container(" + id + ")"}>
      {React.createElement(widgetTypeConnected, { key: "connect(" + id + ")", id: id })}
    </div>
  )
}

function mapStateToContractParametersProps(state) {
  return {
    parameterIds: getParameterIds(state)
  }
}

export const ContractParameters = connect(
  mapStateToContractParametersProps
)(ContractParametersUnconnected)

function ContractParametersUnconnected(props: { parameterIds: string[] }) {
  if (props.parameterIds.length === 0) return <div />
  const parameterInputs = props.parameterIds.map((id) => {
    return <div key={id} className="argument">{getWidget(id)}</div>
  })
  return (
    <section style={{wordBreak: 'break-all'}}>
      <form className="form">
        {parameterInputs}
      </form>
    </section>
  )
}

export const ClauseParameters = connect(
  (state) => ({ parameterIds: getClauseParameterIds(state) })
)(ClauseParametersUnconnected)

function ClauseParametersUnconnected(props: { parameterIds: string[] }) {
  if (props.parameterIds.length === 0) return <div />
  let parameterInputs = props.parameterIds.map((id) => {
    return <div key={id} className="argument">{getWidget(id)}</div>
  })
  return <section style={{wordBreak: 'break-all'}}>
    <h4>Clause Arguments</h4>
    <form className="form">
    {parameterInputs}
  </form></section>
}

function mapStateToClauseValueProps(state) {
  return {
    valueId: getClauseValueId(state),
    assetMap: getAssetMap(state),
    assetAmount: getRequiredAssetAmount(state),
    balanceMap: getBalanceMap(state),
    spendInputMap: getSpendInputMap(state)
  }
}

export const ClauseValue = connect(
  mapStateToClauseValueProps
)(ClauseValueUnconnected)

function ClauseValueUnconnected(props: { spendInputMap, balanceMap, assetAmount, assetMap, valueId: string }) {
  if (props.valueId === undefined || props.assetAmount === undefined) {
    return <div />
  } else {
    const parameterName = props.valueId.split('.').pop()
    const valueType = "Value"
    props.spendInputMap[props.valueId + ".valueInput.assetInput"].value = props.assetAmount.assetId
    props.spendInputMap[props.valueId + ".valueInput.amountInput"].value = props.assetAmount.amount
    return (
      <section style={{wordBreak: 'break-all'}}>
        <h4>Required Value</h4>
        <form className="form">
          <label>{parameterName}: <span className='type-label'>{valueType}</span></label>
          <InsufficientFundsAlert namePrefix={props.valueId} />
          {getWidget(props.valueId + ".valueInput.accountInput")}
          <div className="form-group">
            <div className="input-group">
              <div className="input-group-addon">Asset</div>
              <input type="text" className="form-control" value={props.assetMap[props.assetAmount.assetId].alias} disabled />
            </div>
          </div>
          <div className="form-group">
            <div className="input-group">
              <div className="input-group-addon">Amount</div>
              <input type="text" className="form-control" value={props.assetAmount.amount} disabled />
            </div>
          </div>
          <BalanceWidget namePrefix={props.valueId} />
        </form>
      </section>
    )
  }
}

function mapStateToContractValueProps(state) {
  return {
    valueId: getContractValueId(state)
  }
}

export const ContractValue = connect(
  mapStateToContractValueProps
)(ContractValueUnconnected)

function ContractValueUnconnected(props: { valueId: string }) {
  if (props.valueId === undefined) {
    return <div></div>
  }
  return (
    <section style={{wordBreak: 'break-all'}}>
      <form className="form">
        <div className="argument">{getWidget(props.valueId)}</div>
      </form>
    </section>
  )
}

