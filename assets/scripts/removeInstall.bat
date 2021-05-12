REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. In this case the State
REM Tool binary cannot be removed while it is running.
REM The expected argument is the path to the State Tool binary

timeout 3
del /s /q %1\state.exe
exit 0
