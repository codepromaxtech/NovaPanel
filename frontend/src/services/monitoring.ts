import api from './api';

export const monitoringService = {
    async getLiveMetrics(serverId: string) {
        const { data } = await api.get(`/servers/${serverId}/metrics/live`);
        return data;
    },
    async getHistoryMetrics(serverId: string, hours: number = 1) {
        const { data } = await api.get(`/servers/${serverId}/metrics/history?hours=${hours}`);
        return data;
    },
    async getServices(serverId: string) {
        const { data } = await api.get(`/servers/${serverId}/services`);
        return data;
    },
};
