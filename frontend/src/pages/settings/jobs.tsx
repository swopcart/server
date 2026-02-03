import { useState, useEffect } from "react";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { listJobs, triggerJob, updateJobSettings } from "@/lib/api/jobs";
import type { Job } from "@/lib/api/jobs";
import { LucidePlay, LucideClock } from "lucide-react";
import { useAsync } from "@/hooks/use-async";
import { useAsyncFn } from "@/hooks/use-async";

export function JobsPage() {
  const [refreshCounter, setRefreshCounter] = useState(0);
  const { data: jobs, loading, error } = useAsync(listJobs, [refreshCounter]);
  const [
    { loading: triggering, error: triggerError },
    executeTrigger,
    resetTrigger,
  ] = useAsyncFn(triggerJob);
  const [
    { loading: updating, error: updateError },
    executeUpdate,
    resetUpdate,
  ] = useAsyncFn(updateJobSettings);
  const [triggeringJob, setTriggeringJob] = useState<string | null>(null);
  const [updatingJob, setUpdatingJob] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Auto-refresh every minute
  useEffect(() => {
    const interval = setInterval(() => {
      setRefreshCounter((prev) => prev + 1);
    }, 60000); // 60000ms = 1 minute

    return () => clearInterval(interval);
  }, []);

  const formatSchedule = (schedule: string | null) => {
    if (!schedule) return "On-demand only";
    // Parse cron expression for common patterns
    if (schedule === "0 0 3 * * *") return "Daily at 3:00 AM";
    if (schedule.startsWith("0 0 ")) return "Daily";
    if (schedule.startsWith("0 ")) return "Hourly";
    return schedule;
  };

  const formatDate = (dateString: string | null) => {
    if (!dateString) return "Never";
    return new Date(dateString).toLocaleString();
  };

  const canTrigger = (job: Job): boolean => {
    // Can trigger if no parameters required or all have defaults
    return job.defaultParameters !== null || job.defaultParameters === null;
  };

  const handleToggleEnabled = async (job: Job, enabled: boolean) => {
    setUpdatingJob(job.name);
    setSuccessMessage(null);
    resetUpdate();

    await executeUpdate(job.name, { enabled });

    if (!updateError) {
      // Refresh the jobs list to get the updated state
      setRefreshCounter((prev) => prev + 1);
    }

    setUpdatingJob(null);
  };

  const handleTrigger = async (job: Job) => {
    setTriggeringJob(job.name);
    setSuccessMessage(null);
    resetTrigger();

    // Parse default parameters if they exist
    let parameters: Record<string, string> | undefined;
    if (job.defaultParameters) {
      try {
        parameters = JSON.parse(job.defaultParameters);
      } catch {
        // If parsing fails, trigger without parameters
        parameters = undefined;
      }
    }

    await executeTrigger(job.name, parameters);

    if (!triggerError) {
      setSuccessMessage(`${job.name} has been queued for execution`);
      // Clear success message after 5 seconds
      setTimeout(() => setSuccessMessage(null), 5000);
    }

    setTriggeringJob(null);
  };

  if (loading) {
    return (
      <>
        <Header title="Background Jobs" backHref="/settings" />
        <div className="p-6 flex justify-center">
          <Spinner />
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <Header title="Background Jobs" backHref="/settings" />
        <div className="p-6">
          <p className="text-destructive">
            {error.message || "Failed to load jobs"}
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <Header title="Background Jobs" backHref="/settings" />
      <Container className="p-6 flex flex-col gap-4">
        {successMessage && (
          <div className="p-3 border border-green-500 bg-green-50 dark:bg-green-950 rounded-lg text-sm text-green-900 dark:text-green-100">
            {successMessage}
          </div>
        )}

        {triggerError && (
          <div className="p-3 border border-destructive bg-destructive/10 rounded-lg text-sm text-destructive">
            {triggerError.message || "Failed to trigger job"}
          </div>
        )}

        {updateError && (
          <div className="p-3 border border-destructive bg-destructive/10 rounded-lg text-sm text-destructive">
            {updateError.message || "Failed to update job"}
          </div>
        )}

        <div className="flex flex-col gap-3">
          {jobs?.map((job) => (
            <div
              key={job.uuid}
              className="p-4 border rounded-lg flex flex-col gap-3"
            >
              <div className="flex justify-between items-start gap-3">
                <div className="flex flex-col gap-1 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium">{job.name}</span>
                    {job.schedule && (
                      <Badge variant="outline" className="gap-1">
                        <LucideClock className="size-3" />
                        Scheduled
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {job.description}
                  </p>
                  <div className="flex items-center gap-2 mt-1">
                    <Switch
                      id={`job-enabled-${job.name}`}
                      checked={job.enabled}
                      onCheckedChange={(checked) =>
                        handleToggleEnabled(job, checked)
                      }
                      disabled={updating && updatingJob === job.name}
                    />
                    <Label
                      htmlFor={`job-enabled-${job.name}`}
                      className="text-sm cursor-pointer"
                    >
                      {job.enabled ? "Enabled" : "Disabled"}
                    </Label>
                  </div>
                </div>
                {canTrigger(job) && (
                  <Button
                    size="sm"
                    onClick={() => handleTrigger(job)}
                    disabled={triggering && triggeringJob === job.name}
                  >
                    <LucidePlay className="size-4 mr-1" />
                    {triggering && triggeringJob === job.name
                      ? "Running..."
                      : "Run Now"}
                  </Button>
                )}
              </div>

              <div className="flex flex-col gap-1 text-xs text-muted-foreground">
                <div className="flex gap-4 flex-wrap">
                  <div>
                    <span className="font-medium">Schedule:</span>{" "}
                    {formatSchedule(job.schedule)}
                  </div>
                  <div>
                    <span className="font-medium">Queue:</span> {job.queue}
                  </div>
                  <div>
                    <span className="font-medium">Priority:</span>{" "}
                    {job.priority}
                  </div>
                </div>
                <div className="flex gap-4 flex-wrap">
                  <div>
                    <span className="font-medium">Last run:</span>{" "}
                    {formatDate(job.lastRunAt)}
                  </div>
                  {job.nextRunAt && (
                    <div>
                      <span className="font-medium">Next run:</span>{" "}
                      {formatDate(job.nextRunAt)}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}

          {jobs?.length === 0 && (
            <p className="text-muted-foreground">No jobs configured.</p>
          )}
        </div>
      </Container>
    </>
  );
}
