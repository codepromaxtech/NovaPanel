import api from './api';
import type { AuthResponse } from '../types';

export const authService = {
    async login(email: string, password: string): Promise<AuthResponse> {
        const { data } = await api.post<AuthResponse>('/auth/login', { email, password });
        localStorage.setItem('novapanel_token', data.token);
        localStorage.setItem('novapanel_user', JSON.stringify(data.user));
        return data;
    },

    async register(email: string, password: string, firstName: string, lastName: string): Promise<AuthResponse> {
        const { data } = await api.post<AuthResponse>('/auth/register', {
            email,
            password,
            first_name: firstName,
            last_name: lastName,
        });
        localStorage.setItem('novapanel_token', data.token);
        localStorage.setItem('novapanel_user', JSON.stringify(data.user));
        return data;
    },

    async me() {
        const { data } = await api.get('/auth/me');
        return data;
    },

    logout() {
        localStorage.removeItem('novapanel_token');
        localStorage.removeItem('novapanel_user');
        window.location.href = '/login';
    },

    isAuthenticated(): boolean {
        return !!localStorage.getItem('novapanel_token');
    },
};
