import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ThemeSwitch } from '@/components/theme-switch'
import { SelectDropdown } from '@/components/select-dropdown'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { api } from '@/lib/api'

const cliOptions = [
  { label: 'Claude Code', value: 'claude-code' },
  { label: 'Codex', value: 'codex' },
]

export default function SettingsRepos() {
  const queryClient = useQueryClient()
  const { data: configs } = useQuery({
    queryKey: ['repo-configs'],
    queryFn: api.getRepoConfigs,
  })
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({
    repo_full_name: '',
    local_path: '',
    cli: 'claude-code',
    extra_rules: '',
    remote_name: 'origin',
  })
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const createMutation = useMutation({
    mutationFn: (data: any) => api.createRepoConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repo-configs'] })
      setShowAdd(false)
      setForm({ repo_full_name: '', local_path: '', cli: 'claude-code', extra_rules: '', remote_name: 'origin' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.deleteRepoConfig(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repo-configs'] })
      setDeleteId(null)
    },
  })

  const testMutation = useMutation({
    mutationFn: (id: number) => api.testRepoConfig(id),
  })

  return (
    <>
      <Header>
        <div className='me-auto' />
        <ThemeSwitch />
      </Header>
      <Main>
        <div className='space-y-4'>
          <div className='flex items-center justify-between'>
            <h2 className='text-2xl font-bold tracking-tight'>
              Repo Configurations
            </h2>
            <Button onClick={() => setShowAdd(true)}>Add Repo</Button>
          </div>

          {showAdd && (
            <Card>
              <CardHeader>
                <CardTitle>Add Repository</CardTitle>
              </CardHeader>
              <CardContent className='space-y-3'>
                <Input
                  placeholder='owner/repo'
                  value={form.repo_full_name}
                  onChange={(e) =>
                    setForm({ ...form, repo_full_name: e.target.value })
                  }
                />
                <Input
                  placeholder='/path/to/local/repo'
                  value={form.local_path}
                  onChange={(e) =>
                    setForm({ ...form, local_path: e.target.value })
                  }
                />
                <Input
                  placeholder='Extra rules (optional)'
                  value={form.extra_rules}
                  onChange={(e) =>
                    setForm({ ...form, extra_rules: e.target.value })
                  }
                />
                <Input
                  placeholder='origin'
                  value={form.remote_name}
                  onChange={(e) =>
                    setForm({ ...form, remote_name: e.target.value })
                  }
                />
                <SelectDropdown
                  defaultValue={form.cli}
                  onValueChange={(v) => setForm({ ...form, cli: v })}
                  items={cliOptions}
                />
                <div className='flex gap-2'>
                  <Button
                    onClick={() =>
                      createMutation.mutate({
                        ...form,
                        active: true,
                        extra_rules: form.extra_rules || null,
                      })
                    }
                  >
                    Save
                  </Button>
                  <Button variant='outline' onClick={() => setShowAdd(false)}>
                    Cancel
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          <div className='space-y-2'>
            {(Array.isArray(configs) ? configs : []).map((cfg: any) => (
              <Card key={cfg.id}>
                <CardContent className='flex items-center justify-between py-4'>
                  <div>
                    <span className='font-medium'>{cfg.repo_full_name}</span>
                    <span className='text-sm text-muted-foreground ml-4'>
                      {cfg.local_path}
                    </span>
                    <span className='text-sm text-muted-foreground ml-2'>
                      [{cfg.remote_name || 'origin'}]
                    </span>
                    <span className='text-sm text-muted-foreground ml-2'>
                      [{cfg.cli}]
                    </span>
                  </div>
                  <div className='flex gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => testMutation.mutate(cfg.id)}
                    >
                      Test
                    </Button>
                    <Button
                      variant='destructive'
                      size='sm'
                      onClick={() => setDeleteId(cfg.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <ConfirmDialog
            open={deleteId !== null}
            onOpenChange={() => setDeleteId(null)}
            title='Delete Repo Config'
            desc='Are you sure you want to delete this configuration?'
            handleConfirm={() => deleteMutation.mutate(deleteId!)}
          />
        </div>
      </Main>
    </>
  )
}
