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

1. Set up the HSM(s) and the client host(s) according to the
   directions from Thales
   1.1 Install CodeSafe software on client host(s)
   1.2 Initialize the Remote File System
   1.3 Configure host(s) and HSM(s) with each other’s IP addresses
   1.4 Create the “Security World” with the Secure Execution Engine
     (SEE) enabled, along with SEE debugging
   1.5 Create an administrator cardset and an operator cardset

   These steps are complex and involve policy decisions and other
   choices.  Refer to the Thales documentation for details.  Note:
   Chain has tested this using 64-bit Ubuntu Linux on the client host.

2. On the client host, use the “nfast” utilities from Thales to create
   a _user-data signing key_ and a signed copy of the userdata file.
   2.1 `generatekey --batch seeinteg plainname=xprvseemoduledevusk`
   2.2 `tct2 --sign-and-pack -k xprvseemoduledevusk --machine-key=02cf0cd76a8e726ffba2e4a1d2648b447d505aaa -o userdata.sar -i userdata.bin`
   
   The hex value here is the hash of the key that Chain used to sign
   the SEE machine. **Important**: this hash is of a _development key_ and
   should not be trusted for production.
   
3. On the client host (RFS server), copy the signed firmware and userdata files to the custom-seemachines directory. 
   Here, `a` and `b` have been chosen to simplify entering the file names on the HSM front panel. 
   This command must be run on each module.
   3.1 `cp xprvseemodule.sar /opt/nfast/custom-seemachines/a`
   3.2 `cp userdata.sar /opt/nfast/custom-seemachines/b`

4. On the HSM, use the front panel to load the machine image and userdata file. 
   This must be done for each HSM.
   4.1 Choose `CodeSafe` on the front panel 
   4.2 Enter`a` for the machine image 
   4.3 Enter`b` for the userdata file
   4.4 Choose`SEElib` for the world type
   4.5 Enter`chainenclave` for the name

5. On the client host, use the `xprvseetool` binary from Chain to
   create a private key in the HSM.
   5.1 `xprvseetool gen xprvseemoduledevusk`

   The output of this tool is the hex-encoded public key corresponding
   to the new private key. Save it for use in asset issuance programs
   and/or transaction control programs.

   The development version of Chain’s Thales integration operates on a
   single public/private keypair. The production version of Chain Core
   Enterprise Edition will permit the creation, storage, and use of
   arbitrarily many keys.

6. On the client host, run the `signerd` binary from Chain.
   6.1 `signerd`
   
   This launches an HTTP+JSON server listening for `/sign-transaction`
   requests on port 8080. The request and response formats for this
   endpoint are byte-for-byte compatible with the
   `/mockhsm/sign-transaction` endpoint in Chain Core.
