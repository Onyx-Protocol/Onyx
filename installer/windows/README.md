# windows-installer

These instructions assume that your PATH includes the wix tools binary. My wix tools binary is located at `C:\Program Files (x86)\WiX Toolset v3.10\bin`.

The chain bundler is capable of building multiple .msi's and .exe's into a single installer .exe. 

First, build the chain core msi. To do this, from inside of `ChainPackage` run: 

`candle -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wxs`

This generates `ChainCoreInstaller.wixobj`

Next, run 

`light -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wixobj`

to generate `ChainCoreInstaller.msi`. 

Next, `cd ../ChainBundle` and run 

```
candle Bundle.wxs \
  -ext WixBalExtension \
  -dChainPackage.TargetPath='C:\Users\tess\src\windows-installer\ChainMSI\ChainPackage\ChainCoreInstaller.msi' \
  -dPostgresPackage.TargetPath='C:\Users\tess\src\windows-installer\ChainMSI\Postgres\postgresql-9.5.4-2-windows-x64.exe'
```
(but obviously sub out your path for my target paths) 

This generates `bundle.wixobj`. Next, run

`light Bundle.wixobj -ext WixBalExtension`

This generates Bundle.exe in your current working directory. Clicking on Bundle.exe will install Chain Core as an application on your PC. 
