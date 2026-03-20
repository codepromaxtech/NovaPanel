import api from './api';

// All CF endpoints require api_key (token or global key) + optional email
interface CFAuth { api_key: string; email?: string; }

function withAuth(auth: CFAuth, data: Record<string, unknown> = {}) {
    return { ...auth, ...data };
}

export const cloudflareService = {
    // Account
    async verify(auth: CFAuth) {
        const { data } = await api.post('/cloudflare/verify', withAuth(auth));
        return data;
    },

    // Zones
    async listZones(auth: CFAuth, page = 1) {
        const { data } = await api.post('/cloudflare/zones', withAuth(auth, { page }));
        return data;
    },
    async getZone(auth: CFAuth, zoneId: string) {
        const { data } = await api.post('/cloudflare/zones/get', withAuth(auth, { zone_id: zoneId }));
        return data;
    },

    // DNS
    async listDNS(auth: CFAuth, zoneId: string, page = 1) {
        const { data } = await api.post('/cloudflare/dns/list', withAuth(auth, { zone_id: zoneId, page }));
        return data;
    },
    async createDNS(auth: CFAuth, zoneId: string, record: { type: string; name: string; content: string; ttl?: number; proxied?: boolean; priority?: number }) {
        const { data } = await api.post('/cloudflare/dns/create', withAuth(auth, { zone_id: zoneId, ...record }));
        return data;
    },
    async updateDNS(auth: CFAuth, zoneId: string, recordId: string, record: { type: string; name: string; content: string; ttl?: number; proxied?: boolean }) {
        const { data } = await api.post('/cloudflare/dns/update', withAuth(auth, { zone_id: zoneId, record_id: recordId, ...record }));
        return data;
    },
    async deleteDNS(auth: CFAuth, zoneId: string, recordId: string) {
        const { data } = await api.post('/cloudflare/dns/delete', withAuth(auth, { zone_id: zoneId, record_id: recordId }));
        return data;
    },

    // SSL
    async getSSL(auth: CFAuth, zoneId: string) {
        const { data } = await api.post('/cloudflare/ssl/get', withAuth(auth, { zone_id: zoneId }));
        return data;
    },
    async setSSL(auth: CFAuth, zoneId: string, mode: string) {
        const { data } = await api.post('/cloudflare/ssl/set', withAuth(auth, { zone_id: zoneId, mode }));
        return data;
    },

    // Cache
    async purgeAll(auth: CFAuth, zoneId: string) {
        const { data } = await api.post('/cloudflare/cache/purge-all', withAuth(auth, { zone_id: zoneId }));
        return data;
    },
    async purgeURLs(auth: CFAuth, zoneId: string, urls: string[]) {
        const { data } = await api.post('/cloudflare/cache/purge-urls', withAuth(auth, { zone_id: zoneId, urls }));
        return data;
    },
    async setCacheTTL(auth: CFAuth, zoneId: string, ttl: number) {
        const { data } = await api.post('/cloudflare/cache/ttl', withAuth(auth, { zone_id: zoneId, ttl }));
        return data;
    },

    // Security & Dev Mode
    async setDevMode(auth: CFAuth, zoneId: string, value: string) {
        const { data } = await api.post('/cloudflare/devmode', withAuth(auth, { zone_id: zoneId, value }));
        return data;
    },
    async setSecurity(auth: CFAuth, zoneId: string, level: string) {
        const { data } = await api.post('/cloudflare/security', withAuth(auth, { zone_id: zoneId, level }));
        return data;
    },
    async listFirewall(auth: CFAuth, zoneId: string) {
        const { data } = await api.post('/cloudflare/firewall/list', withAuth(auth, { zone_id: zoneId }));
        return data;
    },

    // Analytics
    async analytics(auth: CFAuth, zoneId: string, since = -1440) {
        const { data } = await api.post('/cloudflare/analytics', withAuth(auth, { zone_id: zoneId, since }));
        return data;
    },

    // Bulk settings
    async getSettings(auth: CFAuth, zoneId: string) {
        const { data } = await api.post('/cloudflare/settings/get', withAuth(auth, { zone_id: zoneId }));
        return data;
    },
    async updateSetting(auth: CFAuth, zoneId: string, setting: string, value: string) {
        const { data } = await api.post('/cloudflare/settings/update', withAuth(auth, { zone_id: zoneId, setting, value }));
        return data;
    },
};
