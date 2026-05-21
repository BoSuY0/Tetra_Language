param(
  [string]$Report = "",
  [string]$ReportDir = "",
  [switch]$Help
)

$ErrorActionPreference = "Stop"

function Show-Usage {
  @"
Usage: pwsh -File scripts/release/full_platform/windows-ui-runtime-smoke.ps1 [-Report FILE] [-ReportDir DIR]

Runs the Windows platform UI runtime smoke on a real Windows x64 target host and
validates the report. The report is only production evidence when the host is
Windows x64 and the runtime-backed UI smoke plus validator pass.
"@
}

if ($Help) {
  Show-Usage
  exit 0
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..\..\..")).Path

if ([string]::IsNullOrWhiteSpace($Report)) {
  if ([string]::IsNullOrWhiteSpace($ReportDir)) {
    $ReportDir = Join-Path $repoRoot "reports\full-platform-ui-runtime"
  } elseif (![System.IO.Path]::IsPathRooted($ReportDir)) {
    $ReportDir = Join-Path $repoRoot $ReportDir
  }
  $Report = Join-Path $ReportDir "windows-ui-runtime.json"
} elseif (![System.IO.Path]::IsPathRooted($Report)) {
  $Report = Join-Path $repoRoot $Report
}

if ($env:OS -ne "Windows_NT") {
  Write-Error "windows-x64 UI runtime production evidence requires a real Windows x64 host; current host is not Windows_NT"
  exit 1
}

$osArchitecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($osArchitecture -ne "X64") {
  Write-Error "windows-x64 UI runtime production evidence requires a real Windows x64 host; OSArchitecture is $osArchitecture"
  exit 1
}

$reportDirPath = Split-Path -Parent $Report
if (![string]::IsNullOrWhiteSpace($reportDirPath)) {
  New-Item -ItemType Directory -Force -Path $reportDirPath | Out-Null
}

Push-Location $repoRoot
try {
  go run ./tools/cmd/platform-ui-runtime-smoke --target windows-x64 --report $Report
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }

  go run ./tools/cmd/validate-windows-ui-runtime --report $Report
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
} finally {
  Pop-Location
}

Write-Output "target-host UI runtime report: $Report"
