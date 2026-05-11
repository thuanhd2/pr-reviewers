import { createFileRoute } from '@tanstack/react-router'
import PRList from '@/features/pull-requests'

export const Route = createFileRoute('/_authenticated/prs/')({
  component: PRList,
})
