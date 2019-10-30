# Test runs for install.ps1 scripts

This is a list of tests to manually run after modifying the `install.ps1` file.

## As administrator

First save your path

```powershell
$oldpath = $env:Path
```

### Install as administrator

#### Version A.1: Update path

```powershell
powershell .\public\install.ps1 -t C:\temp\state\bin
```

When prompted to continue after being presented with installation information select `y`.

**What to look for**:

- All messages should be accurate and make sense.
- You should see a warning about running as administrator.
- You should be presented with a message that says: `Please restart your command prompt in order to start using the 'state.exe' program`
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::Machine)
  ```

  to ensure that the path is added to the system path.

#### Cleanup A.1

```powershell
Remove-Item -Recurse -Force C:\temp\state\bin

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::Machine)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::Machine)
```

#### Version A.2: Path is set already

```powershell
$env:Path += ";C:\temp\state\bin"
powershell .\public\install.ps1 -t C:\temp\state\bin
```

When prompted to continue after being presented with installation information select `y`.

**What to look for**:

- All messages should be accurate and make sense.
- You should receive a warning about running as administrator.
- You should be presented with a message that says: `You may now start using the 'state.exe' program`

#### Cleanup A.2

```powershell
$env:Path = $oldpath
Remove-Item -Recurse -Force C:\temp\state\bin
```

### Install and activate as administrator

#### Version A.3 Invalid options

```powershell
.\public\install.ps1 -Activate ActiveState/cli -n
```

**What to look for**:

- An error message about incompatible options

#### Version A.4 Install and Activate

```powershell
.\public\install.ps1 -t C:\temp\state\bin -Activate ActiveState/cli
```

When prompted for the activation directory, type `C:\temp\state`

**What to look for**:

- All messages should be accurate and make sense.
- You should end up in an activated environment.  Ensure that the `state` tool is in the PATH.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::Machine)
  ```

  to ensure that the path is added to the system path.

#### Cleanup A.4

Ensure that you exit out of your activated session.

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

#### Version U.1: Update path

```powershell
powershell .\public\install.ps1 -t C:\temp\state\bin
```

When prompted to continue after being presented with installation information select `y`.

**What to look for**:

- All messages should be accurate and make sense.
- You should be presented with a message that says: `Please restart your command prompt in order to start using the 'state.exe' program`
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::User)
  ```

  to ensure that the path is added to the system path.

#### Cleanup U.1

```powershell
Remove-Item -Recurse -Force C:\temp\state\bin

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::User)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::User)
```

#### Version U.2: Path is set already

```powershell
$env:Path += ";C:\temp\state\bin"
powershell .\public\install.ps1 -t C:\temp\state\bin
```

**What to look for**:

- All messages should be accurate and make sense.
- You should be presented with a message that says: `You may now start using the 'state.exe' program`

### Cleanup U.2

```powershell
$env:Path = $oldpath
Remove-Item -Recurse -Force C:\temp\state\bin
```

### Install and activate as user

#### Version U.3 Install and Activate

```powershell
.\public\install.ps1 -t C:\temp\state\bin -Activate ActiveState/cli
```

When prompted for the activation directory, type `C:\temp\state`

**What to look for**:

- All messages should be accurate and make sense.
- You should end up in an activated environment.  Ensure that the `state` tool is in the PATH.
- Run

  ```powershell
  [Environment]::GetEnvironmentVariable('Path', [EnvironmentVariableTarget]::User)
  ```

  to ensure that the path is added to the system path.

#### Cleanup U.3

Ensure that you exit out of your activated session.

```powershell
Remove-Item -Recurse -Force C:\temp\state

$path = [System.Environment]::GetEnvironmentVariable( 'PATH', [EnvironmentVariableTarget]::User)
$path = ($path.Split(';') | Where-Object { $_ -ne 'C:\temp\state\bin' }) -join ';'
[System.Environment]::SetEnvironmentVariable('PATH', $path, [EnvironmentVariableTarget]::User)
```
