import { createFileRoute } from '@tanstack/react-router'
import SettingsCLIs from '@/features/settings/clis'

export const Route = createFileRoute('/_authenticated/settings/clis')({
  component: SettingsCLIs,
})
