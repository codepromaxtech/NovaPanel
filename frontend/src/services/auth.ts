import api from './api';
import type { AuthResponse } from '../types';

export const authService = {
    async login(email: string, password: string): Promise<AuthResponse> {
        const { data } = await api.post<AuthResponse>('/auth/login', { email, password });
        if (data.token) {
            localStorage.setItem('novapanel_token', data.token);
            localStorage.setItem('novapanel_user', JSON.stringify(data.user));
        }
        return data;
    },

    async register(email: string, password: string, firstName: string, lastName: string): Promise<AuthResponse> {
        const { data } = await api.post<AuthResponse>('/auth/register', {
            email, password, first_name: firstName, last_name: lastName,
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

    // ── Password Reset ──
    async forgotPassword(email: string) {
        const { data } = await api.post('/auth/forgot-password', { email });
        return data;
    },
    async resetPassword(token: string, newPassword: string) {
        const { data } = await api.post('/auth/reset-password', { token, new_password: newPassword });
        return data;
    },

    // ── TOTP / 2FA ──
    async totpSetup() {
        const { data } = await api.post('/auth/2fa/setup');
        return data as { secret: string; qr_uri: string };
    },
    async totpEnable(code: string) {
        const { data } = await api.post('/auth/2fa/enable', { code });
        return data as { backup_codes: string[] };
    },
    async totpVerify(email: string, code: string) {
        const { data } = await api.post<AuthResponse>('/auth/2fa/verify', { email, code });
        if (data.token) {
            localStorage.setItem('novapanel_token', data.token);
            localStorage.setItem('novapanel_user', JSON.stringify(data.user));
        }
        return data;
    },
    async totpDisable(password: string) {
        const { data } = await api.delete('/auth/2fa', { data: { password } });
        return data;
    },

    // ── Sessions ──
    async listSessions() {
        const { data } = await api.get('/auth/sessions');
        return data as Array<{ id: string; ip_address: string; user_agent: string; last_seen_at: string; created_at: string }>;
    },
    async revokeSession(id: string) {
        await api.delete(`/auth/sessions/${id}`);
    },
    async revokeAllOtherSessions() {
        await api.delete('/auth/sessions');
    },
};
