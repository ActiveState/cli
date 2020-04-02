REM This script is used by 'state clean' on Windows.
REM Arguments are expected in the following order:
REM <cache-dir> <config-dir> <install-path>

timeout 2
rmdir /s /q %1
rmdir /s /q %2
del /s /q %3
exit 0