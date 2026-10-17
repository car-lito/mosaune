@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"
if not exist "build\mosaune-extension-lab-windows-amd64.exe" (
 echo Le laboratoire n'a pas encore ete construit. Consultez extension-kit\README.md.
 pause
 exit /b 1
)
"build\mosaune-extension-lab-windows-amd64.exe"
if errorlevel 1 (
 echo Le test n'a pas termine correctement. Gardez le message pour le signaler.
 pause
 exit /b 1
)
echo.
echo Vous pouvez fermer cette fenetre. Votre serveur Mosaune n'a pas ete modifie.
pause
