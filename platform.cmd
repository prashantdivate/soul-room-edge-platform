@echo off
setlocal EnableExtensions

set "ROOT_DIR=%~dp0"
set "PLATFORM_DIR=%ROOT_DIR%cloud-infra"
set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=help"
if not "%~1"=="" shift

if /I "%COMMAND%"=="help" goto :help
if /I "%COMMAND%"=="-h" goto :help
if /I "%COMMAND%"=="--help" goto :help

call :require_docker
if errorlevel 1 exit /b %errorlevel%

pushd "%PLATFORM_DIR%" || exit /b 1

if /I "%COMMAND%"=="up" goto :up
if /I "%COMMAND%"=="down" goto :down
if /I "%COMMAND%"=="refresh" goto :refresh
if /I "%COMMAND%"=="restart" goto :restart
if /I "%COMMAND%"=="status" goto :status
if /I "%COMMAND%"=="logs" goto :logs
if /I "%COMMAND%"=="doctor" goto :doctor

echo Unknown command: %COMMAND% 1>&2
popd
call :help
exit /b 2

:up
docker compose up --build -d || goto :failed
docker compose ps || goto :failed
echo Soul Room is available at http://localhost:3080
goto :success

:down
docker compose down || goto :failed
echo Soul Room stopped. Persistent volumes were preserved.
goto :success

:refresh
docker compose up --build --force-recreate --remove-orphans -d || goto :failed
docker compose ps || goto :failed
echo Soul Room was rebuilt and refreshed at http://localhost:3080
goto :success

:restart
docker compose restart || goto :failed
docker compose ps || goto :failed
goto :success

:status
docker compose ps
if errorlevel 1 goto :failed
goto :success

:logs
if not "%~2"=="" (
  echo Usage: platform.cmd logs [service] 1>&2
  popd
  exit /b 2
)
if "%~1"=="" (
  docker compose logs --tail=200 -f
) else (
  docker compose logs --tail=200 -f "%~1"
)
if errorlevel 1 goto :failed
goto :success

:doctor
docker compose version || goto :failed
docker compose config --quiet || goto :failed
echo Docker is ready and cloud-infra/compose.yaml is valid.
goto :success

:success
popd
exit /b 0

:failed
set "RESULT=%errorlevel%"
popd
exit /b %RESULT%

:require_docker
where docker >nul 2>&1
if errorlevel 1 (
  echo Docker is not installed or is not available in PATH. 1>&2
  exit /b 1
)
docker compose version >nul 2>&1
if errorlevel 1 (
  echo Docker Compose v2 is required. Install the "docker compose" plugin. 1>&2
  exit /b 1
)
docker info >nul 2>&1
if errorlevel 1 (
  echo Docker is installed, but the Docker engine is not running or cannot be reached. 1>&2
  exit /b 1
)
exit /b 0

:help
echo Soul Room platform launcher
echo.
echo Usage:
echo   platform.cmd ^<command^> [service]
echo.
echo Commands:
echo   up                 Build and start the complete platform
echo   down               Stop the platform and preserve persistent data
echo   refresh            Rebuild and recreate from the current source
echo   restart            Restart existing containers without rebuilding
echo   status             Show service and health status
echo   logs [service]     Follow all logs, or logs for one service
echo   doctor             Check Docker and validate the Compose configuration
echo   help               Show this help
echo.
echo Examples:
echo   platform.cmd up
echo   platform.cmd refresh
echo   platform.cmd logs platform
exit /b 0
