@echo off
chcp 65001 >nul 2>nul
setlocal EnableDelayedExpansion

echo ========================================
echo   Lark Robot - Build Script
echo ========================================
echo.

set APP_NAME=lark-robot
set BUILD_DIR=build

where go >nul 2>&1
if !errorlevel! neq 0 (
    echo [ERROR] Go not found
    goto :end
)

where node >nul 2>&1
if !errorlevel! neq 0 (
    echo [ERROR] Node.js not found
    goto :end
)

echo [1/5] Clean...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist static\dist rmdir /s /q static\dist

echo [2/5] Build frontend...
cd web
call npm install --silent
if !errorlevel! neq 0 (
    echo [ERROR] npm install failed
    cd ..
    goto :end
)
call npm run build
if !errorlevel! neq 0 (
    echo [ERROR] frontend build failed
    cd ..
    goto :end
)
cd ..

echo [3/5] Embed frontend...
xcopy /E /I /Y /Q web\dist static\dist >nul

echo [4/5] Compile Go binary...

echo        - linux/amd64
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\linux-amd64\%APP_NAME% .

echo        - linux/arm64
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\linux-arm64\%APP_NAME% .

echo        - windows/amd64
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\windows-amd64\%APP_NAME%.exe .

echo        - darwin/amd64
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\darwin-amd64\%APP_NAME% .

echo        - darwin/arm64
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\darwin-arm64\%APP_NAME% .

echo [5/5] Copy config...
copy config.yaml.example %BUILD_DIR%\linux-amd64\config.yaml.example >nul
copy config.yaml.example %BUILD_DIR%\linux-arm64\config.yaml.example >nul
copy config.yaml.example %BUILD_DIR%\windows-amd64\config.yaml.example >nul
copy config.yaml.example %BUILD_DIR%\darwin-amd64\config.yaml.example >nul
copy config.yaml.example %BUILD_DIR%\darwin-arm64\config.yaml.example >nul

echo.
echo ========================================
echo   Build OK!  Output: %BUILD_DIR%\
echo ========================================
echo.
echo   build\linux-amd64\       Linux x86_64
echo   build\linux-arm64\       Linux ARM64
echo   build\windows-amd64\     Windows x86_64
echo   build\darwin-amd64\      macOS x86_64
echo   build\darwin-arm64\      macOS ARM64 (Apple Silicon)
echo.

:end
endlocal
pause
