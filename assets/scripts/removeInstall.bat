REM This script is used by 'state clean' on Windows.
REM The expected argument is the path to the State Tool binary

timeout 2
del /s /q %1
exit 0