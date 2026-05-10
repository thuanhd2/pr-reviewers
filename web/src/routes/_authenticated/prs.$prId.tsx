import { createFileRoute } from '@tanstack/react-router'
import PRDetail from '@/features/pull-requests/pr-detail'

export const Route = createFileRoute('/_authenticated/prs/$prId')({
  component: PRDetail,
})
