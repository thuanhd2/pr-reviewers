import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { DataTable } from '@/components/data-table'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'
import { useAppStore } from '@/stores/app-store'

const columns = [
  { accessorKey: 'repo_full_name', header: 'Repo' },
  { accessorKey: 'title', header: 'Title',
    cell: ({ row }: any) => (
      <Link
        to='/prs/$prId'
        params={{ prId: String(row.original.id) }}
        className='font-medium hover:underline'
      >
        {row.original.title}
      </Link>
    ),
  },
  { accessorKey: 'author', header: 'Author' },
  { accessorKey: 'status', header: 'Status',
    cell: ({ row }: any) => <Badge variant='outline'>{row.original.status}</Badge>,
  },
  { accessorKey: 'created_at', header: 'Created',
    cell: ({ row }: any) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

export default function PRList() {
  useWebSocket()
  const { prStatusFilter } = useAppStore()

  const { data, isLoading } = useQuery({
    queryKey: ['prs', prStatusFilter],
    queryFn: () => api.getPRs(prStatusFilter ? { status: prStatusFilter } : {}),
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
          <h2 className='text-2xl font-bold tracking-tight'>Pull Requests</h2>
          <DataTable
            columns={columns}
            data={data?.items ?? []}
            pageCount={Math.ceil((data?.meta?.total ?? 0) / (data?.meta?.per_page ?? 20))}
            isLoading={isLoading}
          />
        </div>
      </Main>
    </>
  )
}
