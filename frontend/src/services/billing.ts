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

    // ── Stripe Subscription ──
    async createCheckout(planId: string) {
        const { data } = await api.post('/billing/checkout', { plan_id: planId });
        return data as { checkout_url: string };
    },
    async getSubscription() {
        const { data } = await api.get('/billing/subscription');
        return data as { plan_name: string; plan_type: string; expires_at: string | null; stripe_subscription_id: string | null };
    },
    async cancelSubscription() {
        await api.delete('/billing/subscription');
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

    // ── API Keys ──
    async listApiKeys() {
        const { data } = await api.get('/settings/api-keys');
        return (data?.data || []) as Array<{ id: string; name: string; key_prefix: string; scopes: string[]; last_used_at: string | null; expires_at: string | null; created_at: string }>;
    },
    async createApiKey(name: string, scopes: string[], expiresAt?: string) {
        const { data } = await api.post('/settings/api-keys', { name, scopes, expires_at: expiresAt });
        return data as { id: string; name: string; key_prefix: string; raw_key: string; created_at: string };
    },
    async revokeApiKey(id: string) {
        await api.delete(`/settings/api-keys/${id}`);
    },

    // ── System Settings (admin only) ──
    async getSystemSettings(): Promise<Record<string, string>> {
        const { data } = await api.get('/settings/system');
        return data;
    },
    async updateSystemSettings(payload: Record<string, string>) {
        const { data } = await api.put('/settings/system', payload);
        return data;
    },
    async testSmtp() {
        const { data } = await api.post('/settings/system/test-smtp');
        return data;
    },
};
