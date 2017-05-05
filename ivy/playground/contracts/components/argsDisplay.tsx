import * as React from 'react'
import { connect } from 'react-redux'
import { getSpendInputMap } from '../selectors'
import { Input, ParameterInput, NumberInput, BooleanInput, StringInput,
         ProvideStringInput, GenerateStringInput, HashInput, DurationInput, SecondsDurationInput,
         BlocksDurationInput, TimeInput, BlockheightTimeInput, TimestampTimeInput,
         PublicKeyInput, GeneratePublicKeyInput, ProvidePublicKeyInput, GenerateHashInput,
         ProvideHashInput, InputType, ComplexInput, ValueInput,
         AssetAliasInput, AccountAliasInput, AssetAmountInput, AmountInput, AddressInput } from '../../inputs/types'
import { getChild, getParameterIdentifier, getGenerateStringInputValue, computeDataForInput } from '../../inputs/data'
import { typeToString } from 'ivy-compiler'
import { getItemMap as getAssetMap } from '../../assets/selectors'
import { getItemMap as getAccountMap } from '../../accounts/selectors'

import { getSpendContractParameterSelector, getSpendParameterIds } from '../selectors'

function getChildWidget(input: ComplexInput) {
  return getWidget(getChild(input))
}

function ParameterWidget(props: { input: ParameterInput }) {
  return <div key={props.input.name}>
    <label>{getParameterIdentifier(props.input)}: <span className='type-label'>{typeToString(props.input.valueType)}</span></label>
    <div>{getChildWidget(props.input)}</div>
  </div>
}

function GeneratePublicKeyWidget(props: { input: GeneratePublicKeyInput, computedValue: string }) {
  return <div>
      <pre>{props.computedValue}</pre>
      <div className="nested">
      <div className="description">derived from:</div>
      <label className="type-label">PrivateKey</label>
    {getChildWidget(props.input)}
  </div></div>
}

function ValueWidget(props: { input: ValueInput }) {
  return <div>
    {getWidget(props.input.name + ".accountAliasInput")}
    {getWidget(props.input.name + ".assetAmountInput")}
  </div>
}

function AssetAmountWidget(props: { input: AssetAmountInput }) {
  return <div>
    {getWidget(props.input.name + ".assetAliasInput")}
    {getWidget(props.input.name + ".amountInput")}
  </div>
}

function AssetAliasWidgetUnconnected(props: { input: AssetAliasInput,
                                              assetMap: {[s: string]: any}}) {
  return <pre>{props.assetMap[props.input.value].alias}</pre>
}

let AssetAliasWidget = connect(
  (state) => ({ assetMap: getAssetMap(state) })
)(AssetAliasWidgetUnconnected)

function AccountAliasWidgetUnconnected(props: { input: AccountAliasInput,
                                              accountMap: {[s: string]: any}}) {
  return <pre>{props.accountMap[props.input.value].alias}</pre>
}

let AccountAliasWidget = connect(
  (state) => ({ accountMap: getAccountMap(state) })
)(AccountAliasWidgetUnconnected)

function TextWidget(props: { input: Input }) {
  return <div><pre>{props.input.value}</pre></div>
}

function ComputedWidget(props: { input: AddressInput }) {
  return <div><pre>{props.input.computedData}</pre></div>
}

function GenerateHashWidget(props: { input: GenerateHashInput, computedValue: string }) {
  return <div>
    <pre>{props.computedValue}</pre>
    <div className="nested">
      <div className="description">{props.input.hashType.hashFunction} of:</div>
      <label className="type-label">{typeToString(props.input.hashType.inputType)}</label>
      {getChildWidget(props.input)}
    </div>
  </div>
}

function ParentWidget(props: { input: ComplexInput }) {
  return getChildWidget(props.input)
}

function BlocksDurationWidget(props: { input: BlocksDurationInput }) {
  return <div>{props.input.value} blocks</div>
}

function SecondsDurationWidget(props: { input: BlocksDurationInput }) {
  let numIncrements = parseInt(props.input.value, 10)
  return <div>{numIncrements * 512} seconds</div>
}

function BlockheightTimeWidget(props: { input: BlockheightTimeInput }) {
  return <div>block {props.input.value}</div>
}

function TimestampTimeWidget(props: { input: TimestampTimeInput }) {
  return <div>{props.input.value} seconds after the epoch</div> // super lazy for now!
}

function GenerateStringWidget(props: { input: GenerateStringInput }) {
  return <div><pre>{getGenerateStringInputValue(props.input)}</pre></div>
}


function getWidgetType(type: InputType): ((props: { input: Input }) => JSX.Element) {
  switch (type) {
    case "stringInput":
    case "hashInput":
    case "durationInput":
    case "timeInput":
      return ParentWidget
    case "generatePublicKeyInput": return GeneratePublicKeyWidget
    case "generateHashInput": return GenerateHashWidget
    case "blocksDurationInput": return BlocksDurationWidget
    case "secondsDurationInput": return SecondsDurationWidget
    case "timestampTimeInput": return TimestampTimeWidget
    case "blockheightTimeInput": return BlockheightTimeWidget
    case "generateStringInput": return GenerateStringWidget
    case "valueInput": return ValueWidget
    case "assetAmountInput": return AssetAmountWidget
    case "accountAliasInput": return AccountAliasWidget
    case "assetAliasInput": return AssetAliasWidget
    case "addressInput":
    case "publicKeyInput": return ComputedWidget
    case "numberInput":
    case "booleanInput":
    case "provideStringInput":
    case "providePublicKeyInput":
    case "provideHashInput":
    case "generatePrivateKeyInput":
    case "providePrivateKeyInput":
    case "accountAliasInput":
    case "assetAliasInput":
    case "providePrivateKeyInput":
    case "amountInput":
      return TextWidget
    default: return ParameterWidget
  }
}

function getWidget(id: string): JSX.Element {
  let type = id.split(".").pop() as InputType
  let widgetTypeConnected = connect(
    (state) => ({ input: getSpendContractParameterSelector(id)(state) })
  )(getWidgetType(type))
  if (type === "generateHashInput" || type === "generatePublicKeyInput") {
    widgetTypeConnected = connect(
      (state) => {
        return {
          input: getSpendContractParameterSelector(id)(state),
          computedValue: computeDataForInput(id, getSpendInputMap(state))
        }
      }
    )(getWidgetType(type))
  }
  return React.createElement(widgetTypeConnected, { key: "connect(" + id + ")", id: id })
}

function SpendInputsUnconnected(props: { spendInputIds: string[] }) {
  if (props.spendInputIds.length === 0) return <div />
  let spendInputWidgets = props.spendInputIds.map((id) => {
    return <div key={id} className="argument">{getWidget(id)}</div>
  })
  return <section style={{ wordBreak: 'break-all'}}>
    <h4>Contract Arguments</h4>
    <form className="form">
    {spendInputWidgets}
  </form></section>
}

const SpendInputs = connect(
  (state) => ({ spendInputIds: getSpendParameterIds(state) })
)(SpendInputsUnconnected)

export default SpendInputs