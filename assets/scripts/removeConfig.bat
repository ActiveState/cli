REM This script is used by 'state clean' on Windows.
REM The expected argument is a directory that will be removed

timeout 2
rmdir /s /q %1
exit 0