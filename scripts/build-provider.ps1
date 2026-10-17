param([string]$Go='go',[Parameter(Mandatory=$true)][string]$Key,[string]$Version='0.4.0',[switch]$NewSigningKey)
$ErrorActionPreference='Stop'
$goCommand=(Get-Command $Go).Source
$projectRoot=Split-Path -Parent $PSScriptRoot
$keyFile=[IO.Path]::GetFullPath($Key)
$previousDirectory=Get-Location
$previousOS=$env:GOOS
$previousArch=$env:GOARCH
$previousCGO=$env:CGO_ENABLED
try {
 Set-Location -LiteralPath (Join-Path $projectRoot 'extension-kit')
 $env:CGO_ENABLED='0'
 $env:GOOS='wasip1';$env:GOARCH='wasm'
 & $goCommand build -trimpath "-ldflags=-s -w -X main.buildVersion=$Version" -o providers/tmdb/provider.wasm ./providers/tmdb/source
 if($LASTEXITCODE -ne 0){throw 'Compilation du fournisseur impossible.'}
 $env:GOOS=$previousOS;$env:GOARCH=$previousArch
 $packages=Join-Path $projectRoot 'packages'
 [IO.Directory]::CreateDirectory($packages) | Out-Null
 $arguments=@('run','./cmd/package-provider','-key',$keyFile,'-public',(Join-Path $projectRoot 'provider-package-key.pub'),'-version',$Version,'-code','providers/tmdb/provider.wasm','-out',(Join-Path $packages "tmdb-$Version.mosaune.zip"))
 if($NewSigningKey){$arguments+='-create-key'}
 & $goCommand @arguments
 if($LASTEXITCODE -ne 0){throw 'Signature du fournisseur impossible.'}
} finally {
 $env:GOOS=$previousOS;$env:GOARCH=$previousArch;$env:CGO_ENABLED=$previousCGO
 Set-Location -LiteralPath $previousDirectory
}
