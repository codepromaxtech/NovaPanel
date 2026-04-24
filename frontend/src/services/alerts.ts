import api from './api';

export interface AlertRule {
    id: string;
    user_id: string;
    server_id?: string;
    name: string;
    metric: string;
    threshold: number;
    operator: string;
    duration_min: number;
    channel: string;
    destination: string;
    is_active: boolean;
    created_at: string;
}

export interface AlertIncident {
    id: string;
    rule_id: string;
    rule_name: string;
    server_id?: string;
    fired_at: string;
    resolved_at?: string;
    value: number;
    notified: boolean;
}

export const alertsService = {
    async listRules() {
        const { data } = await api.get('/alerts/rules');
        return (data.data || data || []) as AlertRule[];
    },
    async createRule(payload: Omit<AlertRule, 'id' | 'user_id' | 'is_active' | 'created_at'>) {
        const { data } = await api.post('/alerts/rules', payload);
        return data as AlertRule;
    },
    async updateRule(id: string, payload: Partial<AlertRule>) {
        const { data } = await api.put(`/alerts/rules/${id}`, payload);
        return data as AlertRule;
    },
    async deleteRule(id: string) {
        await api.delete(`/alerts/rules/${id}`);
    },
    async listIncidents() {
        const { data } = await api.get('/alerts/incidents');
        return (data.data || data || []) as AlertIncident[];
    },
};
