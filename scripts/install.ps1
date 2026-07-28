<#
.SYNOPSIS
    Install git-fleet on Windows by downloading the release binary for the
    current architecture. No Go toolchain is required.

.DESCRIPTION
    Downloads the git-fleet release archive, extracts the binary into
    %LOCALAPPDATA%\git-fleet\bin, and adds that directory to the user PATH so
    the command is available in new terminals.

    Run:
        irm https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.ps1 | iex

.PARAMETER Version
    Version tag to install. Defaults to the latest release. Can also be set via
    the GIT_FLEET_VERSION environment variable.
#>
[CmdletBinding()]
param(
    [string]$Version = $env:GIT_FLEET_VERSION
)

$ErrorActionPreference = 'Stop'
$repo = 'black-osiris-technologies/git-fleet'
$binary = 'git-fleet'

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }

if (-not $Version) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $Version = $release.tag_name
}
if (-not $Version) {
    throw 'Could not determine the latest version.'
}

$number = $Version.TrimStart('v')
$archive = "${binary}_${number}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$Version/$archive"

$installDir = Join-Path $env:LOCALAPPDATA 'git-fleet\bin'
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName() + '.zip')
try {
    Write-Host "Downloading $binary $Version (windows/$arch)"
    Invoke-WebRequest -Uri $url -OutFile $tmp
    Expand-Archive -Path $tmp -DestinationPath $installDir -Force
}
finally {
    if (Test-Path $tmp) { Remove-Item $tmp -Force }
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$installDir*") {
    $newPath = if ([string]::IsNullOrEmpty($userPath)) { $installDir } else { "$userPath;$installDir" }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "Added $installDir to your user PATH. Restart your terminal to use git-fleet."
}

Write-Host "Installed $binary $Version to $installDir"
