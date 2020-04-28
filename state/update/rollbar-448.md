## darwin

### Unable to self update: rename ... no such file or directory (x6)

Unable to self update: rename /Users/administrator/Library/Application Support/KomodoIDE/12.0/XRE/state/.state.new /Users/administrator/Library/Application Support/KomodoIDE/12.0/XRE/state/state: no such file or directory

Most recent commit: https://github.com/ActiveState/cli/commit/10194c2 (20200423T1614-0700)

## linux

### Unable to self update: open /usr/sbin/.state.new: permission denied (x4)

Unable to self update: open /usr/sbin/.state.new: permission denied Stacktrace: /home/circleci/workdir/github.com/ActiveState/cli/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 /home/circleci/workdir/github.com/ActiveState/cli/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:121 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.autoUpdate:280 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.run:142 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.main:56 /usr/local/go/src/runtime/proc.go:runtime.main:203 /usr/local/go/src/runtime/asm_amd64.s:runtime.goexit:1357

### Unable to self update: mkdir /usr/local/bin/update: permission denied (x1)

Unable to self update: mkdir /usr/local/bin/update: permission denied Stacktrace: /home/circleci/workdir/github.com/ActiveState/cli/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 /home/circleci/workdir/github.com/ActiveState/cli/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:121 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.autoUpdate:280 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.run:142 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.main:56 /usr/local/go/src/runtime/proc.go:runtime.main:203 /usr/local/go/src/runtime/asm_amd64.s:runtime.goexit:1357

### Unable to self update: read tcp ...: read: connection reset by peer (x1)

Unable to self update: read tcp 192.168.1.5:34226->52.95.146.124:443: read: connection reset by peer Stacktrace: /home/circleci/workdir/github.com/ActiveState/cli/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 /home/circleci/workdir/github.com/ActiveState/cli/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:121 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.autoUpdate:288 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.run:143 /home/circleci/workdir/github.com/ActiveState/cli/cmd/state/main.go:main.main:57 /usr/local/go/src/runtime/proc.go:runtime.main:203 /usr/local/go/src/runtime/asm_amd64.s:runtime.goexit:1357``

## windows

### Unable to automatically check for updates: rename .state.exe.new state.exe: used by another process. (x4)

Unable to automatically check for updates: rename {homedir}\AppData\Roaming\ActiveState\bin\.state.exe.new {homedir}\AppData\Roaming\ActiveState\bin\state.exe: The process cannot access the file because it is being used by another process.

### Unable to self update: rename .state.new state.exe: used by another process. (x17)

Unable to self update: rename {homedir}\AppData\Roaming\ActiveState\bin\.state.new {homedir}\AppData\Roaming\ActiveState\bin\state.exe: The process cannot access the file because it is being used by another process. Stacktrace: d:/a/1/s/internal/osutils/stacktrace/stacktrace.go:github.com/ActiveState/cli/internal/osutils/stacktrace.Get:45 d:/a/1/s/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 d:/a/1/s/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:121 d:/a/1/s/cmd/state/main.go:main.autoUpdate:280 d:/a/1/s/cmd/state/main.go:main.run:142 d:/a/1/s/cmd/state/main.go:main.main:56 C:/Go1.14.1/src/runtime/proc.go:runtime.main:203 C:/Go1.14.1/src/runtime/asm_amd64.s:runtime.goexit:1373

### Unable to automatically check for updates: rename state.exe .state.exe.old: used by another process. (x30)

Unable to automatically check for updates: rename {homedir}\AppData\Roaming\ActiveState\bin\state.exe {homedir}\AppData\Roaming\ActiveState\bin\.state.exe.old: The process cannot access the file because it is being used by another process.

### Unable to self update: rename state.exe .state.old: used by another process. (x90)

Unable to self update: rename {homedir}\AppData\Roaming\ActiveState\bin\state.exe {homedir}\AppData\Roaming\ActiveState\bin\.state.old: The process cannot access the file because it is being used by another process.

### Unable to self update: rename state.exe .state.old: cannot find the file specified. (x18)

Unable to self update: rename {homedir}\AppData\Roaming\ActiveState\bin\state.exe {homedir}\AppData\Roaming\ActiveState\bin\.state.old: The system cannot find the file specified. Stacktrace: d:/a/1/s/internal/osutils/stacktrace/stacktrace.go:github.com/ActiveState/cli/internal/osutils/stacktrace.Get:45 d:/a/1/s/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 d:/a/1/s/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:121 d:/a/1/s/cmd/state/main.go:main.autoUpdate:280 d:/a/1/s/cmd/state/main.go:main.run:142 d:/a/1/s/cmd/state/main.go:main.main:56 C:/Go1.14.1/src/runtime/proc.go:runtime.main:203 C:/Go1.14.1/src/runtime/asm_amd64.s:runtime.goexit:1373

### Unable to automatically check for updates: Couldn't get url=...: no such host (x1)

Unable to automatically check for updates: Couldn't get url=https://s3.ca-central-1.amazonaws.com/cli-update/update/state/unstable/windows-amd64.json: Get "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/unstable/windows-amd64.json": dial tcp: lookup s3.ca-central-1.amazonaws.com: no such host Stacktrace: d:/a/1/s/internal/osutils/stacktrace/stacktrace.go:github.com/ActiveState/cli/internal/osutils/stacktrace.Get:45 d:/a/1/s/internal/logging/logging.go:github.com/ActiveState/cli/internal/logging.Error:250 d:/a/1/s/internal/updater/check.go:github.com/ActiveState/cli/internal/updater.TimedCheck:107 d:/a/1/s/cmd/state/main.go:main.autoUpdate:288 d:/a/1/s/cmd/state/main.go:main.run:143 d:/a/1/s/cmd/state/main.go:main.main:57 C:/Go1.14.2/src/runtime/proc.go:runtime.main:203 C:/Go1.14.2/src/runtime/asm_amd64.s:runtime.goexit:1373
