# Implementing a controller

The basic logic of the CronJob controller is this:

1. Load the named CronJob

2. List all active jobs, and update the status

3. Clean up old jobs according to the history limits

4. Check if suspended (and don't do anything else if it is)

5. Get the next scheduled run

6. Run a new job if it's on schedule, not past the deadline, and not
   blocked by the concurrency policy

7. Requeue when either seeing a running job (done automatically) or it's
   time for the next scheduled run.

{{#literatego ./testdata/project/internal/controller/cronjob_controller.go}}

That was a doozy, but now you have a working controller.  Test
against the cluster, then, if there are no issues, deploy it!
