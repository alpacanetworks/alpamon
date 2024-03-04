import os
import sys
import shutil

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


def get_editor():
    return (os.environ.get('VISUAL') or os.environ.get('EDITOR') or DEFAULT_EDITOR)


def configure():
    try:
        os.mkdir('/etc/alpamon')
        os.chmod('/etc/alpamon', 0o700)
    except FileExistsError:
        pass

    # copy configuration file if not exists
    if not os.path.exists(CONFIG_TARGET):
        shutil.copyfile(
            resource_filename(__name__, 'config/alpamon.conf'),
            CONFIG_TARGET
        )
    
    # open an editor for the configuration file
    os.system('%s %s' % (get_editor(), CONFIG_TARGET))


def install():
    print('Installing systemd service...')
    shutil.copyfile(
        resource_filename(__name__, 'config/tmpfile.conf'),
        TMPFILE_TARGET
    )
    os.system('/bin/systemd-tmpfiles --create')
    with open(CONFIG_TARGET, 'w') as f:
        f.write(CONFIG_TEMPLATE % {
            'url': os.environ.get('ALPACON_URL', 'https://alpacon.io'),
            'id': os.environ.get('ALPAMON_ID', ''),
            'key': os.environ.get('ALPAMON_KEY', ''),
            'verify': os.environ.get('ALPACON_SSL_VERIFY', 'true'),
            'ca_cert': os.environ.get('ALPACON_CA_CERT', ''),
            'debug': os.environ.get('ALPAMON_DEBUG', 'true'),
        })
    
    shutil.copyfile(
        resource_filename(__name__, 'config/alpamon.service'),
        SERVICE_TARGET
    )
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
