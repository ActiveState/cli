# Test runs for install.ps1 scripts

---

## NOTE

In order to run these tests you must first uninstall the state tool.
To do this run:

```powershell
Remove-Item (Get-Command state.exe).Source
```

If you installed the state tool with a different binary name replace 'state.exe' with it.

When the tests are complete run the install script with your preferred configuration
options to reinstall the state tool.

---

This is a list of tests to manually run after modifying the `install.ps1` file.

## As administrator

First save your path

```powershell
$oldpath = $env:Path
```

### Install as administrator

#### Version A.1: Do not update path

```powershell
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

You should be asked if you want to modify the system path, enter `n`.

**What to look for**:

- All messages should be accurate and make sense.
- You should see instructions how to update the PATH in order to use the state tool.
- You should see a warning about running as an administrator

#### Version A.2: Update path

```powershell
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

You should be asked if you want to modify the system path, enter `y`.  

**What to look for**:

- All messages should be accurate and make sense.
- You should see a warning about running as administrator.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::Machine)
  ```

  to ensure that the path is added to the system path.
- Ensure that the path is also `$env:Path`.

#### Version A.3: Path is set already

```powershell
$env:Path += ";C:\temp\state\bin"
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

**What to look for**:

- All messages should be accurate and make sense.
- You should receive a warning about running as administrator.
- You should not be asked to set up the path.

#### Cleanup A.1

```powershell
Remove-Item -Recurse -Force C:\temp\state\bin

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::Machine)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::Machine)
```

### Install and activate as administrator

#### Version A.4 Invalid options

```powershell
.\public\install.ps1 -Activate ActiveState/cli -n
```

***What to look for**:

- An error message about incompatible options

#### Version A.5 Install

```powershell
.\public\install.ps1 -Activate ActiveState/cli
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

When prompted for a user name and password, enter your credentials.

When prompted for the installation directory, type `C:\temp\state`

**What to look for**:

- You should not be prompted whether you want to add the PATH to the system path.
- All messages should be accurate and make sense.
- You should end up in an activated environment.  Ensure that the `state` tool is in the PATH.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::Machine)
  ```

  to ensure that the path is added to the system path.

#### Cleanup A.2

Ensure that you exit out of your activated session.

```powershell
Remove-Item -Recurse -Force C:\temp\state

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::Machine)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::Machine)
```

#### Version A.6 Install twice

```powershell
.\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

Run command again

```powershell
.\public\install.ps1
```

**What to look for**:

- You should be prompted to update your PATH
- You should see a warning for running as administrator
- After the second install attempt you should be presented with a message that says:

```powershell
Previous install detected at '<install-dir>'
If you would like to reinstall the state tool please first uninstall it.
You can do this by running 'Remove-Item' <install-dir>'
```

### Cleanup A.6

```powershell
Remove-Item -Recurse -Force C:\temp\state

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::Machine)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::Machine)
```

## As User

First save your path

```powershell
$oldpath = $env:Path
```

### Install as user

#### Version U.1: Do not update path

```powershell
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

You should be asked if you want to modify the system path, enter `n`.

**What to look for**:

- All messages should be accurate and make sense.
- You should receive instructions how to update the PATH in order to use the state tool.

#### Version U.2: Update path

```powershell
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

You should be asked if you want to modify the system path, enter `y`.  

**What to look for**:

- All messages should be accurate and make sense.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::User)
  ```

  to ensure that the path is added to the system path.

#### Version U.3: Path is set already

```powershell
$env:Path += ";C:\temp\state\bin"
powershell .\public\install.ps1
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

**What to look for**:

- All messages should be accurate and make sense.
- You should not be asked to set up the path.

#### Cleanup U.1

```powershell
Remove-Item -Recurse -Force C:\temp\state\bin

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::User)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::User)
```

### Install and activate as user

#### Version U.4 Install

```powershell
.\public\install.ps1 -Activate ActiveState/cli
```

When prompted for installation directory, respond with temporary directory `C:\temp\state\bin`.

When prompted for a user name and password, enter your credentials.

When prompted for the installation directory, type `C:\temp\state`

**What to look for**:

- You should not be prompted whether you want to add the PATH to the system path.
- All messages should be accurate and make sense.
- You should end up in an activated environment.  Ensure that the `state` tool is in the PATH.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::User)
  ```

  to ensure that the path is added to the system path.

#### Cleanup U.2

Ensure that you exit out of your activated session.

```powershell
Remove-Item -Recurse -Force C:\temp\state

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::User)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::User)
```
