$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$url = "http://localhost/"
$useDevProfile = $false
$showLogs = $false

foreach ($arg in $args) {
    switch ($arg) {
        "--dev" { $useDevProfile = $true }
        "-Dev" { $useDevProfile = $true }
        "--logs" { $showLogs = $true }
        "-Logs" { $showLogs = $true }
        default {
            Write-Error "Unknown option '$arg'. Supported options: --dev, --logs"
        }
    }
}

Set-Location $repoRoot

Write-Host "Starting Roller_hoops with Docker Compose..."

$composeArgs = @("compose")
if ($useDevProfile) {
    $composeArgs += @("--profile", "dev")
}
$composeArgs += @("up", "--build", "-d")

& docker @composeArgs

if ($LASTEXITCODE -ne 0) {
    throw "Docker Compose failed. Confirm Docker Desktop is running and port 80 is available."
}

Write-Host "Waiting for the web UI health check..."

$deadline = (Get-Date).AddMinutes(3)
while ((Get-Date) -lt $deadline) {
    try {
        $response = Invoke-WebRequest -Uri "$($url)healthz" -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
            Write-Host "Opening $url"
            Start-Process $url
            if ($showLogs) {
                & docker compose logs -f --tail=200
            }
            exit 0
        }
    }
    catch {
        Start-Sleep -Seconds 3
    }
}

Write-Warning "The stack started, but the UI did not become healthy within 3 minutes."
Write-Warning "Open $url manually or run: docker compose logs -f --tail=200"
exit 1
