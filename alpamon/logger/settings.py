import os
import logging.handlers

LOG_DIR = '/var/log/alpamon'
LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'verbose': {
            'format': '%(asctime)s [%(levelname)s] (%(name)s) %(message)s'
        },
        'simple': {
            'format': '[%(levelname)s] %(message)s'
        },
    },
    'handlers': {
        'socket': {
            'class': 'logging.handlers.SocketHandler',
            'host': 'localhost',
            'port': logging.handlers.DEFAULT_TCP_LOGGING_PORT,
            'level': 'INFO',
        },
        'file': {
            'class': 'logging.handlers.RotatingFileHandler',
            'filename':  os.path.join(LOG_DIR, 'alpamon.log') if os.path.exists(LOG_DIR) else 'alpamon.log',
            'formatter': 'verbose',
        },
        'console': {
            'class': 'logging.StreamHandler',
            'formatter': 'verbose',
        },
    },
    'loggers': {
        '': {
            'handlers': ['socket', 'file', 'console'],
            'level': 'DEBUG',
        },
        'alpamon.runner.shell': {
            'handlers': ['socket', 'file', 'console'],
            'level': 'INFO',
            'propagate': False,
        },
        'alpamon.io.reporter': {
            'handlers': ['file', 'console'],
            'level': 'INFO',
            'propagate': False,
        },
        'PidFile': {
            'handlers': ['socket', 'file', 'console'],
            'level': 'INFO',
            'propagate': False,
        },
        'websocket': {
            'handlers': ['file', 'console'],
            'level': 'WARN',
            'propagate': False,
        },
        'urllib3': {
            'handlers': ['file', 'console'],
            'level': 'DEBUG',
            'propagate': False,
        }
    },
}
