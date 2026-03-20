import api from './api';

export const billingService = {
    async listPlans() {
        const { data } = await api.get('/billing/plans');
        return data;
    },
    async createPlan(payload: {
        name: string; price_cents: number; currency?: string; interval?: string;
        max_domains?: number; max_databases?: number; max_email?: number; disk_gb?: number;
    }) {
        const { data } = await api.post('/billing/plans', payload);
        return data;
    },
    async listInvoices() {
        const { data } = await api.get('/billing/invoices');
        return data;
    },
};

export const settingsService = {
    async getProfile() {
        const { data } = await api.get('/settings/profile');
        return data;
    },
    async updateProfile(payload: { name?: string; email?: string; password?: string }) {
        const { data } = await api.put('/settings/profile', payload);
        return data;
    },
};
