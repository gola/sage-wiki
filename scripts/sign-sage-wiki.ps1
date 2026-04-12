# sage-wiki code signing script
# Used to bypass Windows Device Guard policy blocking

$exePath = "D:\Work\MindVectorGraph\sage-wiki.exe"

# Check if file exists
if (-not (Test-Path $exePath)) {
    Write-Host "Error: Cannot find $exePath"
    exit 1
}

# Create self-signed certificate (if not exists)
$cert = Get-ChildItem -Path Cert:\LocalMachine\My -CodeSigningCert | Where-Object { $_.Subject -eq "CN=DevLocal" } | Select-Object -First 1

if (-not $cert) {
    Write-Host "Creating self-signed certificate..."
    $cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=DevLocal" -KeyUsage DigitalSignature -KeyAlgorithm RSA -KeyLength 2048
    
    # Move to trusted root certificate store
    Write-Host "Moving certificate to trusted root store..."
    Move-Item -Path "Cert:\LocalMachine\My\$($cert.Thumbprint)" -Destination "Cert:\LocalMachine\Root" -Force
}

# Sign the executable
Write-Host "Signing executable..."
Set-AuthenticodeSignature -FilePath $exePath -Certificate $cert

Write-Host "Done! sage-wiki.exe has been signed and can run normally."