param (
    [Parameter(Mandatory = $True)][string]$Path
)

function IsWritable($path) {
    # Selection of file sytems rights to represent write permission
    # Taken from: https://docs.microsoft.com/en-us/dotnet/api/system.security.accesscontrol.filesystemrights?view=dotnet-plat-ext-3.1
    $permWrite = 0 -bor`
        [System.Security.AccessControl.FileSystemRights]::AppendData -bor `
        [System.Security.AccessControl.FileSystemRights]::CreateFiles -bor `
        [System.Security.AccessControl.FileSystemRights]::CreateDirectories -bor `
        [System.Security.AccessControl.FileSystemRights]::Write -bor `
        [System.Security.AccessControl.FileSystemRights]::WriteData -bor `
        [System.Security.AccessControl.FileSystemRights]::Synchronize

    $User = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name

    $acl = (Get-Acl $Path)

    $userSID = (New-Object System.Security.Principal.NTAccount($User)).Translate([System.Security.Principal.SecurityIdentifier])

    $accessRules = $acl.GetAccessRules($True, $True, [System.Security.Principal.NTAccount])

    foreach ($item in $accessRules) {
        $aclSID = New-Object System.Security.Principal.SecurityIdentifier((New-Object System.Security.Principal.NTAccount($item.IdentityReference.ToString())).Translate([System.Security.Principal.SecurityIdentifier]))
    
        # Check user permissions
        if ($aclSID.Equals($userSID)) {
            Write-Host $item.FileSystemRights

            if (($item.FileSystemRights.value__ -band $permWrite) -eq $permWrite) {
                # Write-Host "Another match!"
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

            if ($aclSID.ToString() -eq $groupSID.ToString()) {
                if (($item.FileSystemRights.value__ -band $permWrite) -eq $permWrite) {
                    exit 0
                }
            }
        }
    }

    exit 1
}

IsWritable $Path
