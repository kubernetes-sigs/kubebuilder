const winston = require('../');
const { createLogger, format, transports } = winston;

const logger = createLogger({
  transports: [
    new transports.Console()
  ]
});

process.on('unhandledRejection', function (reason, p) {
  console.dir(arguments);
  logger.error({ message: 'Unhandled Rejection at Promise', reason });
});

Promise.reject(new Error('fail'))
