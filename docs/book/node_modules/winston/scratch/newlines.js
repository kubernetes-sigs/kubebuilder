'use strict';

const { createLogger, format, transports } = require('../');
const logger = createLogger({
  format: format.printf(info => info.message),
  transports: [new transports.Console()]
});

logger.info(`
  Hey this is a multi line string
  wouldn't it be great it if it was displayed
  in the console as a multi-line string instead
  of a JSON.stringify-ed object with \\n characters?
`)
