# Using a Thales HSM with Chain Core

The security of assets and accounts in Chain Core depends on strong
cryptographic keys that must be kept secure themselves. Chain
recommends the use of dedicated devices known as Hardware Security
Modules (HSMs) for protecting sensitive key material. Chain has
partnered with the vendor Thales to integrate its industry-leading HSM
hardware, the **nShield Connect**, with the Chain Core programming
interface.

To begin using the Thales nShield Connect with Chain Core, you must
have the following files supplied by Chain:

- `xprvseemodule.sar`, the firmware (or “SEE machine”) to be loaded
  into the HSM;
- `userdata.bin`, a file of associated data to be signed and loaded
  into the HSM;
- `xprvseetool`, a Linux binary for creating a key in the HSM suitable
  for use in Chain Core;
- `signerd`, a Linux binary implementing a server for responding to
  transaction-signing requests.
  
You must also perform the following steps:

-  Set up the HSM(s) and the client host(s) according to the
   directions from Thales
   - Install CodeSafe software on client host(s)
   - Initialize the Remote File System
   - Configure host(s) and HSM(s) with each other’s IP addresses
   - Create the “Security World” with the Secure Execution Engine
     (SEE) enabled, along with SEE debugging
   - Create an administrator cardset and an operator cardset

   These steps are complex and involve policy decisions and other
   choices.  Refer to the Thales documentation for details.  Note:
   Chain has tested this using 64-bit Ubuntu Linux on the client host.

-  On the client host, use the “nfast” utilities from Thales to create
   a _user-data signing key_ and a signed copy of the userdata file.
   - `generatekey --batch seeinteg plainname=xprvseemoduledevusk`
   - `tct2 --sign-and-pack -k xprvseemoduledevusk --machine-key=68bcec164114318f31b2e353bef9e8b1ce67872e -o userdata.sar -i userdata.bin`
   
   The hex value here is the hash of the key that Chain used to sign
   the SEE machine. **Important**: this hash is of a _development key_ and
   should not be trusted for production.
   
-  On the client host, use the “nfast” utilities from Thales to
   install the signed firmware and userdata files.
   - `loadmache xprvseemodule.sar`

-  On the client host, use the `xprvseetool` binary from Chain to
   create a private key in the HSM.
   - `xprvseetool gen xprvseemoduledevusk`

   The output of this tool is the hex-encoded public key corresponding
   to the new private key. Save it for use in asset issuance programs
   and/or transaction control programs.

   The development version of Chain’s Thales integration operates on a
   single public/private keypair. The production version of Chain Core
   Enterprise Edition will permit the creation, storage, and use of
   arbitrarily many keys.

-  On the client host, run the `signerd` binary from Chain.
   - `signerd`
   
   This launches an HTTP+JSON server listening for `/sign-transaction`
   requests on port 8080. The request and response formats for this
   endpoint are byte-for-byte compatible with the
   `/mockhsm/sign-transaction` endpoint in Chain Core.
