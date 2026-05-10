import { create } from 'zustand'

interface AppStore {
  sidebarOpen: boolean
  toggleSidebar: () => void
  prStatusFilter: string
  setPrStatusFilter: (status: string) => void
  repoFilter: string
  setRepoFilter: (repo: string) => void
}

export const useAppStore = create<AppStore>((set) => ({
  sidebarOpen: true,
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  prStatusFilter: '',
  setPrStatusFilter: (prStatusFilter) => set({ prStatusFilter }),
  repoFilter: '',
  setRepoFilter: (repoFilter) => set({ repoFilter }),
}))
