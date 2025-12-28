@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   Noein Video Editor Launcher
echo ========================================
echo.

:: Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

:: Check if noein.exe exists
if not exist "build\bin\noein.exe" (
    echo [INFO] noein.exe not found - building project...
    echo.
    call :BuildProject
    if %errorlevel% neq 0 (
        echo.
        echo [ERROR] Build failed!
        pause
        exit /b 1
    )
    echo.
    echo [OK] Build completed successfully!
    echo.
)

:: Check for FFmpeg
set "FFMPEG_FOUND=0"

:: Check bundled FFmpeg first
if exist "build\bin\ffmpeg.exe" (
    echo [OK] FFmpeg found ^(bundled^)
    set "FFMPEG_FOUND=1"
    goto :LaunchApp
)

:: Check system PATH
where ffmpeg >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] FFmpeg found (system PATH)
    set "FFMPEG_FOUND=1"
    goto :LaunchApp
)

:: FFmpeg not found - offer options
echo [WARNING] FFmpeg not found!
echo.
echo FFmpeg is required for video processing.
echo Options:
echo   1. Download FFmpeg automatically (recommended)
echo   2. Install FFmpeg manually from https://ffmpeg.org
echo   3. Continue without FFmpeg (app will not work)
echo.
choice /c 123 /n /m "Choose option (1/2/3): "

if %errorlevel% equ 1 (
    call :DownloadFFmpeg
    if %errorlevel% neq 0 (
        echo Failed to download FFmpeg
        pause
        exit /b 1
    )
)
if %errorlevel% equ 2 (
    echo.
    echo Please install FFmpeg and run this launcher again.
    pause
    exit /b 1
)

:LaunchApp

:: Launch the application
echo.
echo [LAUNCHING] Starting Noein Video Editor...
echo.
start "" "build\bin\noein.exe"

:: Wait a moment to see if it crashes immediately
timeout /t 2 /nobreak >nul

echo.
echo Application launched successfully!
echo You can close this window.
echo.
pause
exit /b 0

:BuildProject
echo ========================================
echo   Building Noein...
echo ========================================
echo.
wails build
if %errorlevel% neq 0 (
    exit /b 1
)
exit /b 0

:DownloadFFmpeg
echo.
echo ========================================
echo   Downloading FFmpeg...
echo ========================================
echo.

:: Create temp directory
set "TEMP_DIR=%TEMP%\noein_ffmpeg"
if not exist "%TEMP_DIR%" mkdir "%TEMP_DIR%"

:: Download FFmpeg using PowerShell
echo Downloading FFmpeg (this may take a few minutes)...
powershell -Command "& {[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri 'https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip' -OutFile '%TEMP_DIR%\ffmpeg.zip'}"

if %errorlevel% neq 0 (
    echo [ERROR] Failed to download FFmpeg
    echo Please download manually from: https://ffmpeg.org
    rd /s /q "%TEMP_DIR%" 2>nul
    exit /b 1
)

echo Extracting FFmpeg...
powershell -Command "Expand-Archive -Path '%TEMP_DIR%\ffmpeg.zip' -DestinationPath '%TEMP_DIR%' -Force"

if %errorlevel% neq 0 (
    echo [ERROR] Failed to extract FFmpeg
    rd /s /q "%TEMP_DIR%" 2>nul
    exit /b 1
)

:: Find and copy ffmpeg.exe and ffprobe.exe
set "COPY_SUCCESS=0"
for /d %%i in ("%TEMP_DIR%\ffmpeg-*") do (
    if exist "%%i\bin\ffmpeg.exe" (
        copy "%%i\bin\ffmpeg.exe" "build\bin\ffmpeg.exe" >nul
        copy "%%i\bin\ffprobe.exe" "build\bin\ffprobe.exe" >nul
        echo [OK] FFmpeg installed successfully!
        set "COPY_SUCCESS=1"
    )
)

:: Cleanup
rd /s /q "%TEMP_DIR%" 2>nul

if %COPY_SUCCESS% equ 0 (
    echo [ERROR] Failed to install FFmpeg
    exit /b 1
)

exit /b 0
