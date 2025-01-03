# systat

A comprehensive system information and monitoring CLI tool.

## Features

- **DNS Queries**: Query DNS information for *.admin.uds.dev and \*.uds.dev domains
- **System Information**: Detailed system hardware and OS information
- **Network Monitoring**: Network interface and routing information (Linux only)
- **Process Management**: List and monitor system processes
- **Disk Usage**: Monitor disk space and I/O statistics
- **System Metrics**: Real-time CPU, memory, and system metrics
- **Kubernetes Info**: Basic Kubernetes cluster information
- **Beautiful Output**:
  - Colorized output using github.com/alecthomas/chroma
  - Structured logging using github.com/charmbracelet/log
  - JSON output support for scripting

## Installation

```bash
go install github.com/noxsios/systat/cmd/systat@latest
```

## Usage

### System Information

```bash
# Get basic system information
systat sysinfo

# Get detailed system metrics
systat metrics

# Monitor disk usage
systat disk

# View network information (Linux only)
systat network

# List processes
systat process
```

### DNS and Kubernetes

```bash
# Query DNS information
systat dns keycloak.admin.uds.dev

# Get Kubernetes cluster info
systat k8s
```

### Output Options

```bash
# Get JSON output for any command
systat <command> --json

# Raw output without formatting
systat <command> --raw

# Watch mode for real-time updates
systat <command> --watch

# Set log level
systat <command> --log-level debug
```

## Requirements

- Go 1.21 or higher
- Linux for network monitoring features
- kubectl configured for Kubernetes features

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
