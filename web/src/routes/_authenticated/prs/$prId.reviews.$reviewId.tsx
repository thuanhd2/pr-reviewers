import { createFileRoute } from '@tanstack/react-router'
import ReviewDetail from '@/features/pull-requests/review-detail'

export const Route = createFileRoute('/_authenticated/prs/$prId/reviews/$reviewId')({
  component: ReviewDetail,
})
