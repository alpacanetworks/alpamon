import os
import sys
import shutil

if sys.version_info >= (3, 9):
    import importlib.resources
else:
    from pkg_resources import resource_filename


DEFAULT_EDITOR = 'vi'

CONFIG_TEMPLATE = ('''[server]
url = %(url)s
id = %(id)s
key = %(key)s

[ssl]
verify = %(verify)s
ca_cert = %(ca_cert)s

[logging]
debug = %(debug)s
''')

SERVICE_TEMPLATE = ('''[Unit]
Description=%(display_name)s for Alpaca Infra Platform
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


def get_base_prefix_compat():
    """Get base/real prefix, or sys.prefix if there is none."""
    return (
        getattr(sys, 'base_prefix', None)
        or getattr(sys, 'real_prefix', None)
        or sys.prefix
    )


def in_virtualenv():
    return sys.prefix != get_base_prefix_compat()


def usage():
    sys.stderr.write('%s install|uninstall|configure\n' % sys.argv[0])
    sys.stderr.flush()


class Service:
    def __init__(self, name):
        self.name = name
        self.display_name = ' '.join(map(lambda x: x.capitalize(), name.split('-')))

    def get_resource_path(self, resource):
        if sys.version_info >= (3, 9):
            return importlib.resources.files(self.name).joinpath(resource)
        else:
            return resource_filename(__name__, resource)

    def write_config(self, target, template=CONFIG_TEMPLATE):
        with open(target, 'w') as f:
            f.write(template % {
                'url': os.environ.get('ALPACON_URL', 'https://alpacon.io'),
                'id': os.environ.get('PLUGIN_ID', ''),
                'key': os.environ.get('PLUGIN_KEY', ''),
                'verify': os.environ.get('ALPACON_SSL_VERIFY', 'true'),
                'ca_cert': os.environ.get('ALPACON_CA_CERT', ''),
                'debug': os.environ.get('PLUGIN_DEBUG', 'true'),
            })

    def write_service(self, target, template=SERVICE_TEMPLATE):
        if in_virtualenv():
            base_dir = sys.prefix
        else:
            base_dir = '/usr/local'
        exec_start = os.path.join(base_dir, 'bin', self.name)

        with open(target, 'w') as f:
            f.write(SERVICE_TEMPLATE % {
                'exec': exec_start,
                'display_name': self.display_name,
            })

    def configure(target):
        try:
            os.mkdir('/etc/alpamon')
            os.chmod('/etc/alpamon', 0o700)
        except FileExistsError:
            pass

        # copy configuration file if not exists
        if not os.path.exists(target):
            self.write_config(target)
        
        # open an editor for the configuration file
        os.system('%s %s' % (get_editor(), target))

    def install(self):
        print('Installing systemd service...')
        shutil.copyfile(
            self.get_resource_path('config/tmpfile.conf'),
            TMPFILE_TARGET
        )
        os.system('/bin/systemd-tmpfiles --create')

        self.write_config()
        self.write_service()

        os.system('/bin/systemctl daemon-reload')
        os.system('/bin/systemctl start %s.service' % self.name)
        os.system('/bin/systemctl enable %s.service' % self.name)
        os.system('/bin/systemctl --no-pager status %s.service' % self.name)
        print(
            '%s has been installed as a systemd service and '
            'will be launched automatically on system boot.' % self.display_name
        )

    def uninstall(self):
        print('Uninstalling systemd service...')
        os.system('/bin/systemctl stop %s.service' % self.name)
        os.system('/bin/systemctl disable %s.service' % self.name)
        os.remove(TMPFILE_TARGET)
        os.remove(SERVICE_TARGET)
        os.system('/bin/systemctl daemon-reload')
        print('Removing configuration files...')
        try:
            shutil.rmtree('/var/lib/alpamon')
        except:
            pass
        try:
            os.remove('/etc/alpamon/%s.conf' % self.name)
            os.rmdir('/etc/alpamon')
        except:
            pass
        print('%s has been removed successfully!' % self.name)
        print('Run "rm -rf /var/log/alpamon" to remove logs as well.')
