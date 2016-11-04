require_relative './lib/chain/version'

Gem::Specification.new do |s|
  s.name = 'chain-sdk'
  s.version = Chain::VERSION
  s.authors = ['Chain Engineering']
  s.description = 'The Official Ruby SDK for the Chain Core Developer Edition'
  s.summary = 'The Official Ruby SDK for the Chain Core Developer Edition'
  s.licenses = ['Apache-2.0']

  s.files = ['README.md']
  s.files += Dir['lib/**/*.rb']

  s.require_path = 'lib'

  s.add_development_dependency 'bundler', '~> 1.0'
  s.add_development_dependency 'rspec', '~> 3.5.0', '>= 3.5.0'
end
