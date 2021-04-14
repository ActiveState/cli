REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. 
REM In this case the State Tool, the binary cannot be removed while it is running.
REM In the case of the config directory this is the log file, which 
REM remains open and inaccessable while the State Tool is running.
REM The expected argument is a directory that will be removed

timeout 2
rmdir /s /q %1
exit 0