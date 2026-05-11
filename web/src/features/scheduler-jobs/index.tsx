import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'

const statusColors: Record<string, string> = {
  idle: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  running: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  failed: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
}

function formatTime(t: string | null) {
  if (!t) return 'Never'
  return new Date(t).toLocaleString()
}

export default function SchedulerJobs() {
  useWebSocket()
  const { data: jobs, isLoading } = useQuery({
    queryKey: ['scheduler-jobs'],
    queryFn: api.getSchedulerJobs,
    refetchInterval: 30_000,
  })

  return (
    <>
      <Header>
        <div className='me-auto' />
        <ThemeSwitch />
      </Header>
      <Main>
        <div className='space-y-6'>
          <h2 className='text-2xl font-bold tracking-tight'>Scheduler Jobs</h2>
          {isLoading ? (
            <div className='space-y-4'>
              <Skeleton className='h-32' />
              <Skeleton className='h-32' />
              <Skeleton className='h-32' />
            </div>
          ) : (
            <div className='grid gap-4'>
              {(jobs ?? []).map((job: any) => (
                <Card key={job.id}>
                  <CardHeader className='pb-3'>
                    <div className='flex items-center justify-between'>
                      <CardTitle className='text-lg'>{job.job_name}</CardTitle>
                      <Badge className={statusColors[job.status] || 'bg-gray-100'}>
                        {job.status}
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className='grid grid-cols-2 lg:grid-cols-4 gap-4 text-sm'>
                      <div>
                        <span className='text-muted-foreground'>Cron</span>
                        <p className='font-mono mt-1'>{job.cron_spec}</p>
                      </div>
                      <div>
                        <span className='text-muted-foreground'>Last Run</span>
                        <p className='mt-1'>{formatTime(job.last_run_at)}</p>
                      </div>
                      <div>
                        <span className='text-muted-foreground'>Next Run</span>
                        <p className='mt-1'>{formatTime(job.next_run_at)}</p>
                      </div>
                      <div>
                        <span className='text-muted-foreground'>Task Type</span>
                        <p className='font-mono mt-1'>{job.task_type}</p>
                      </div>
                    </div>
                    {job.last_error && (
                      <div className='mt-4 p-3 rounded-md bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 text-sm'>
                        <span className='font-semibold'>Error: </span>
                        {job.last_error}
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
              {(!jobs || jobs.length === 0) && (
                <p className='text-muted-foreground'>No scheduler jobs found. Start the worker to see jobs.</p>
              )}
            </div>
          )}
        </div>
      </Main>
    </>
  )
}
