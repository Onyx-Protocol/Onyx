# Using the Thales nShield Connect Hardware Security Module with Chain Core

The security of assets and accounts in Chain Core depends on strong
cryptographic keys that must be kept secure themselves. Chain
recommends the use of dedicated devices known as Hardware Security
Modules (HSMs) for protecting sensitive key material. Chain has
partnered with the vendor Thales to integrate its industry-leading HSM
hardware, the **nShield Connect**, with the Chain Core programming
interface.

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
   a _code-signing key_ and a signed copy of the Chain firmware.
   - `generatekey --batch seeinteg plainname=xprvseemoduledevcsk`
   - `tct2 --sign-and-pack -k xprvseemoduledevcsk --is-machine -o xprvseemodule.sar -i xprvseemodule.sxf`

   This step will not be required in the production version of Chain
   Core Enterprise Edition, which will include an already-signed copy
   of the Chain firmware.

-  On the client host, use the “nfast” utilities from Thales to create
   a _user-data signing key_ and a signed copy of the userdata file.
   - `generatekey --batch seeinteg plainname=xprvseemoduledevusk`
   - `tct2 --sign-and-pack -k xprvseemoduledevusk --machine-key-ident=xprvseemoduledevcsk -o userdata.sar -i userdata.bin`
   
-  On the client host, use the “nfast” utilities from Thales to
   install the signed firmware and userdata files.
   - `loadmache xprvseemodule.sar userdata.sar`

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
