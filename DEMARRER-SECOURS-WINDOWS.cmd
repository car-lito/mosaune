@echo off
setlocal
cd /d "%~dp0"
if not exist "build\mosaune-0.0.38-windows-amd64.exe" (
  echo Le programme n'a pas encore ete construit. Consultez README.md.
  pause
  exit /b 1
)
title Projet Mosaune - mode de secours
echo Mosaune 0.0.38 - mode de secours sans extensions actives
echo Fermez d'abord le serveur normal si celui-ci est ouvert.
echo Adresse : http://127.0.0.1:47831
echo Gardez cette fenetre ouverte. Ctrl+C pour arreter.
start "" "http://127.0.0.1:47831"
"build\mosaune-0.0.38-windows-amd64.exe" -data-dir "%~dp0data" -safe-mode
if errorlevel 1 pause
