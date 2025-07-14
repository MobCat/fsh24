@echo off
:: FSH24 Installation Script
:: This script must be run as Administrator
set "SCRIPT_DIR=%~dp0"
echo Installing Fast Sample Hash 24-bit...
echo.

:: Check if running as administrator
net session >nul 2>&1
if %errorLevel% NEQ 0 (
    echo This script must be run as Administrator!
    echo Right-click on the batch file and select "Run as administrator"
    pause
    exit /b 1
)

:: Check if fsh24.exe exists in current directory
if not exist "%SCRIPT_DIR%fsh24.exe" (
    echo Error: fsh24.exe not found in current directory!
    echo Please place fsh24.exe in the same folder as this script.
    pause
    exit /b 1
)

:: User choice for verbose or friendly mode
:CHOICE
echo.
echo Select installation mode:
echo These modes affect how FSH24 will run and display info.
echo   [1] Friendly Mode (Displays essential info)
echo   [2] Verbose Mode  (Displays detailed info and benchmark times)
set /p "MODE_CHOICE=Enter your choice (1 or 2): "

if /i "%MODE_CHOICE%"=="1" (
    set "FSH24_COMMAND_FILE=fsh24.exe \"%%1\""
    set "FSH24_COMMAND_DIR=fsh24.exe"
    set "MODE_DESCRIPTION=Friendly Mode"
) else if /i "%MODE_CHOICE%"=="2" (
    set "FSH24_COMMAND_FILE=fsh24.exe -v \"%%1\""
    set "FSH24_COMMAND_DIR=fsh24.exe -v"
    set "MODE_DESCRIPTION=Verbose Mode"
) else (
    echo Invalid choice. Please enter 1 or 2.
    goto :CHOICE
)

echo.
echo Proceeding with %MODE_DESCRIPTION% installation...
echo.

echo [1/4] Copying fsh24.exe to System32...
copy "%SCRIPT_DIR%fsh24.exe" "%SystemRoot%\System32\" >nul
if %errorLevel% NEQ 0 (
    echo Error: Failed to copy fsh24.exe to System32
    pause
    exit /b 1
)
echo Done.

echo [2/4] Setting up context menu for files...
:: Add context menu for individual files, excluding .fsh24
reg add "HKEY_CLASSES_ROOT\*\shell\ChecksumFSH24" /ve /d "Checksum with FSH24" /f >nul
reg add "HKEY_CLASSES_ROOT\*\shell\ChecksumFSH24" /v "Icon" /d "%SystemRoot%\System32\fsh24.exe" /f >nul
:: Add the AppliesTo rule to exclude .fsh24 files
reg add "HKEY_CLASSES_ROOT\*\shell\ChecksumFSH24" /v "AppliesTo" /d "NOT System.FileExtension:=\"fsh24\"" /f >nul
reg add "HKEY_CLASSES_ROOT\*\shell\ChecksumFSH24\command" /ve /d "%FSH24_COMMAND_FILE%" /f >nul

:: Add context menu for multiple file selection (for directory background) - no change needed here
reg add "HKEY_CLASSES_ROOT\Directory\Background\shell\ChecksumFSH24" /ve /d "Checksum with FSH24" /f >nul
reg add "HKEY_CLASSES_ROOT\Directory\Background\shell\ChecksumFSH24" /v "Icon" /d "%SystemRoot%\System32\fsh24.exe" /f >nul
reg add "HKEY_CLASSES_ROOT\Directory\Background\shell\ChecksumFSH24\command" /ve /d "%FSH24_COMMAND_DIR%" /f >nul

echo Done.

echo Done.

echo [3/4] Registering .fsh24 file type...
:: Register the .fsh24 file extension
reg add "HKEY_CLASSES_ROOT\.fsh24" /ve /d "FSH24File" /f >nul
reg add "HKEY_CLASSES_ROOT\.fsh24" /v "Content Type" /d "application/x-fsh24" /f >nul

:: Create the file type description
reg add "HKEY_CLASSES_ROOT\FSH24File" /ve /d "FSH24 Checksum File" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\DefaultIcon" /ve /d "%SystemRoot%\System32\fsh24.exe" /f >nul

echo Done.

echo [4/4] Setting up .fsh24 file actions...
:: Set up "Run checksum" as default action
reg add "HKEY_CLASSES_ROOT\FSH24File\shell" /ve /d "runchecksum" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\runchecksum" /ve /d "Run FSH24 Checksum Here" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\runchecksum" /v "Icon" /d "%SystemRoot%\System32\fsh24.exe" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\runchecksum\command" /ve /d "%FSH24_COMMAND_FILE%" /f >nul

:: Set up "Edit" action
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\edit" /ve /d "Edit FSH24" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\edit" /v "Icon" /d "%SystemRoot%\System32\fsh24.exe" /f >nul
reg add "HKEY_CLASSES_ROOT\FSH24File\shell\edit\command" /ve /d "notepad++.exe \"%%1\"" /f >nul

echo Done.

echo.
echo Fast Sample Hash 24-bit was "Installed" successfully! \^^__^^/
echo.
echo Features installed:
echo - Copy FSH to system
echo - Right-click context menu "Checksum with FSH24" for files
echo - .fsh24 file type registered to FSH24
echo - .fsh24 files can be opened with "Run checksum" or "Edit"
echo.
echo You may need to refresh your desktop or restart Explorer to see changes.
pause