@echo off
:: Setup and first build of the go script.
if not exist "go.mod" (
	go mod init fsh24
)
if not exist "go.sum" (
	go mod tidy
)
setlocal

rem Define the base output name for your executable (without extension)
set OUTPUT_BASE_NAME=fsh24
set LDFLAGS="-s"
set GO_SOURCE_FILE=main.go

:: Build project wihtout debug symbols.
::go build -ldflags "-s"
echo === Building for Windows (AMD64) ===
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags %LDFLAGS% -o %OUTPUT_BASE_NAME%.exe %GO_SOURCE_FILE%
echo Done.


echo === Building for Linux (AMD64) ===
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags %LDFLAGS% -o %OUTPUT_BASE_NAME%-linux-amd64 %GO_SOURCE_FILE%
echo Done.

echo === Building for macOS (ARM64/Apple Silicon) ===
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags %LDFLAGS% -o %OUTPUT_BASE_NAME%-mac-arm64 %GO_SOURCE%
echo Done.

echo === Building for Raspberry Pi (ARM64) ===
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags %LDFLAGS% -o %OUTPUT_BASE_NAME%_Pi3_arm64 %GO_SOURCE_FILE%
echo Done.
echo.

endlocal
:::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::

:: Add icon with resource hacker
echo === Adding Icon to windows exe ===
setlocal
:: --- Configuration ---
set "RH_PATH=C:\Program Files (x86)\Resource Hacker\ResourceHacker.exe"
set "ICON_FILE=icon.ico"
set "INPUT_EXE=fsh24.exe"
set "ICON_FILE=icon.ico" 
set "BITMAP_FILE=icon.bmp"

set "TEMP_RH_SCRIPT=script.txt"

:: --- Generate the .rc script on the fly ---
:: Using `>` for the first line to create/overwrite, `>>` for subsequent lines to append.
echo [FILENAMES] > "%TEMP_RH_SCRIPT%"
echo Exe=    "%INPUT_EXE%" >> "%TEMP_RH_SCRIPT%"
echo SaveAs= "%INPUT_EXE%" >> "%TEMP_RH_SCRIPT%"
echo.                                >> "%TEMP_RH_SCRIPT%"
echo [COMMANDS] >> "%TEMP_RH_SCRIPT%"
echo -addoverwrite "%BITMAP_FILE%", BITMAP,128, >> "%TEMP_RH_SCRIPT%"
echo -addoverwrite "%BITMAP_FILE%", BITMAP,129,0 >> "%TEMP_RH_SCRIPT%"
echo -addoverwrite "%ICON_FILE%", ICONGROUP,MAINICON,0 >> "%TEMP_RH_SCRIPT%"

:: --- Run Resource Hacker ---
echo Adding icon "%ICON_FILE%" to "%INPUT_EXE%"...

"%RH_PATH%" -script "%TEMP_RH_SCRIPT%"
del "%TEMP_RH_SCRIPT%"
echo Done.
endlocal
:: This will get the exe to under 1mb.
:: This has to be done after we add the icon
echo === Packing windows exe ===
upx --best fsh24.exe
echo Done.

