import os
import sys
import shutil

if sys.version_info >= (3, 9):
    import importlib.resources
else:
    from pkg_resources import resource_filename


CONFIG_TARGET = '/etc/alpamon/alpamon.conf'
TMPFILE_TARGET = '/usr/lib/tmpfiles.d/alpamon.conf'
SERVICE_TARGET = '/lib/systemd/system/alpamon.service'
DEFAULT_EDITOR = 'vi'

CONFIG_TEMPLATE = ('''[server]
url = %(url)s
id = %(id)s
key = %(key)s

[ssl]
verify = %(verify)s
ca_cert = %(ca_cert)s

[logging]
debug = %(debug)s''')

SERVICE_TEMPLATE = ('''[Unit]
Description=Alpamon Agent for Alpaca Infra Platform
After=network.target syslog.target

[Service]
Type=simple
ExecStart=%(exec)s
WorkingDirectory=/var/lib/alpamon
Restart=always
StandardOutput=null
StandardError=null

[Install]
WantedBy=multi-user.target
''')


def get_editor():
    return (os.environ.get('VISUAL') or os.environ.get('EDITOR') or DEFAULT_EDITOR)


def get_resource_path(resource):
    if sys.version_info >= (3, 9):
        return importlib.resources.files('alpamon').joinpath(resource)
    else:
        return resource_filename(__name__, resource)


def get_base_prefix_compat():
    """Get base/real prefix, or sys.prefix if there is none."""
    return (
        getattr(sys, 'base_prefix', None)
        or getattr(sys, 'real_prefix', None)
        or sys.prefix
    )


def in_virtualenv():
    return sys.prefix != get_base_prefix_compat()


def write_config():
    with open(CONFIG_TARGET, 'w') as f:
        f.write(CONFIG_TEMPLATE % {
            'url': os.environ.get('ALPACON_URL', 'https://alpacon.io'),
            'id': os.environ.get('ALPAMON_ID', ''),
            'key': os.environ.get('ALPAMON_KEY', ''),
            'verify': os.environ.get('ALPACON_SSL_VERIFY', 'true'),
            'ca_cert': os.environ.get('ALPACON_CA_CERT', ''),
            'debug': os.environ.get('ALPAMON_DEBUG', 'true'),
        })


def write_service():
    if in_virtualenv():
        base_dir = sys.prefix
    else:
        base_dir = '/usr/local'
    exec_start = os.path.join(base_dir, 'bin/alpamon')

    with open(SERVICE_TARGET, 'w') as f:
        f.write(SERVICE_TEMPLATE % {
            'exec': exec_start,
        })


def configure():
    try:
        os.mkdir('/etc/alpamon')
        os.chmod('/etc/alpamon', 0o700)
    except FileExistsError:
        pass

    # copy configuration file if not exists
    if not os.path.exists(CONFIG_TARGET):
        write_config()
    
    # open an editor for the configuration file
    os.system('%s %s' % (get_editor(), CONFIG_TARGET))


def install():
    print('Installing systemd service...')
    shutil.copyfile(
        get_resource_path('config/tmpfile.conf'),
        TMPFILE_TARGET
    )
    os.system('/bin/systemd-tmpfiles --create')

    write_config()
    write_service()

    os.system('/bin/systemctl daemon-reload')
    os.system('/bin/systemctl start alpamon.service')
    os.system('/bin/systemctl enable alpamon.service')
    os.system('/bin/systemctl --no-pager status alpamon.service')
    print(
        'Alpamon has been installed as a systemd service and '
        'will be launched automatically on system boot.'
    )


def uninstall():
    print('Uninstalling systemd service...')
    os.system('/bin/systemctl stop alpamon.service')
    os.system('/bin/systemctl disable alpamon.service')
    os.remove(TMPFILE_TARGET)
    os.remove(SERVICE_TARGET)
    os.system('/bin/systemctl daemon-reload')
    print('Removing configuration files...')
    try:
        shutil.rmtree('/var/lib/alpamon')
    except:
        pass
    try:
        shutil.rmtree('/etc/alpamon')
    except:
        pass
    print('Alpamon has been removed successfully! Run "rm -rf /var/log/alpamon" to remove logs as well.')


def usage():
    sys.stderr.write('%s install|uninstall|configure\n' % sys.argv[0])
    sys.stderr.flush()


def main():
    if len(sys.argv) < 2:
        usage()
        sys.exit(1)

    if sys.argv[1] == 'install':
        install()
    elif sys.argv[1] == 'uninstall':
        uninstall()
    elif sys.argv[1] == 'configure':
        configure()


if __name__ == '__main__':
    main()
