import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { DataTable } from '@/components/data-table'
import { api } from '@/lib/api'

const columns = [
  { accessorKey: 'id', header: 'Review ID' },
  { accessorKey: 'pull_request_id', header: 'PR' },
  {
    accessorKey: 'overall_verdict',
    header: 'Verdict',
    cell: ({ row }: any) => <Badge>{row.original.overall_verdict}</Badge>,
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }: any) => <Badge variant='outline'>{row.original.status}</Badge>,
  },
  { accessorKey: 'executor_name', header: 'Executor' },
  {
    accessorKey: 'created_at',
    header: 'Date',
    cell: ({ row }: any) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

export default function History() {
  const { data, isLoading } = useQuery({
    queryKey: ['history'],
    queryFn: () => api.getHistory(),
    refetchInterval: 30_000,
  })

  return (
    <>
      <Header>
        <div className='me-auto' />
        <ThemeSwitch />
      </Header>
      <Main>
        <div className='space-y-4'>
          <h2 className='text-2xl font-bold tracking-tight'>Review History</h2>
          <DataTable
            columns={columns}
            data={data?.items ?? []}
            pageCount={Math.ceil(
              (data?.meta?.total ?? 0) / (data?.meta?.per_page ?? 20)
            )}
            isLoading={isLoading}
          />
        </div>
      </Main>
    </>
  )
}
