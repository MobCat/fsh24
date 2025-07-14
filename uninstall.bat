@echo off
:: FSH24 Uninstall Script
:: This script must be run as Administrator

echo Uninstalling Fast Sample Hash 24-bit...
echo.

:: Check if running as administrator
net session >nul 2>&1
if %errorLevel% NEQ 0 (
    echo This script must be run as Administrator!
    echo Right-click on the batch file and select "Run as administrator"
    pause
    exit /b 1
)

echo [1/4] Removing fsh24.exe from System32...
if exist "%SystemRoot%\System32\fsh24.exe" (
    del "%SystemRoot%\System32\fsh24.exe" >nul 2>&1
    if %errorLevel% EQU 0 (
        echo Done.
    ) else (
        echo Warning: Could not remove fsh24.exe from System32
    )
) else (
    echo fsh24.exe not found in System32 - skipping.
)

echo [2/4] Removing context menu entries...
:: Remove context menu for individual files
reg delete "HKEY_CLASSES_ROOT\*\shell\ChecksumFSH24" /f >nul 2>&1

:: Remove context menu for multiple file selection
reg delete "HKEY_CLASSES_ROOT\Directory\Background\shell\ChecksumFSH24" /f >nul 2>&1

echo Done.

echo [3/4] Removing .fsh24 file type registration...
:: Remove the .fsh24 file extension
reg delete "HKEY_CLASSES_ROOT\.fsh24" /f >nul 2>&1

echo Done.

echo [4/4] Removing .fsh24 file actions...
:: Remove the file type description and actions
reg delete "HKEY_CLASSES_ROOT\FSH24File" /f >nul 2>&1

echo Done.

echo.
echo Fast Sample Hash 24-bit was uninstalled successfully :(
echo.
echo Features removed:
echo - Removed FSH from system
echo - Removed right-click context menu "Checksum with FSH24"
echo - Removed .fsh24 file type registration
echo - Removed .fsh24 file actions
echo.
echo You may need to refresh your desktop or restart Explorer to see changes.
pause