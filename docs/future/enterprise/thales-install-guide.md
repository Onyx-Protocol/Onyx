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
        - If running multiple hosts, the module numbers will be configured at this time.
   - Create the “Security World” with the Secure Execution Engine
     (SEE) enabled, along with SEE debugging
   - Create an administrator cardset and an operator cardset

- Thales configuration involves many policy choices related to SEE Activation. To check which SEE features have been enabled, on the client host:
   - `fet` 
   - If `SEE Activation (Restricted)` is configured, Thales may need to assist with additional certifcations or licenses prior to proceeding. Refer to the Thales documentation or contact your Thales representative for details.
    

   These steps are complex and involve policy decisions and other
   choices.  Refer to the Thales documentation for details.  Note:
   Chain has tested this using 64-bit Ubuntu Linux on the client host.

-  On the client host, use the “nfast” utilities from Thales to create
   a _user-data signing key_ and a signed copy of the userdata file.
   - `generatekey --batch seeinteg plainname=xprvseemoduledevusk`
   - `tct2 --sign-and-pack -k xprvseemoduledevusk --machine-key=02cf0cd76a8e726ffba2e4a1d2648b447d505aaa -o userdata.sar -i userdata.bin`
   
   The hex value here is the hash of the key that Chain used to sign
   the SEE machine. **Important**: this hash is of a _development key_ and
   should not be trusted for production.
   
-  On the client host (RFS server), copy the signed firmware and userdata files to the custom-seemachines directory. These commands must be run on each module.
-  Here, `a` and `b` have been chosen to simplify entering the file names on the HSM front panel. These names can be changed if desired.
   - `cp xprvseemodule.sar /opt/nfast/custom-seemachines/a`
   - `cp userdata.sar /opt/nfast/custom-seemachines/b`

-  On the HSM, use the front panel to load the machine image and userdata file. 
   This must be done for each HSM. Note: there is no need to reboot the modules.
   - Choose `CodeSafe` on the front panel 
   - Enter `a` (or name of choice) for the machine image 
   - Enter `b` (or name of choice) for the userdata file
   - Choose `SEElib` for the world type
   - Enter `chainenclave` for the name

-  On the client host, check that the SEE world is running on each module using stattree.
   - `stattree PerModule 1 ModuleEnvStats`
   - The `MemAllocUser` value in the output should be nonzero

-  On the client host, use the `xprvseetool` binary from Chain to check that the client can access the SEE world on each module (update the module number in the command for each host).
   - `xprvseetool -m 1 seeversion`

-  On the client host, use the `xprvseetool` binary from Chain to
   create an `xprv` (for transactions) and a `prv` (for blocks) in the HSM.
   - `xprvseetool -i xprv0 genx xprvseemoduledevusk`
   - `xprvseetool -i prv0 gen xprvseemoduledevusk`

   The output of this tool is the hex-encoded public key corresponding
   to the new private key. Save it for use in asset issuance programs
   and/or transaction control programs.

   The development version of Chain’s Thales integration operates on a
   single public/private keypair. The production version of Chain Core
   Enterprise Edition will permit the creation, storage, and use of
   arbitrarily many keys.

-  On the client host, run the `signerd` binary from Chain (update the module number in the command for each host).  
   - `MODULE=1 KEY_IDENT=prv0 XPRV_KEY_IDENT=xprv0 signerd`
   
   This launches an HTTP+JSON server listening for `/sign-transaction`
   requests on port 8080. The request and response formats for this
   endpoint are byte-for-byte compatible with the
   `/mockhsm/sign-transaction` endpoint in Chain Core.
