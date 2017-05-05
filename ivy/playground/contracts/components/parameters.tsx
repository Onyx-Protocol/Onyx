import * as React from 'react'
import { connect } from 'react-redux'
import { isHash, isTypeVariable, isList, typeToString,
         toContractParameter, Type, Template, ContractParameter,
         ContractParameterType } from 'ivy-compiler'
import { Input, InputContext, ParameterInput, NumberInput, BooleanInput, StringInput,
         ProvideStringInput, GenerateStringInput, HashInput, DurationInput, SecondsDurationInput,
         BlocksDurationInput, TimeInput, BlockheightTimeInput, TimestampTimeInput,
         PublicKeyInput, GeneratePublicKeyInput, ProvidePublicKeyInput, GeneratePrivateKeyInput, GenerateHashInput,
         ProvideHashInput, InputType, ComplexInput, SignatureInput, GenerateSignatureInput,
         ProvideSignatureInput, ProvidePrivateKeyInput,
         MaxtimeInput, MintimeInput, ValueInput, AccountAliasInput, AssetAliasInput, AssetAmountInput, AmountInput,
         AddressInput, ChoosePublicKeyInput, KeyData } from '../../inputs/types'
import { getParameterIdentifier, getInputContext } from '../../inputs/data'
import { getInputMap, getSpendInputMap, getClauseParameterIds, getShowErrors } from '../selectors'
import { updateClauseInput } from '../actions'
/* import { getAssetAliasesById, getAssetIds } from '../assets' */
/* import { getAccountAliasesById, getAccountIds } from '../accounts' */
import accounts from '../../accounts'
import { Item as Account } from '../../accounts/types'

import assets from '../../assets'
import { Item as Asset } from '../../assets/types'

import { updateInput } from '../actions'
import { getParameterIds } from '../selectors'
import { validateInput, computeDataForInput, getChild } from '../../inputs/data'
// import { getSpendParameterIds, getSpending, getSignatureData } from '../spend'
// import { updateClauseInput } from '../contracts'
import app from '../../app'


function getChildWidget(input: ComplexInput) {
  return getWidget(getChild(input))
}

function ParameterWidget(props: { input: ParameterInput, handleChange: (e)=>undefined }) {
  // handle the fact that clause arguments look like spend.sig rather than sig
  let parameterName = getParameterIdentifier(props.input)
  let valueType = typeToString(props.input.valueType)
  return <div key={props.input.name}>
    <label>{parameterName}: <span className='type-label'>{valueType}</span></label>
    {getChildWidget(props.input)}
  </div>
}

function GenerateStringWidget(props: { input: GenerateStringInput, handleChange: (e)=>undefined, computedValue: string }) {
  return <div>{ props.computedValue ? <pre>{props.computedValue}</pre> : <span />}<div className="input-group">
    <div className="input-group-addon">Length</div>
    <input type="number" className="form-control" key={props.input.name} value={props.input.value} onChange={props.handleChange} />
  </div></div>
}

function NumberWidget(props: { input: NumberInput,
                               handleChange: (e)=>undefined }) {
  return <input type="number" className="form-control" style={{width: 200}} key={props.input.name} value={props.input.value} onChange={props.handleChange} />
}

function PositiveNumberWidget(props: { input: BlockheightTimeInput |
                                      TimestampTimeInput | BlocksDurationInput |
                                      SecondsDurationInput | AmountInput
                               handleChange: (e)=>undefined }) {
  return <input id={props.input.name} type="number" min={0} className="form-control" style={{width: 200}} key={props.input.name} value={props.input.value} onChange={props.handleChange} />
}

function AmountWidget(props: { input: AmountInput
                               handleChange: (e)=>undefined }) {
  return <div className="form-group form-inline">
    <div className="input-group">
    <div className="input-group-addon">Amount</div>
    <PositiveNumberWidget input={props.input} handleChange={props.handleChange} />
    </div>
  </div>
}

function BooleanWidget(props: { input: BooleanInput, handleChange: (e)=>undefined }) {
  return <input type="checkbox" key={props.input.name} checked={(props.input.value === "true")} onChange={props.handleChange} />
}

function StringWidget(props: { input: StringInput, handleChange: (e)=>undefined }) {
  let options = [{label: "Generate String", value: "generateStringInput"},
                 {label: "Provide String", value: "provideStringInput"}]
  let handleChange = (s: string) => undefined
  return <div>
    <app.components.RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
    {getChildWidget(props.input)}
  </div>
}

function TextWidget(props: { input: ProvideStringInput | ProvideHashInput |
                                    ProvidePublicKeyInput | ProvideSignatureInput |
                                    ProvidePrivateKeyInput
                             handleChange: (e)=>undefined }) {
  return <div className = "form-group"><input type="text" key={props.input.name} className="form-control" value={props.input.value} onChange={props.handleChange} /></div>
}

function HashWidget(props: { input: HashInput, handleChange: (e)=>undefined }) {
  let options = [{label: "Generate Hash", value: "generateHashInput"},
                 {label: "Provide Hash", value: "provideHashInput"}]
  let handleChange = (s: string) => undefined
  return <div>
    <app.components.RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
    {getChildWidget(props.input)}
  </div>
}

function GenerateHashWidget(props: { input: GenerateHashInput, handleChange: (e)=>undefined,
                                     computedValue: string }) {
  return <div>
      {props.computedValue ? <pre>{props.computedValue}</pre> : <span />}
      <div className="nested">
      <div className="description">{props.input.hashType.hashFunction} of:</div>
      <label className="type-label">{typeToString(props.input.hashType.inputType)}</label>
      {getChildWidget(props.input)}
  </div></div>
}

function PublicKeyWidget(props: { input: PublicKeyInput, handleChange: (e)=>undefined }) {
  return <div>
    {getChildWidget(props.input)}
  </div>
}

function GeneratePublicKeyWidget(props: { input: GeneratePublicKeyInput, handleChange: (e)=>undefined,
                                          computedValue: string }) {
  let options = [{label: "Generate Private Key", value: "generatePrivateKeyInput"},
                 {label: "Provide Private Key", value: "providePrivateKeyInput"}]
  return <div>{props.computedValue ? <pre>{props.computedValue}</pre> : <span />}
      <div className="nested">
      <div className="description">derived from:</div>
      <label className="type-label">PrivateKey</label>
    <app.components.RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
    {getChildWidget(props.input)}
  </div></div>
}

function GenerateSignatureWidget(props: { input: GenerateSignatureInput, handleChange: (e)=>undefined,
                                          computedValue: string }) {
  return <div>{props.computedValue ? <pre>{props.computedValue}</pre> : <span />}
      <div className="nested">
      <div className="description">signed using:</div>
      <label className="type-label">PrivateKey</label>
    {getChildWidget(props.input)}
  </div></div>
}

function SignatureWidget(props: { input: SignatureInput, handleChange: (e)=>undefined,
                                          computedValue: string }) {
  return <div>
    {getChildWidget(props.input)}
  </div>
}

function GeneratePrivateKeyWidget(props: { input: GeneratePrivateKeyInput, handleChange: (e)=>undefined }) {
  return <div><pre>{props.input.value}</pre></div>
}

function DurationWidget(props: { input: DurationInput, handleChange: (e)=>undefined }) {
  let options = [{label: "Blocks", value: "blocksDurationInput"},
                 {label: "512 Second Units", value: "secondsDurationInput"}]
  return <div>
    <app.components.RadioSelect options={options} selected={props.input.value} name={props.input.name} handleChange={props.handleChange} />
    {getChildWidget(props.input)}
  </div>
}

function TimeWidget(props: { input: TimeInput, handleChange: (e)=>undefined }) {
  return <div>
    {getChildWidget(props.input)}
  </div>
}

function MintimeWidget(props: { input: MintimeInput, handleChange: (e)=>undefined }) {
  return getChildWidget(props.input)
}

function MaxtimeWidget(props: { input: MaxtimeInput, handleChange: (e)=>undefined }) {
  return getChildWidget(props.input)
}

function ValueWidget(props: { input: ValueInput, handleChange: (e)=>undefined }) {
  return <div>
    {getWidget(props.input.name + ".accountAliasInput")}
    {getWidget(props.input.name + ".assetAmountInput")}
  </div>
}

function AssetAmountWidget(props: { input: AssetAmountInput, handleChange: (e)=>undefined }) {
  return <div>
    {getWidget(props.input.name + ".assetAliasInput")}
    {getWidget(props.input.name + ".amountInput")}
  </div>
}

function AddressWidget(props: { input: AddressInput, handleChange: (e)=>undefined }) {
  return <div>
    {getChildWidget(props.input)}
  </div>
}

function AccountAliasWidgetUnconnected(props: { input: AccountAliasInput,
                                     handleChange: (e)=>undefined,
                                     accounts: Account[]}) {
  let options = props.accounts.map(account => <option key={account.id} value={account.id}>{account.alias}</option>)
  options.unshift(<option key="" value="">Select Account</option>)
  return <div className="form-group form-inline">
    <div className="input-group">
    <div className="input-group-addon">Account</div>
    <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
      {options}
    </select>
    </div>
    </div>
}

let AccountAliasWidget = connect(
  (state) => ({ accounts: accounts.selectors.getItemList(state) })
)(AccountAliasWidgetUnconnected)

function AssetAliasWidgetUnconnected(props: { input: AssetAliasInput,
                                              handleChange: (e)=>undefined,
                                              assets: Asset[]}) {
  let options = props.assets.map(asset => <option key={asset.id} value={asset.id}>{asset.alias}</option>)
  options.unshift(<option key="" value="">Select Asset</option>)
  return <div className="form-inline form-group">
    <div className="input-group">
    <div className="input-group-addon">Asset</div>
    <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
      {options}
    </select>
    </div>
    </div>
}

let AssetAliasWidget = connect(
  (state) => ({ assets: assets.selectors.getItemList(state) })
)(AssetAliasWidgetUnconnected)

function ChoosePublicKeyWidget(props: { input: ChoosePublicKeyInput,
                                        handleChange: (e)=>undefined }) {
  if (props.input.keyMap === undefined) throw 'keyMap is undefined'
  let options : any[] = []
  let map: {[s: string]: KeyData} = props.input.keyMap
  for (let key in map) {
    options.push(<option key={key} value={key}>{key}</option>)
  }
  options.unshift(<option key="" value="">Select Public Key</option>)
  return <div className="form-inline form-group">
    <div className="input-group">
    <div className="input-group-addon">Pubkey</div>
    <select id={props.input.name} className="form-control with-addon" value={props.input.value} onChange={props.handleChange}>
      {options}
    </select>
    </div>
    </div>
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
    case "durationInput": return DurationWidget
    case "blocksDurationInput": return PositiveNumberWidget
    case "secondsDurationInput": return PositiveNumberWidget
    case "timeInput": return TimeWidget
    case "timestampTimeInput": return PositiveNumberWidget
    case "blockheightTimeInput": return PositiveNumberWidget
    case "addressInput": return AddressWidget
    case "mintimeInput": return MintimeWidget
    case "maxtimeInput": return MaxtimeWidget
    case "valueInput": return ValueWidget
    case "accountAliasInput": return AccountAliasWidget
    case "assetAmountInput": return AssetAmountWidget
    case "assetAliasInput": return AssetAliasWidget
    case "amountInput": return AmountWidget
    case "amountInput": return AmountWidget
    case "addressInput": return AddressWidget
    case "choosePublicKeyInput": return ChoosePublicKeyWidget
    default: return ParameterWidget
  }
}

function InputErrorUnconnected(props: { state, input: Input, showErrors: boolean }) {
  let isValid = validateInput(props.input)
  return (isValid || !props.showErrors) ? <div /> : <div className="alert alert-danger">Invalid input</div>
}

function mapToInputProps(state, inputsById: {[s: string]: Input}, id: string) {
  let input = inputsById[id]
  let showErrors = getShowErrors(state)
  if (input === undefined) throw "bad input ID: " + id
  if (input.type === "generatePublicKeyInput" ||
      input.type === "generateHashInput" ||
      input.type === "generateStringInput") {
    try {
      let computedValue = computeDataForInput(id, inputsById)
      return {
        input: input,
        computedValue: computedValue,
        showErrors: showErrors
      }
    } catch(e) {
      console.log(e)
    }
  }
  if (input.type === "generateSignatureInput") {
    return {
      input: input,
      computedValue: "",
      showErrors: showErrors//getSignatureData(state, id, inputsById)
    }
  }
  return {
    input: input,
    showErrors: showErrors
  }
}

function mapStateToSpendInputProps(state, ownProps: { id: string }) {
  let inputsById = getSpendInputMap(state)
  return mapToInputProps(state, inputsById, ownProps.id)
}

function mapStateToContractInputProps(state, ownProps: { id: string }) {
  let inputsById = getInputMap(state)
  return mapToInputProps(state, inputsById, ownProps.id)
}

function mapDispatchToContractInputProps(dispatch, ownProps: { id: string }) {
  return {
    handleChange: (e) => {
      dispatch(updateInput(ownProps.id, e.target.value.toString()))
    }
  }
}

function mapDispatchToSpendInputProps(dispatch, ownProps: { id: string} ) {
  return {
    handleChange: (e) => {
      dispatch(updateClauseInput(ownProps.id, e.target.value.toString()))
    }
  }
}

export function getWidget(id: string): JSX.Element {
  let inputContext = id.split(".").shift() as InputContext
  let type = id.split(".").pop() as InputType
  let widgetTypeConnected
  let InputError
  if (inputContext === "contractParameters") {
    widgetTypeConnected = connect(
      mapStateToContractInputProps,
      mapDispatchToContractInputProps
    )(getWidgetType(type))
    InputError = connect(
      mapStateToContractInputProps
    )(InputErrorUnconnected)
  } else {
    widgetTypeConnected = connect(
      mapStateToSpendInputProps,
      mapDispatchToSpendInputProps
    )(getWidgetType(type))
    InputError = connect(
      mapStateToSpendInputProps
    )(InputErrorUnconnected)
  }
  return <div key={"container(" + id + ")"}>
    {React.createElement(widgetTypeConnected, { key: "connect(" + id + ")", id: id })}
    <InputError id={id} key={"error(" + id + ")"} />
  </div>
}

function ContractParametersUnconnected(props: { parameterIds: string[] }) {
  if (props.parameterIds.length === 0) return <div />
  let parameterInputs = props.parameterIds.map((id) => {
    return <div key={id} className="argument">{getWidget(id)}</div>
  })
  return <section style={{wordBreak: 'break-all'}}>
    <h4>Contract Arguments</h4>
    <form className="form">
    {parameterInputs}
  </form></section>
}

function mapStateToContractParametersProps(state) {
  return {
    parameterIds: getParameterIds(state)
  }
}

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

export const ContractParameters = connect(
  mapStateToContractParametersProps
)(ContractParametersUnconnected)

export const ClauseParameters = connect(
  (state) => ({ parameterIds: getClauseParameterIds(state) })
)(ClauseParametersUnconnected)