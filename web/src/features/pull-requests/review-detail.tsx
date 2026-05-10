import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { api } from '@/lib/api'

export default function ReviewDetail() {
  const { reviewId } = useParams({
    from: '/_authenticated/prs/$prId/reviews/$reviewId',
  })
  const queryClient = useQueryClient()
  const [editedComments, setEditedComments] = useState<Record<number, string>>({})
  const [editedSummary, setEditedSummary] = useState('')
  const [showApprove, setShowApprove] = useState(false)

  const { data: review, isLoading } = useQuery({
    queryKey: ['reviews', Number(reviewId)],
    queryFn: () => api.getReview(Number(reviewId)),
  })

  const updateMutation = useMutation({
    mutationFn: (data: any) => api.updateReview(Number(reviewId), data),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['reviews', Number(reviewId)] }),
  })

  const approveMutation = useMutation({
    mutationFn: () => api.approveReview(Number(reviewId)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['prs'] }),
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

  if (!review)
    return (
      <>
        <Header><div className='me-auto' /><ThemeSwitch /></Header>
        <Main><p>Review not found</p></Main>
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
          <div className='flex items-center justify-between'>
            <div>
              <h2 className='text-2xl font-bold tracking-tight'>
                Review #{review.id}
              </h2>
              <div className='flex gap-2 mt-2'>
                <Badge>{review.overall_verdict}</Badge>
                <Badge variant='outline'>{review.status}</Badge>
                <span className='text-sm text-muted-foreground'>
                  by {review.executor_name}
                </span>
              </div>
            </div>
            <div className='flex gap-2'>
              <Button
                variant='outline'
                onClick={() =>
                  updateMutation.mutate({
                    summary: editedSummary || review.summary,
                    comments: Object.entries(editedComments).map(([id, body]) => ({
                      id: Number(id),
                      body,
                    })),
                  })
                }
              >
                Save Changes
              </Button>
              {review.status !== 'posted' && (
                <Button onClick={() => setShowApprove(true)}>
                  Approve & Post
                </Button>
              )}
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Summary</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={editedSummary !== '' ? editedSummary : review.summary}
                onChange={(e) => setEditedSummary(e.target.value)}
                rows={4}
              />
            </CardContent>
          </Card>

          <h3 className='text-lg font-semibold'>
            Comments ({review.comments?.length ?? 0})
          </h3>
          <div className='space-y-3'>
            {review.comments?.map((c: any) => (
              <Card key={c.id}>
                <CardContent className='py-4'>
                  <div className='text-sm font-medium text-muted-foreground mb-2'>
                    {c.file_path}:{c.line_start}-{c.line_end}
                  </div>
                  <Textarea
                    value={editedComments[c.id] ?? c.body}
                    onChange={(e) =>
                      setEditedComments({
                        ...editedComments,
                        [c.id]: e.target.value,
                      })
                    }
                    rows={3}
                  />
                </CardContent>
              </Card>
            ))}
          </div>

          <ConfirmDialog
            open={showApprove}
            onOpenChange={setShowApprove}
            title='Approve & Post Review'
            desc='This will post the review to GitHub. Are you sure?'
            handleConfirm={() => {
              approveMutation.mutate()
              setShowApprove(false)
            }}
          />
        </div>
      </Main>
    </>
  )
}
