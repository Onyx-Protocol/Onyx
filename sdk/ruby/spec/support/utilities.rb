module Utilities
  def chain
    unless @chain
      @chain = Chain::Client.new
    end
    @chain
  end

  # TODO(dominic): refactor? used by main integration_spec
  def balance_by_asset_alias(balances)
    balances.reduce({}) do |memo, b|
      memo[b.sum_by['asset_alias']] = b.amount
      memo
    end
  end

end
