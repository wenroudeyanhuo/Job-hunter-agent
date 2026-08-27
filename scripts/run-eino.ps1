param(
    [string]$Addr = ":8080",
    [string]$DBPath = "data/job-hunter-agent.db"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$backendPath = Join-Path $repoRoot "backend"

$env:APP_ADDR = $Addr
$env:APP_DB_PATH = $DBPath
$env:AGENT_ORCHESTRATOR = "eino_graph"

Push-Location $backendPath
try {
    go run -tags eino ./cmd/server
}
finally {
    Pop-Location
}
