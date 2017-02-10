require 'chain'
require 'webmock'

include WebMock::API
WebMock.enable!

describe Chain::Connection do

  example 'works with mixtures of relative and absolute paths' do
    stub_request(:any, 'foo.test/bar').to_return(body: '{}', headers: {'Chain-Request-ID' => 'test'})
    stub_request(:any, 'foo.test/bar/baz').to_return(body: '{}', headers: {'Chain-Request-ID' => 'test'})

    expect {
      Chain::Connection.new(url: 'http://foo.test').request('bar')
      Chain::Connection.new(url: 'http://foo.test').request('/bar')
      Chain::Connection.new(url: 'http://foo.test/bar').request('baz')
      Chain::Connection.new(url: 'http://foo.test/bar').request('/baz')
    }.not_to raise_exception
  end

end
