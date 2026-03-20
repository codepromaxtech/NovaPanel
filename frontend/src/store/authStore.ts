import { create } from 'zustand';
import type { User } from '../types';

interface AuthState {
    user: User | null;
    token: string | null;
    isAuthenticated: boolean;
    setAuth: (user: User, token: string) => void;
    clearAuth: () => void;
    loadFromStorage: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
    user: null,
    token: null,
    isAuthenticated: false,

    setAuth: (user, token) => {
        localStorage.setItem('novapanel_token', token);
        localStorage.setItem('novapanel_user', JSON.stringify(user));
        set({ user, token, isAuthenticated: true });
    },

    clearAuth: () => {
        localStorage.removeItem('novapanel_token');
        localStorage.removeItem('novapanel_user');
        set({ user: null, token: null, isAuthenticated: false });
    },

    loadFromStorage: () => {
        const token = localStorage.getItem('novapanel_token');
        const userStr = localStorage.getItem('novapanel_user');
        if (token && userStr) {
            try {
                const user = JSON.parse(userStr) as User;
                set({ user, token, isAuthenticated: true });
            } catch {
                set({ user: null, token: null, isAuthenticated: false });
            }
        }
    },
}));
