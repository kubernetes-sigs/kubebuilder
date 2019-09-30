# Metrics

By default, controller-runtime builds a global prometheus registry and
publishes a collection of performance metrics for each controller.

## Publishing Additional Metrics

If you wish to publish additional metrics from your controllers, this
can be easily achieved by using the global registry from
`controller-runtime/pkg/metrics`.

One way to achieve this is to declare your collectors as global variables and then register them using `init()`.

For example:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    goobers = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goobers_total",
            Help: "Number of goobers proccessed",
        },
    )
    gooberFailures = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goober_failures_total",
            Help: "Number of failed goobers",
        },
    )
)

func init() {
    // Register custom metrics with the global prometheus registry
    metrics.Registry.MustRegister(goobers, gooberFailures)
}
```

You may then record metrics to those collectors from any part of your
reconcile loop, and those metrics will be available for prometheus or
other openmetrics systems to scrape.