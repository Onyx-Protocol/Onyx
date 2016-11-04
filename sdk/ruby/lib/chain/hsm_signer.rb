require_relative './connection'
require_relative './errors'
require_relative './mock_hsm'
require_relative './transaction'

module Chain
  class HSMSigner

    def initialize
      @xpubs_by_signer = {}
    end

    def add_key(xpub_or_key, signer_conn)
      xpub = xpub_or_key.is_a?(MockHSM::Key) ? xpub_or_key.xpub : xpub_or_key
      @xpubs_by_signer[signer_conn] ||= []
      @xpubs_by_signer[signer_conn] << xpub
      @xpubs_by_signer[signer_conn].uniq!
    end

    def sign(tx_template)
      return tx_template if @xpubs_by_signer.empty?

      @xpubs_by_signer.each do |signer_conn, xpubs|
        tx_template = signer_conn.singleton_batch_request(
          '/sign-transaction',
          transactions: [tx_template],
          xpubs: xpubs,
        ) { |item| Transaction::Template.new(item) }
      end

      tx_template
    end

    def sign_batch(tx_templates)
      if @xpubs_by_signer.empty?
        # Treat all templates as if signed successfully.
        successes = tx_templates.each_with_index.reduce({}) do |memo, (t, i)|
          memo[i] = t
          memo
        end
        BatchResponse.new(successes: successes)
      end

      # We need to work towards a single, final BatchResponse that uses the
      # original indexes. For the next cycle, we should retain only those
      # templates for which the most recent sign response was successful, and
      # maintain a mapping of each template's index in the upcoming request
      # to its original index.

      orig_index = (0...tx_templates.size).to_a
      errors = {}

      @xpubs_by_signer.each do |signer_conn, xpubs|
        next_tx_templates = []
        next_orig_index = []

        batch = signer_conn.batch_request(
          '/sign-transaction',
          transactions: tx_templates,
          xpubs: xpubs,
        ) { |item| Transaction::Template.new(item) }

        batch.successes.each do |i, template|
          next_tx_templates << template
          next_orig_index << orig_index[i]
        end

        batch.errors.each do |i, err|
          errors[orig_index[i]] = err
        end

        tx_templates = next_tx_templates
        orig_index = next_orig_index

        # Early-exit if all templates have encountered an error.
        break if tx_templates.empty?
      end

      successes = tx_templates.each_with_index.reduce({}) do |memo, (t, i)|
        memo[orig_index[i]] = t
        memo
      end

      BatchResponse.new(
        successes: successes,
        errors: errors,
      )
    end

  end
end
