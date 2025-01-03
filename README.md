# systat

A system information and DNS query CLI tool.

## Features

- DNS queries for *.admin.uds.dev and \*.uds.dev domains using github.com/miekg/dns
- System information output using github.com/zcalusic/sysinfo
- Detailed system metrics using github.com/shirou/gopsutil/v3
- Colorized output using github.com/alecthomas/chroma
- Structured logging using github.com/charmbracelet/log

## Installation

```bash
go install github.com/noxsios/systat/cmd/systat@latest
```

## Usage

```bash
# Get DNS information
systat dns keycloak.admin.uds.dev

# Get system information (sysinfo)
systat sysinfo

# Get detailed system metrics (gopsutil)
systat metrics
```
