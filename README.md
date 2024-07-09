# Alpamon

Alpamon is a server agent for Alpaca Infra Platform. Each server should have Alpamon installed to be controlled via Alpacon.

This guide outlines the step-by-step process for installing Alpamon within a development environment. The installation requires an active Internet connection or the appropriate configuration of a proxy server.

## Install system packages

Alpamon runs on Python 3.4 or above. Python pip is required for package installation. This procedure assumes you are a standard user with `sudo` priviledge. Alpamon itself does not require root priviledge for the development, but some of its features require `sudo`. For full tests, it is recommended to use docker.

### Supported platforms

- Ubuntu 18.04 (Bionic Beavor) or higher
- Debian 10 (Buster) or higher
- RedHat Enterprise Linux 8, 9
- CentOS 7
- Amazon Linux
- macOS (only for development purposes)

### macOS

```sh
brew install python3
brew install --cask osquery
pip3 install virtualenv
```

### Ubuntu

```sh
sudo -HE apt install python3 python3-pip
sudo -HE pip3 install virtualenv
export OSQUERY_KEY=1484120AC4E9F8A1A577AEEE97A80C63C9D8B80B
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys $OSQUERY_KEY
sudo add-apt-repository 'deb [arch=amd64] https://pkg.osquery.io/deb deb main'
sudo apt-get update
sudo apt-get install osquery
```

### CentOS

```sh
sudo -HE yum install python3 python3-pip
sudo -HE pip3 install virtualenv
curl -L https://pkg.osquery.io/rpm/GPG | sudo tee /etc/pki/rpm-gpg/RPM-GPG-KEY-osquery
sudo yum-config-manager --add-repo https://pkg.osquery.io/rpm/osquery-s3-rpm.repo
sudo yum-config-manager --enable osquery-s3-rpm-repo
sudo yum install osquery
```

For now, we use `osquery` to collect system information. About installing osquery, You can find more resources from the [official documentation](https://osquery.readthedocs.io/en/latest/installation/install-linux/).

## Clone the source code

```sh
git clone https://github.com/alpacanetworks/alpamon.git
```

## Install python virtualenv

Alpamon use `virtualenv` not to override the installed system packages. Create a virtual environment to run Alpamon and its subsidiary applications. It's recommended to create a directory in your home directory as `alpamon` and store all related packages in it. The actual path may vary depending on your system environment. You need to source `alpamon/env/bin/activate` before executing any command for alpamon.

```sh
cd alpamon
virtualenv env --python=python3
source env/bin/activate
pip install -U pip
```

## Install alpamon package

Install alpamon in development mode. `pip` will install all the required packages automatically. Make sure you have network connectivity before executing following commands.

```sh
./setup.py develop
```

Run `pip list` to list all packages installed in your virtual environment. The actual list may vary by the version. It's important to check that your client has been installed in development mode, and the location is also printed.

```sh
pip list
```

```
Package            Version   Location                                
------------------ --------- ----------------------------------------
certifi            2021.10.8 
charset-normalizer 2.0.7     
distro             1.6.0     
docutils           0.18.1b0  
alpamon            1.0.0      /<path>/<to>/<alpamon>
idna               3.3       
lockfile           0.12.2    
pid                3.0.4     
pip                20.0.2    
python-daemon      2.3.0     
requests           2.26.0    
setuptools         46.1.3    
urllib3            1.26.7    
websocket-client   1.2.1     
wheel              0.34.2
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

Please refer to the credentials in the README of `alpacon-server` for development environment setup.

You may also obtain the `id` and `key` from http://localhost:3000. After login, go to "Servers" menu and click "New server". Add proper information, and you will get a script including `ALPACON_URL`, `ALPAMON_ID`, and `ALPAMON_KEY`. Use this information when configuring `alpamon.conf`.

## Run

### Local environment

Type `alpamon` to run. Make sure to be in the virtual environment.

```sh
alpamon
```

#### Deploy as a service

For Linux systems supporting `systemd`, you can run `alpamon` as a systemd service. In this case, you need to adapt `alpamon/config/alpamon.service` for your environment.

Specifically, `ExecStart` should be something like `/<your>/<home>/alpamon/env/bin/alpamon`. The actual path may vary depending on your virtual environment settings.

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

### Docker

You can also use docker to test alpamon in various Linux distributions. We use Docker Desktop to test alpamon on following distributions.

- Ubuntu: 18.04, 20.04, 22.04
- Debian: 10, 11
- RedHat: 8, 9
- CentOS: 7

#### Build

Build docker images with the build script.

```
./tests/build.sh
```

#### Run

You can run containers for these images in Docker Desktop or using command line like below.

```
docker run --mount type=bind,source="$(pwd)",target=/opt/alpamon alpamon:ubuntu-22.04
```

## Notes

When testing `alpamon` on ubuntu docker image, install `curl`, `ca-certificates`, and `systemd` before running the install script. They are not included by default as the docker image is kept minimal.

```bash
apt update && apt install -y --no-install-recommends curl systemd ca-certificates
```

## Release

### Regular release

Regular releases are built by GitHub Actions. To make a release, you need two steps.

#### Bump version in code

Update `alpamon/__init__.py` to change the current version. We use [semantic versioning](https://semver.org/).

#### Draft a release

At GitHub, click "Draft a new release" button. Please be aware of the followings.

- A tag can be created with syntax like `1.0.0`.
- Release title should look like `v1.0.0`.
- Include proper release notes.

Regular releases are distributed automatically to Alpacon.

### Manual release

For manual release, we use Python standard wheel file. You can build the package with the following command.

```bash
./setup.py bdist_wheel
```

You may find the output from `dist/`. The file should look like `alpamon-1.0.0-py3-none-any.whl`.
