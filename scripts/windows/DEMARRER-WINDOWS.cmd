@echo off
setlocal
cd /d "%~dp0..\.."
node scripts\run-server.mjs
if errorlevel 1 pause
