# alpamon-go
New Go-based Secure Server Agent for Alpaca Infra Platform

**Alpamon** is a server agent for **Alpaca Infra Platform**. Each server should have Alpamon installed to be controlled via Alpacon.

This guide outlines the step-by-step process for installing Alpamon within a development environment. The installation requires an active Internet connection or the appropriate configuration of a proxy server.

## Getting started
To build Alpamon, ensure you have:
- [Go](https://go.dev/doc/install) version 1.22 or higher installed.
  - Make sure `$GOPATH` is set and `$GOPATH/bin` is added to your system’s `PATH`.
  
## Installation
Download the latest `Alpamon-Go` directly from our releases page or install it using package managers on Linux.

### Linux

#### Debian and Ubuntu
```bash
curl -s https://packagecloud.io/install/repositories/alpacanetworks/alpamon/script.deb.sh?any=true | sudo bash

sudo apt-get install alpamon
```

#### CentOS and RHEL
```bash
curl -s https://packagecloud.io/install/repositories/alpacanetworks/alpamon/script.rpm.sh?any=true | sudo bash

sudo yum install alpamon
```

### macOS

#### Clone the source code
To get started on macOS, clone the source code from the repository:
```bash
git clone https://github.com/alpacanetworks/alpamon-go.git
```

#### Install Go dependencies
Make sure you have Go installed. Then, navigate to the project root and download the necessary Go packages:
```bash
go mod tidy
```

## Configure

Alpamon can be configured via the files listed below.

- `/etc/alpamon/alpamon.conf`
- `~/.alpamon.conf`

It is recommended to use `/etc/alpamon/alpamon.conf` for deployment, but you can use `~/.alpamon.conf` for development.

```ini
[server]
url = http://localhost:8000
id = 
key = 

[ssl]
verify = true
ca_cert = 

[logging]
debug = true
```

### Configuration details

- `server`: Server settings
    - `url`: The URL for Alpaca Console. If you are in a local development environment, this will be `https://localhost:8000`.
    - `id`: Server ID
    - `key`: Server Key
    - `ca_cert`: Path for the CA certificate
- `logging`: Logging settings
    - `debug`: Whether to print debug logs or not

## Run

### Local environment

To run Alpamon in a local development environment, navigate to the cmd/alpamon directory and run the application using Go:
```sh
cd /path/to/alpamon-go/cmd/alpamon

go run main.go
```
Ensure that you are in the correct directory (`/cmd/alpamon`), as this is where the `main.go` file resides.

### Deploy as a service

For Linux systems supporting `systemd`, you can run `alpamon` as a systemd service. In this case, you need to adapt `alpamon/config/alpamon.service` for your environment.

Specifically, `ExecStart` should be something like `/usr/local/bin/alpamon`.

Run the following commands to prepare system directories.

```sh
sudo cp alpamon/config/tmpfile.conf /usr/lib/tmpfiles.d/alpamon.conf
sudo systemd-tmpfiles --create
```

Run the following commands to install a systemd service.

```sh
sudo cp alpamon/config/alpamon.service /lib/systemd/system/alpamon.service
sudo systemctl daemon-reload
sudo systemctl start alpamon.service
sudo systemctl enable alpamon.service
systemctl status alpamon.service
```

The result would look like the following. The status must be loaded and active (running).

```
alpamon.service - alpamon agent for alpaca infra platform
     Loaded: loaded (/lib/systemd/system/alpamon.service; enabled; vendor preset: enabled)
     Active: active (running) since Thu 2023-09-28 23:48:55 KST; 4 days ago
```