import { useQuery } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { api } from '@/lib/api'

export default function SettingsCLIs() {
  const { data: clis } = useQuery({
    queryKey: ['cli-configs'],
    queryFn: api.getCLIConfigs,
  })

  return (
    <>
      <Header>
        <div className='me-auto' />
        <ThemeSwitch />
      </Header>
      <Main>
        <div className='space-y-4'>
          <h2 className='text-2xl font-bold tracking-tight'>
            CLI Executors
          </h2>
          <div className='space-y-2'>
            {(Array.isArray(clis) ? clis : []).map((name: string) => (
              <Card key={name}>
                <CardContent className='flex items-center justify-between py-4'>
                  <span className='font-medium'>{name}</span>
                  <Badge variant='outline'>Available</Badge>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </Main>
    </>
  )
}
