# Chain Ruby SDK

## Usage

### Get the gem

The Ruby SDK is available [via Rubygems](https://rubygems.org/gems/chain-sdk). Make sure to use the most recent version whose major and minor components (`major.minor.x`) match your version of Chain Core. Ruby 2 is required.

For most applications, you can simply add the following to your `Gemfile`:

```
gem 'chain-sdk', '~> 1.0.0', require: 'chain'
```

### In your code

```
chain = Chain::Client.new
signer = Chain::HSMSigner.new
```

## Testing

To run integration tests, run a configured, empty Chain Core on http://localhost:1999. Then run:

```
bundle exec rspec
```
