# Chain .NET SDK

## Usage

### Get the library

The .NET SDK is available [via Nuget](https://www.nuget.org/packages/Chain.Sdk). Make sure to use the most recent version whose major and minor components (`major.minor.x`) match your version of Chain Core.

### In your code

```
var chain = new Chain.Sdk.Client();
var signer = new Chain.Sdk.HSMSigner();
```

## Testing

To run integration tests, run a configured, empty Chain Core on http://localhost:1999. Then run:

```
dotnet test
```
