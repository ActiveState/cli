REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. 
REM In the case the State Tool, the binary cannot be removed while it is running.
REM In the case of the config directory this is the log file, which 
REM remains open and inaccessable while the State Tool is running.
REM The expected arguments are the process ID, the executable name
REM and a list of directories to be removed when the process has completed

set pid=%1
shift
set exe=%1
shift

set dirs=
:set_dirs
    if not "%1"=="" (
        set dirs=%dirs% %1
        shift
        goto set_dirs
    )

for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
:wait_for_exit
    IF %proc% == %exe% (
        for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
        GOTO :wait_for_exit
    )

for /d %%i in (%dirs%) do (
    rmdir /s /q %%i
)
