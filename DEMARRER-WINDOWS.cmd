@echo off
setlocal
cd /d "%~dp0"
if not exist "build\mosaune-0.0.29-windows-amd64.exe" (
  echo Le programme n'a pas encore ete construit. Consultez README.md.
  pause
  exit /b 1
)
title Projet Mosaune
echo Mosaune 0.0.29 - bibliotheque persistante
echo Adresse : http://127.0.0.1:47831
echo Gardez cette fenetre ouverte. Ctrl+C pour arreter.
start "" "http://127.0.0.1:47831"
"build\mosaune-0.0.29-windows-amd64.exe" -data-dir "%~dp0data"
if errorlevel 1 (
  echo Si le serveur est deja ouvert, utilisez simplement son adresse dans le navigateur.
  pause
)
