# Chain Ruby SDK

## Usage

### Installing the library

#### Via Rubygems

TBA

#### Via downloaded .gem

Install the gem into your gem library:

```
gem install --local chain-sdk-<VERSION>.gem
```

### In your code

```
require 'chain'

chain = Chain::Client.new
```

## Testing

To run integration tests, run a configured, empty Chain Core on http://localhost:1999. Then run:

```
bundle exec rspec
```
