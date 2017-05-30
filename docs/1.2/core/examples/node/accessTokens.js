const chain = require('chain-sdk')

const throwAway = () => {
// snippet connect-with-token
const client = new chain.Client({
  url: 'https://remote-server-url:1999',
  accessToken: 'token:25f658b749f154a790c8a3aeb57ea98968f51a991c4771fb072fcbb2fa63b6f7'
})
// endsnippet
}

// Create client without fake token for next example
const client = new chain.Client()

// snippet create-read-only
client.accessTokens.create({
  id: 'newAccessToken'
}).then(token =>
  client.authorizationGrants.create({
    guard_type: 'access_token',
    guard_data: {
      id: token.id
    },
    policy: 'client-readonly'
  })
)
// endsnippet
