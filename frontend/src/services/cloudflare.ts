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

    // Tunnels
    async listTunnels(auth: CFAuth) {
        const { data } = await api.post('/cloudflare/tunnels/list', withAuth(auth));
        return data;
    },
    async createTunnel(auth: CFAuth, name: string, tunnelSecret: string) {
        const { data } = await api.post('/cloudflare/tunnels/create', withAuth(auth, { name, tunnel_secret: tunnelSecret }));
        return data;
    },
    async deleteTunnel(auth: CFAuth, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/delete', withAuth(auth, { tunnel_id: tunnelId }));
        return data;
    },
    async getTunnel(auth: CFAuth, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/get', withAuth(auth, { tunnel_id: tunnelId }));
        return data;
    },
    async getTunnelConfig(auth: CFAuth, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/config', withAuth(auth, { tunnel_id: tunnelId }));
        return data;
    },
    async updateTunnelConfig(auth: CFAuth, tunnelId: string, config: Record<string, unknown>) {
        const { data } = await api.post('/cloudflare/tunnels/config/update', withAuth(auth, { tunnel_id: tunnelId, config }));
        return data;
    },
    async getTunnelToken(auth: CFAuth, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/token', withAuth(auth, { tunnel_id: tunnelId }));
        return data;
    },
    async listTunnelConnections(auth: CFAuth, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/connections', withAuth(auth, { tunnel_id: tunnelId }));
        return data;
    },
    async createTunnelDNSRoute(auth: CFAuth, zoneId: string, hostname: string, tunnelId: string) {
        const { data } = await api.post('/cloudflare/tunnels/dns-route', withAuth(auth, { zone_id: zoneId, hostname, tunnel_id: tunnelId }));
        return data;
    },
    async installCloudflared(serverId: string) {
        const { data } = await api.post('/cloudflare/tunnels/install', { server_id: serverId });
        return data;
    },
    async runTunnel(serverId: string, tunnelToken: string, tunnelName: string) {
        const { data } = await api.post('/cloudflare/tunnels/run', { server_id: serverId, tunnel_token: tunnelToken, tunnel_name: tunnelName });
        return data;
    },
    async stopTunnel(serverId: string, tunnelName: string) {
        const { data } = await api.post('/cloudflare/tunnels/stop', { server_id: serverId, tunnel_name: tunnelName });
        return data;
    },
    async tunnelStatus(serverId: string, tunnelName: string) {
        const { data } = await api.post('/cloudflare/tunnels/status', { server_id: serverId, tunnel_name: tunnelName });
        return data;
    },
};
