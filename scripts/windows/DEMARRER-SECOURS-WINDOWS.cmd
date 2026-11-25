@echo off
setlocal
cd /d "%~dp0..\.."
node scripts\run-server.mjs --safe-mode
if errorlevel 1 pause
