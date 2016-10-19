# windows-installer

These instructions assume that your PATH includes the wix tools binary. My wix tools binary is located at `C:\Program Files (x86)\WiX Toolset v3.10\bin`.

### Dependencies 

The Postgres Installer can be downloaded from http://www.enterprisedb.com/products-services-training/pgdownload

The VC++ Redistributable can be downloaded from https://www.microsoft.com/en-us/download/search.aspx?q=redistributable+package&first=11

Make sure you have the 64-bit versions. Chain Core Windows does not support 32-bit.

### Build

The chain bundler is capable of building multiple .msi's and .exe's into a single installer .exe. 

First, build the chain core msi. To do this, from inside of `installer/windows` run: 

```
cd ChainPackage
candle -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wxs
```

This generates `ChainPackage/ChainCoreInstaller.wixobj`

Next, run 

`light -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wixobj`

to generate `ChainPackage/ChainCoreInstaller.msi`. 

Next, build the Chain Bundle. Run 

```
cd ../ChainBundle
candle Bundle.wxs \
  -arch x64 \
  -ext WixBalExtension \
  -dChainPackage.TargetPath='Z:\chain\installer\windows\ChainPackage\ChainCoreInstaller.msi' \
  -dPostgresPackage.TargetPath='Z:\chain\installer\windows\Postgres\postgresql-9.5.4-2-windows-x64.exe' \
  -dVCRPackage.TargetPath='Z:\chain\installer\windows\Postgres\vcredist_x64.exe'
```
(but obviously sub out your path for my target paths) 

This generates `ChainBundle/Bundle.wixobj`. Next, run

`light Bundle.wixobj -ext WixBalExtension`

This generates Bundle.exe in your current working directory. 

### Code Signing

In order for Chain to appear as the publisher, Bundle.exe needs to be signed with the private key of Chain's code signing certificate. 

To do this, first put a .pfx file ([generated from the certificate](https://www.digicert.com/code-signing/exporting-code-signing-certificate.htm)) into the `ChainBundle` directory. Then run the following commands. 

The following commands rely on signtool, which is a tool packaged inside the Windows SDK. The Windows SDK is from Microsoft and can be downloaded here: https://developer.microsoft.com/en-us/windows/downloads/windows-10-sdk 

The commands also assume that both the Wix Tools binaries and the signtool are in your path. My signtool was installed at `C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`. 

```
insignia -ib Bundle.exe -o engine.exe
signtool sign -v -f [x.pfx] -p [password] engine.exe
insignia -ab engine.exe Bundle.exe -o Bundle.exe -v
```


Clicking on Bundle.exe will install Chain Core as an application on your PC. 
