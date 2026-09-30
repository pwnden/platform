$ErrorActionPreference = 'Stop'
$checkoutRoot = $PSScriptRoot
$distPath = Join-Path $checkoutRoot 'dist'
if ((Test-Path -LiteralPath $distPath) -and
    ((Get-Item -LiteralPath $distPath).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    throw 'Generated paths must stay inside this checkout.'
}
$architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
$target = switch ($architecture) {
    'x64' { 'windows/amd64' }
    'arm64' { 'windows/arm64' }
    default { throw 'Unsupported host architecture.' }
}
$entryPath = Join-Path $distPath ('.entry.' + [Guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($entryPath) | Out-Null
$exitCode = 1
try {
    & docker buildx build --file (Join-Path $checkoutRoot 'Dockerfile.bootstrap') --target entry `
        --build-arg "PWNDEN_TARGET=$target" --output (Join-Path $entryPath 'payload') $checkoutRoot
    if ($LASTEXITCODE -ne 0) { throw 'Failed to build the checkout entry tool.' }
    & (Join-Path $entryPath 'payload/pwnden-entry.exe') $checkoutRoot @args
    $exitCode = $LASTEXITCODE
} finally {
    $absoluteEntry = [IO.Path]::GetFullPath($entryPath)
    $absoluteDist = [IO.Path]::GetFullPath($distPath) + [IO.Path]::DirectorySeparatorChar
    if (-not $absoluteEntry.StartsWith($absoluteDist, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Entry cleanup target must stay inside the checkout.'
    }
    Remove-Item -LiteralPath $absoluteEntry -Recurse -Force
}
exit $exitCode
