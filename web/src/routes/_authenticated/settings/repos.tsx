import { createFileRoute } from '@tanstack/react-router'
import SettingsRepos from '@/features/settings/repos'

export const Route = createFileRoute('/_authenticated/settings/repos')({
  component: SettingsRepos,
})
