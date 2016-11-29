# Chain Core for Windows

## Installing Chain Core

To install Chain Core Developer Edition for Windows, please visit [our downloads page](https://chain.com/docs/core/get-started/install).

## Building the Windows Installer
### Dependencies

These instructions assume that your PATH includes the wix tools binary. My wix tools binary is located at `C:\Program Files (x86)\WiX Toolset v3.10\bin`.

We don't check `.exe`s into git, so you'll have to provide them yourself. There are four `.exe`s:

1. `cored.exe`
2. `ChainMgr.exe`
3. The Postgres Installer, called `postgresql-9.5.5-1-windows-x64.exe`
4. The VC++ Redistributable for Visual Studio 2013 (which is required to run the Postgres Installer), called `vcredist_x64.exe`

You will want to put them into this directory like this:

```
|-windows
   | cored.exe
   |-ChainBundle
   |-ChainMgr
      | ChainMgr.exe
   |-ChainPackage
   |-Postgres
      | postgresql-9.5.5-1-windows-x64.exe
      | vcredist_x64.exe
```

`ChainMgr.exe` and `cored.exe` can be compiled from any machine using `GOOS` and `GOARCH`:

```
GOOS=windows GOARCH=amd64 go build chain/cmd/cored
GOOS=windows GOARCH=amd64 go build chain/installer/windows/ChainMgr
```

The Postgres Installer can be downloaded from http://www.enterprisedb.com/products-services-training/pgdownload

The VC++ Redistributable can be downloaded from https://www.microsoft.com/en-us/download/details.aspx?id=40784

Make sure you have the 64-bit versions. Chain Core Windows does not support 32-bit. Do not actually run these installers, just provide them.

### Build

The chain bundler is capable of building multiple .msi's and .exe's into a single installer .exe.

First, build the chain core msi. To do this, from inside of `installer/windows` run:

```
cd ChainPackage
candle -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wxs
```

This generates `ChainPackage/ChainCoreInstaller.wixobj`

Next, run

```
light -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wixobj
```

to generate `ChainPackage/ChainCoreInstaller.msi`.

Next, build the Chain Bundle. Run

```
cd ../ChainBundle
candle Bundle.wxs \
  -arch x64 \
  -ext WixBalExtension \
  -dChainPackage.TargetPath='Z:\chain\installer\windows\ChainPackage\ChainCoreInstaller.msi' \
  -dPostgresPackage.TargetPath='Z:\chain\installer\windows\Postgres\postgresql-9.5.5-1-windows-x64.exe' \
  -dVCRPackage.TargetPath='Z:\chain\installer\windows\Postgres\vcredist_x64.exe'
```
(but obviously sub out your path for my target paths)

This generates `ChainBundle/Bundle.wixobj`. Next, run

```
light Bundle.wixobj -ext WixBalExtension
```

This generates Bundle.exe in your current working directory.

### Code Signing

In order for Chain to appear as the publisher, some of the files inside the installer need to be signed with the private key of Chain's code signing certificate.

To do this, you will need:

* A `.pfx` file ([generated from the certificate](https://www.digicert.com/code-signing/exporting-code-signing-certificate.htm)). In order to prevent as many security warnings as possible, the certificate should be an EV Certificate.
* `signtool`, which is packaged inside the Windows SDK. The Windows SDK is from Microsoft and can be downloaded here: https://developer.microsoft.com/en-us/windows/downloads/windows-10-sdk

You will need to sign:

* `cored.exe`
* `ChainMgr.exe`
* `cab1.cab` (a build artifact from building `ChainCoreInstaller.msi`)
* `Bundle.exe`
* The Burn engine contained inside `Bundle.exe`

To sign a file, do the following. The following commands assume that both the Wix Tools binaries and the signtool are in your path. My signtool was installed at `C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`.

1. Sign `cored.exe` and `ChainMgr.exe` by running signtool:

```
signtool sign -v -f [x.pfx] -p [password] [file to sign]
```

2. Build `ChainCoreInstaller.msi` using `candle` and `light`, as above.
3. Sign `cab1.cab`, the build artifact, using `signtool`:

```
signtool sign -v -f [x.pfx] -p [password] ChainPackage/cab1.cab
```

4. Build `Bundle.exe` using `candle` and `light`, as above.
5. Extract `engine.exe` from the bundle and sign it:

```
insignia -ib Bundle.exe -o engine.exe
signtool sign -v -f [x.pfx] -p [password] engine.exe
insignia -ab engine.exe Bundle.exe -o Chain_Core_Latest.exe -v
```

6. Sign `Chain_Core_Latest.exe` directly:

```
signtool sign -v -f [x.pfx] -p [password] ChainBundle/Chain_Core_Latest.exe
```

Clicking on `Chain_Core_Latest.exe` will install Chain Core as an application on your PC.
