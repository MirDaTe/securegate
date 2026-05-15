import { create } from 'zustand';

interface User {
  id: string;
  username: string;
  email?: string;
  role: 'super_admin' | 'admin' | 'auditor' | 'user';
  mustChangePassword: boolean;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  login: (user: User, accessToken: string, refreshToken: string) => void;
  logout: () => void;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: localStorage.getItem('access_token'),
  refreshToken: localStorage.getItem('refresh_token'),

  login: (user, accessToken, refreshToken) => {
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    set({ user, accessToken, refreshToken });
  },

  logout: () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    set({ user: null, accessToken: null, refreshToken: null });
  },

  setUser: (user) => set({ user }),
}));
