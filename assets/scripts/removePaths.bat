REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. 
REM In the case the State Tool, the binary cannot be removed while it is running.
REM In the case of the config directory this is the log file, which 
REM remains open and inaccessable while the State Tool is running.
REM The expected arguments are the process ID, the executable name
REM and a list of directories to be removed when the process has completed

set logfile=C:\Users\Mike\Desktop\%random%.txt

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
:wait_for_exit
    if %proc% == %exe% (
        for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
        goto :wait_for_exit
    )

for /d %%i in (%paths%) do (
    if exist %%i\* (
        rmdir /s /q %%i
    ) else (
        del /s /q %%i
    )
)