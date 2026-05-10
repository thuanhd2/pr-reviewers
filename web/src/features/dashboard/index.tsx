import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'

const statusLabels: Record<string, string> = {
  pending: 'Pending', reviewing: 'Reviewing', drafted: 'Drafted',
  posted: 'Posted', failed: 'Failed', closed: 'Closed',
}

const verdictColors: Record<string, string> = {
  approve: 'bg-green-100 text-green-700',
  request_changes: 'bg-red-100 text-red-700',
  comment: 'bg-blue-100 text-blue-700',
}

export default function Dashboard() {
  useWebSocket()
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: api.getDashboard,
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
          <h2 className='text-2xl font-bold tracking-tight'>Dashboard</h2>
          {isLoading ? (
            <div className='space-y-4'>
              <Skeleton className='h-24' />
              <Skeleton className='h-48' />
            </div>
          ) : (
            <>
              <div className='grid grid-cols-3 lg:grid-cols-6 gap-4'>
                {data?.counts && Object.entries(data.counts).map(([key, count]) => (
                  <Card key={key}>
                    <CardHeader className='pb-2'>
                      <CardTitle className='text-sm text-muted-foreground'>
                        {statusLabels[key] || key}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className='text-3xl font-bold'>{count as number}</div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              <h3 className='text-lg font-semibold'>Recent Reviews</h3>
              <div className='space-y-3'>
                {data?.recent?.map((r: any) => (
                  <Card key={r.id}>
                    <CardContent className='flex items-center justify-between py-4'>
                      <div>
                        <span className='font-medium'>Review #{r.id}</span>
                        <span className='text-sm text-muted-foreground ml-2'>
                          PR #{r.pull_request_id}
                        </span>
                        <p className='text-sm mt-1 text-muted-foreground line-clamp-2'>
                          {r.summary}
                        </p>
                      </div>
                      <Badge className={verdictColors[r.overall_verdict] || 'bg-gray-100'}>
                        {r.overall_verdict}
                      </Badge>
                    </CardContent>
                  </Card>
                ))}
                {(!data?.recent || data.recent.length === 0) && (
                  <p className='text-muted-foreground'>No reviews yet.</p>
                )}
              </div>
            </>
          )}
        </div>
      </Main>
    </>
  )
}
