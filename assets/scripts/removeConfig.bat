REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. In the case of the config
REM directory this is the log file, which remains open and inaccessable
REM while the State Tool is running.
REM The expected argument is a directory that will be removed

timeout 2
rmdir /s /q %1
exit 0