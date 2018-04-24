var winston = require('../');

const logger = winston.createLogger({
  transports: [new winston.transports.Console]
});

var test = {a: "a", b: new Buffer(10)};
logger.log('info', ' Test Log Message' , test);
