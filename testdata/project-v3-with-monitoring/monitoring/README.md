
## Operator Monitoring

The monitoring sub folder should be used to add new Prometheus metrics, alerts and other monitoring related resources, that are specific to this operator.

Using it will help the operator developers to start adding monitoring to the operator and avoid common pitfalls.
It also includes useful tools, like an auto metrics documentation generator.


### How to add  a new metric

1. Edit/Copy the /monitoring/metrics/example_metrics.go file and replace the example metric with your metrics.

	1.1. For each metric provide the metric name, description and type.

	1.2. Make sure to add the metrics value update logic to this file and not in your core operator code.

2. Call the function to update your metric value (defined in example_metrics.go) from your operator code.

3. Use the /monitoring/metrics/util/util.go file to see the available Prometheus API functions you can use and additional metrics related help functions.

4. Register the metrics - Update the metrics list in /monitoring/metrics/metrics.go to register your metrics.
Make sure to uncomment the call for the register function in main.go.


### How to generate metrics documentation

1. In you operator root directory run: 
make generate-metricsdocs

2. To view the generated metrics documentation go to /docs/monitoring/metrics.md


### How to add a new alert
TBA
### How to add unit tests for alerts
TBA
### How to add an alert runbook
TBA
