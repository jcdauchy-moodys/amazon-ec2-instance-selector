@echo off
setlocal enabledelayedexpansion

:: Set default values
set "BINARY_NAME=ec2-instance-selector"
set "VERSION=dev"
set "BUILD_DATE="
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set "BUILD_DATE=%%I"
set "BUILD_DATE=!BUILD_DATE:~0,8!"

:: Parse command line arguments
:parse_args
if "%~1"=="" goto end_parse_args
if /i "%~1"=="--release" (
    set "VERSION="
    for /f "tokens=*" %%i in ('git describe --tags') do set "VERSION=%%i"
    if "!VERSION!"=="" set "VERSION=unknown"
)
shift
goto parse_args
:end_parse_args

:: Set build flags - note the quotes around the entire -ldflags value
set "LDFLAGS=-X github.com/aws/amazon-ec2-instance-selector/v3/pkg/version.Version=!VERSION! -X github.com/aws/amazon-ec2-instance-selector/v3/pkg/version.BuildDate=!BUILD_DATE!"

:: Create build directory if it doesn't exist
if not exist "build" mkdir "build"

echo Building %BINARY_NAME% version !VERSION!...
echo Build date: !BUILD_DATE!

:: Build the API server
echo Building API server...
go build -ldflags "!LDFLAGS!" -o build\api-server.exe .\cmd\api-server

:: Build the CLI
echo Building CLI...
go build -ldflags "!LDFLAGS!" -o build\%BINARY_NAME%.exe .\cmd\main.go

if %ERRORLEVEL% neq 0 (
    echo Build failed
    exit /b %ERRORLEVEL%
)

echo Build completed successfully
echo Binaries are in the build directory:
echo   build\api-server.exe
echo   build\%BINARY_NAME%.exe

endlocal 