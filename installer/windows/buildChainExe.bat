SET certPath=%1
SET certPass=%2
SET version=%3

IF "%version%"=="" (
  SET version=Latest
)

SET currentDirectory=%cd%

signtool sign -v -f %certPath% -p %certPass% cored.exe
signtool sign -v -f %certPath% -p %certPass% ChainMgr/ChainMgr.exe
cd ChainPackage
candle -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wxs
light -ext WixHttpExtension -ext WixUtilExtension ChainCoreInstaller.wixobj
signtool sign -v -f %certPath% -p %certPass% cab1.cab
cd ../ChainBundle
candle Bundle.wxs -arch x64 -ext WixBalExtension -dChainPackage.TargetPath=%currentDirectory%\ChainPackage\ChainCoreInstaller.msi -dPostgresPackage.TargetPath=%currentDirectory%\Postgres\postgresql-9.5.5-1-windows-x64.exe -dVCRPackage.TargetPath=%currentDirectory%\Postgres\vcredist_x64.exe
light Bundle.wixobj -ext WixBalExtension
insignia -ib Bundle.exe -o engine.exe
signtool sign -v -f %certPath% -p %certPass% engine.exe
insignia -ab engine.exe Bundle.exe -o Chain_Core_%version%.exe -v
signtool sign -v -f %certPath% -p %certPass% Chain_Core_%version%.exe
