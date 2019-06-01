# Implementing a controller

The basic logic of our CronJob controller is this:

1. Load the named CronJob

2. List all active jobs, and update the status

3. Clean up old jobs according to the history limits

4. Check if we're suspended (and don't do anything else if we are)

5. Get the next scheduled run

6. Run a new job if it's on schedule, not past the deadline, and not
   blocked by our concurrency policy

7. Requeue when we either see a running job (done automatically) or it's
   time for the next scheduled run.

{{#literatego ./testdata/project/controllers/cronjob_controller.go}}

That was a doozy, but now we've got a working controller.  Let's test
against the cluster, then, if we don't have any issues, deploy it!
