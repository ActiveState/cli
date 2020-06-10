[CmdletBinding()]
param (
    [Parameter(Mandatory = $True)][string]$Path
    , [Parameter(Mandatory = $False)]
    [string]
    $User = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
)

$writeRights = @(
    [System.Security.AccessControl.FileSystemRights]::AppendData
    [System.Security.AccessControl.FileSystemRights]::CreateFiles
    [System.Security.AccessControl.FileSystemRights]::CreateDirectories
    [System.Security.AccessControl.FileSystemRights]::FullControl
    [System.Security.AccessControl.FileSystemRights]::Modify
    [System.Security.AccessControl.FileSystemRights]::TakeOwnership
    [System.Security.AccessControl.FileSystemRights]::Write
    [System.Security.AccessControl.FileSystemRights]::WriteData
)

# TODO: For testing
$User = "DESKTOP-03VON80\TestUser"

$acl = (Get-Acl $Path)

$userSID = (New-Object System.Security.Principal.NTAccount($User)).Translate([System.Security.Principal.SecurityIdentifier])

$accessRules = $acl.GetAccessRules($True, $True, [System.Security.Principal.NTAccount])

foreach ($item in $accessRules) {
    # Write-Host $item.IdentityReference
    $aclSID = New-Object System.Security.Principal.SecurityIdentifier((New-Object System.Security.Principal.NTAccount($item.IdentityReference.ToString())).Translate([System.Security.Principal.SecurityIdentifier]))
    
    # Check user permissions
    if ($aclSID.Equals($userSID)) {
        Write-Host "Match!"
        Write-Host $item.FileSystemRights

        if ($writeRights -contains $item.FileSystemRights) {
            Write-Host "Another match!"
            exit 0
        }
    }

    # Check group permissions
    $groups = (Get-LocalGroup).Name

    $userGroups = [System.Collections.ArrayList]@()
    foreach ($group in $groups) {
        $members = (Get-LocalGroupMember -Group $group).Name
        if ($members -contains $User) {
            $userGroups.Add($group) > $null
        }
    }

    foreach ($group in $userGroups) {
        $groupSID = (New-Object System.Security.Principal.NTAccount($group)).Translate([System.Security.Principal.SecurityIdentifier]).value

        # $val = $item.FileSystemRights | Get-Member
        # Write-Host $val
        if ($aclSID.ToString() -eq $groupSID.ToString()) {
            # TODO: The FileSystemRights value can be a comma-separated
            # list of enums. Need to break up and compare with above
            if ($writeRights -contains $item.FileSystemRights) {
                Write-Host "Group match in group: "
                Write-Host $group
            }
        }
    }
}

