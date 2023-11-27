import logging
import logging.config


def _log(self, level, msg, args, exc_info=None, extra={}):
    if 'program' not in extra:
        extra['program'] = program
    self._log_super(level, msg, args, exc_info, extra)


def configure(program_name, config):
    global program
    program = program_name

    # override _log to inject program field globally
    logging.Logger._log_super = logging.Logger._log
    logging.Logger._log = _log

    logging.config.dictConfig(config)
