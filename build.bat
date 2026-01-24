@echo off
chcp 65001 >nul

echo 正在构建端口转发工具...

:: 检查Go是否安装
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo 错误: 未找到Go命令，请确保已安装Go并添加到环境变量
    pause
    exit /b 1
)

:: 获取依赖
echo 获取依赖...
go mod tidy
if %errorlevel% neq 0 (
    echo 错误: 获取依赖失败
    pause
    exit /b 1
)

:: 构建项目
echo 构建项目...
go build -o port-forwarder.exe .
if %errorlevel% neq 0 (
    echo 错误: 构建失败
    pause
    exit /b 1
)

echo 构建成功！
echo 可执行文件port-forwarder.exe

pause