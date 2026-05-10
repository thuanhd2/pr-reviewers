import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { api } from '@/lib/api'

export default function PRDetail() {
  const { prId } = useParams({ from: '/_authenticated/prs/$prId' })
  const { data: pr, isLoading } = useQuery({
    queryKey: ['prs', Number(prId)],
    queryFn: () => api.getPR(Number(prId)),
  })

  if (isLoading)
    return (
      <>
        <Header><div className='me-auto' /><ThemeSwitch /></Header>
        <Main>
          <div className='space-y-4'>
            <Skeleton className='h-32' />
            <Skeleton className='h-64' />
          </div>
        </Main>
      </>
    )

  if (!pr)
    return (
      <>
        <Header><div className='me-auto' /><ThemeSwitch /></Header>
        <Main><p>PR not found</p></Main>
      </>
    )

  return (
    <>
      <Header>
        <div className='me-auto' />
        <ThemeSwitch />
      </Header>
      <Main>
        <div className='space-y-6'>
          <div>
            <h2 className='text-2xl font-bold tracking-tight'>{pr.title}</h2>
            <p className='text-sm text-muted-foreground mt-1'>
              {pr.repo_full_name}#{pr.number} by {pr.author} · {pr.base_branch} ← {pr.head_branch}
            </p>
            <div className='flex gap-2 mt-2'>
              <Badge variant='outline'>{pr.status}</Badge>
              <a href={pr.url} target='_blank' rel='noopener noreferrer'>
                <Button variant='outline' size='sm'>
                  View on GitHub
                </Button>
              </a>
            </div>
          </div>
          <Tabs defaultValue='reviews'>
            <TabsList>
              <TabsTrigger value='reviews'>
                Reviews ({pr.reviews?.length ?? 0})
              </TabsTrigger>
              <TabsTrigger value='info'>Info</TabsTrigger>
            </TabsList>
            <TabsContent value='reviews' className='space-y-3'>
              {pr.reviews?.map((r: any) => (
                <Link
                  key={r.id}
                  to='/prs/$prId/reviews/$reviewId'
                  params={{ prId: String(pr.id), reviewId: String(r.id) }}
                >
                  <Card className='hover:shadow-md transition-shadow cursor-pointer'>
                    <CardContent className='flex items-center justify-between py-4'>
                      <div>
                        <span className='text-sm text-muted-foreground'>
                          Commit {r.commit_sha?.slice(0, 7)}
                        </span>
                        <p className='mt-1 line-clamp-2'>{r.summary}</p>
                      </div>
                      <div className='flex gap-2 items-center'>
                        <Badge>{r.overall_verdict}</Badge>
                        <Badge variant='outline'>{r.status}</Badge>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              ))}
              {(!pr.reviews || pr.reviews.length === 0) && (
                <p className='text-muted-foreground'>No reviews yet.</p>
              )}
            </TabsContent>
            <TabsContent value='info'>
              <Card>
                <CardContent className='py-4 space-y-2 text-sm'>
                  <div>
                    <span className='font-medium'>Head SHA:</span> {pr.head_sha}
                  </div>
                  <div>
                    <span className='font-medium'>Worktree:</span>{' '}
                    {pr.worktree_path || 'N/A'}
                  </div>
                  <div>
                    <span className='font-medium'>First seen:</span>{' '}
                    {new Date(pr.created_at).toLocaleDateString()}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </Main>
    </>
  )
}
