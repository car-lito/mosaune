param([string]$Go='go')
$ErrorActionPreference='Stop'
$kitRoot=Split-Path -Parent $PSScriptRoot
$previousDirectory=Get-Location
$previousGoOS=$env:GOOS
$previousGoArch=$env:GOARCH
$previousCGO=$env:CGO_ENABLED
try {
 Set-Location -LiteralPath $kitRoot
 $env:CGO_ENABLED='0'
 $env:GOOS='wasip1'; $env:GOARCH='wasm'
 & $Go build -trimpath '-ldflags=-s -w' -o providers/tmdb/provider.wasm ./providers/tmdb/source
 if($LASTEXITCODE -ne 0){throw 'Compilation du fournisseur TMDB impossible.'}
 & $Go build -trimpath '-ldflags=-s -w' -o examples/demo/provider.wasm ./examples/demo/source
 if($LASTEXITCODE -ne 0){throw 'Compilation de la démonstration impossible.'}
 & $Go build -trimpath '-ldflags=-s -w' -o runtime/testdata/probe.wasm ./runtime/testdata/probe
 if($LASTEXITCODE -ne 0){throw 'Compilation des tests impossible.'}
 $env:GOOS=''; $env:GOARCH=''
 & $Go test ./...
 if($LASTEXITCODE -ne 0){throw 'Les tests ont échoué.'}
 & $Go vet ./...
 if($LASTEXITCODE -ne 0){throw 'La vérification statique a échoué.'}
 foreach($target in @(@('windows','amd64','.exe'),@('linux','amd64',''),@('linux','arm64',''))){
  $env:GOOS=$target[0];$env:GOARCH=$target[1]
  $destination="../build/mosaune-extension-lab-$($target[0])-$($target[1])$($target[2])"
  & $Go build -trimpath '-ldflags=-s -w' -o $destination ./cmd/lab
  if($LASTEXITCODE -ne 0){throw "Compilation impossible : $destination"}
 }
} finally {
 $env:GOOS=$previousGoOS;$env:GOARCH=$previousGoArch;$env:CGO_ENABLED=$previousCGO
 Set-Location -LiteralPath $previousDirectory
}
