REM This script is used by 'state clean' on Windows as we cannot
REM remove files that are currently in use. 
REM In the case the State Tool, the binary cannot be removed while it is running.
REM In the case of the config directory this is the log file, which 
REM remains open and inaccessable while the State Tool is running.
REM The expected arguments is a directory that will be removed are
REM the directory to be removed, the State Tool PID, and the State Tool binary name

@REM @echo off
setlocal ENABLEDELAYEDEXPANSION
@REM timeout 2
set pid=%2
for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
:wait_for_exit
    IF !proc! == %3 (
        for /f %%i in ('tasklist /NH /FI "PID eq %pid%"') do set proc=%%i
        GOTO :wait_for_exit
    )
rmdir /s /q %1
