const winston = require('../');
const { createLogger, format, transports } = winston;

const testLogger = winston.createLogger ({
  format: format.simple(),
  transports :[
    new winston.transports.Console({
      level: 'silly'
    })
  ]
})

Object.entries(winston.config.npm.levels).forEach(([level, index]) => {
  testLogger.log(level, `Logging to "${level}" with a numeric index of ${index}`)
});
