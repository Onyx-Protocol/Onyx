require 'chain'

# snippet connect-with-token
client = Chain::Client.new({
  url: 'https://remote-server-url:1999',
  access_token: 'token:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7'
})
# endsnippet

# Create client without fake token for next example
client = Chain::Client.new()

# snippet create-read-only
token = client.access_tokens.create({
  id: 'new_access_token'
})

client.authorization_grants.create({
  guard_type: 'access_token',
  guard_data: {
    id: token.id
  },
  policy: 'client-readonly'
})
# endsnippet
