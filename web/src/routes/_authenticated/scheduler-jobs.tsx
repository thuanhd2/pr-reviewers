import { createFileRoute } from '@tanstack/react-router'
import SchedulerJobs from '@/features/scheduler-jobs'

export const Route = createFileRoute('/_authenticated/scheduler-jobs')({
  component: SchedulerJobs,
})
