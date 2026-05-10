import {
  LayoutDashboard,
  GitPullRequest,
  History,
  Settings,
  Wrench,
  Terminal,
} from 'lucide-react'
import { type SidebarData } from '../types'

export const sidebarData: SidebarData = {
  user: {
    name: 'PR Reviewer',
    email: '',
    avatar: '',
  },
  teams: [],
  navGroups: [
    {
      title: 'General',
      items: [
        {
          title: 'Dashboard',
          url: '/',
          icon: LayoutDashboard,
        },
        {
          title: 'Pull Requests',
          url: '/prs',
          icon: GitPullRequest,
        },
        {
          title: 'History',
          url: '/history',
          icon: History,
        },
      ],
    },
    {
      title: 'Settings',
      items: [
        {
          title: 'Settings',
          icon: Settings,
          items: [
            {
              title: 'Repo Configs',
              url: '/settings/repos',
              icon: Wrench,
            },
            {
              title: 'CLI Executors',
              url: '/settings/clis',
              icon: Terminal,
            },
          ],
        },
      ],
    },
  ],
}
