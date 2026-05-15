import { create } from 'zustand';

interface UIState {
  theme: 'light' | 'dark';
  sidebarOpen: boolean;
  language: 'ko' | 'en';
  toggleTheme: () => void;
  toggleSidebar: () => void;
  setLanguage: (lang: 'ko' | 'en') => void;
}

export const useUIStore = create<UIState>((set) => ({
  theme: (localStorage.getItem('theme') as 'light' | 'dark') || 'light',
  sidebarOpen: true,
  language: (localStorage.getItem('language') as 'ko' | 'en') || 'ko',

  toggleTheme: () =>
    set((state) => {
      const newTheme = state.theme === 'light' ? 'dark' : 'light';
      localStorage.setItem('theme', newTheme);
      return { theme: newTheme };
    }),

  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

  setLanguage: (lang) => {
    localStorage.setItem('language', lang);
    set({ language: lang });
  },
}));
