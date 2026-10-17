param([string]$Go='go',[string]$Node='node')
$ErrorActionPreference='Stop'
$goCommand=(Get-Command $Go).Source
$nodeCommand=(Get-Command $Node).Source
$projectRoot=Split-Path -Parent $PSScriptRoot
$previousDirectory=Get-Location
$previousGoOS=$env:GOOS
$previousGoArch=$env:GOARCH
$previousCGO=$env:CGO_ENABLED
try {
 Set-Location -LiteralPath $projectRoot
 $env:CGO_ENABLED='0'
 $env:GOOS='wasip1'; $env:GOARCH='wasm'
 Push-Location extension-kit
 try {
  & $goCommand build -trimpath '-ldflags=-s -w' -o providers/tmdb/provider.wasm ./providers/tmdb/source
  if($LASTEXITCODE -ne 0){throw 'Compilation du fournisseur TMDB impossible.'}
 } finally {Pop-Location}
 & $nodeCommand node_modules/vite/bin/vite.js build
 if($LASTEXITCODE -ne 0){throw 'Compilation interface impossible.'}
 $package=Get-Content -Raw package.json | ConvertFrom-Json
 foreach($target in @(@('windows','amd64','.exe'),@('linux','amd64',''),@('linux','arm64',''))){
  $env:GOOS=$target[0];$env:GOARCH=$target[1]
  $destination="build/mosaune-$($package.version)-$($target[0])-$($target[1])$($target[2])"
  & $goCommand build -trimpath '-ldflags=-s -w' -o $destination .
  if($LASTEXITCODE -ne 0){throw "Compilation impossible : $destination"}
 }
} finally {
 $env:GOOS=$previousGoOS;$env:GOARCH=$previousGoArch;$env:CGO_ENABLED=$previousCGO
 Set-Location -LiteralPath $previousDirectory
}
