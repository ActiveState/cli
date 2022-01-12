REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. 
REM In the case the State Tool, the binary cannot be removed while it is running.
REM In the case of the config directory this is the log file, which 
REM remains open and inaccessable while the State Tool is running.
REM The expected arguments are the process ID, the executable name
REM and a list of paths to be removed when the process has completed

set logfile=%1
shift
set pid=%1
shift
set exe=%1
shift

set paths=
:set_paths
    if not "%1"=="" (
        set paths=%paths% %1
        shift
        goto set_paths
    )

for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
echo "Waiting for process %exe% with PID %pid% to end..." >> %logfile%
:wait_for_exit
    if %proc% == %exe% (
        for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
        goto :wait_for_exit
    )

echo "Process %exe% has ended" >> %logfile%
for /d %%i in (%paths%) do (
    echo "Attempting to remove path %%i" >> %logfile%
    if exist %%i\* (
        rmdir /s /q %%i >> %logfile%
    ) else (
        del /q %%i >> %logfile%
    )
    if %ERRORLEVEL% NEQ 0 (
        echo "Could not remove directory: %%i" >> %logfile%
    ) else (
        echo "Successfully removed path %%i" >> %logfile%
    )
)

echo "Successfully removed State Tool installation and related files." >> %logfile%