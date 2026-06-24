# Cross-Platform Monitoring Agent 

This is a monitoring agent which is used to collect system metrics, then send to a Central Hub to analyze. Some metrics include:
- CPU: User, System, IOWait
- Memory: Used Memory, Available, Total RAM (Bytes)
- Disk: Used, Free, Total (Bytes), per mountpoint
- Network: Network IOPS (Input/Output Operations Per Second)
- Service: Scan all services config in `config.yaml`

# ⚙️ Build
## Windows
```
go build -o your-agent-name.exe ./agent
```
## Linux
```
go build -o your-agent ./agent
chmod +x your-agent
```
# 🚀 Run as an OS Service

> ⚠️ **Important Note:** All commands interacting with the system service must be executed with the highest administrative privileges (**Run as Administrator** on Windows or using **sudo** on Linux).

The agent comes with a built-in Service Lifecycle Manager. Instead of writing Systemd configuration files or creating Windows Services manually, you can instruct the binary itself to register with the Operating System using the following arguments: `install`, `start`, `stop`, and `uninstall`.

## 🪟 Windows (CMD / PowerShell with Administrator privileges)

Navigate your terminal to the exact directory containing the executable file (e.g., `D:\Go_WalkThrough`) and run the following command sequence:

```cmd
:: 1. Register the Agent into the Windows Service Manager
your-agent-name.exe install

:: 2. Activate and allow the Agent to run in the background permanently
your-agent-name.exe start

:: 3. Stop the service (if maintenance or configuration updates are needed)
your-agent-name.exe stop

:: 4. Completely uninstall the service from the Windows system
your-agent-name.exe uninstall

```
## 🐧 Linux (with Sudo privileges)

```
:: 1. Register the Agent into the Linux Systemd management system
sudo ./your-agent install

:: 2. Start the background daemon service
sudo ./your-agent start

:: 3. Check the real-time operational status directly from Systemd
sudo systemctl status MonitoringAgent

:: 4. Stop the service or completely uninstall it from the system
sudo ./your-agent stop
sudo ./your-agent uninstall
```
