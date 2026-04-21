import api from './api';

export interface TeamMember {
    id: string;
    member_id: string;
    email: string;
    first_name: string;
    last_name: string;
    role: string;
    scope_type: string;
    scope_id: string | null;
    invited_at: string;
    accepted_at: string | null;
    status: 'active' | 'pending';
}

export interface PendingInvite {
    id: string;
    owner_id: string;
    owner_email: string;
    owner_name: string;
    role: string;
    scope_type: string;
    invited_at: string;
}

export const teamService = {
    async listMembers(): Promise<TeamMember[]> {
        const { data } = await api.get('/team/members');
        return data.data || data || [];
    },
    async listInvites(): Promise<PendingInvite[]> {
        const { data } = await api.get('/team/invites');
        return data.data || data || [];
    },
    async invite(payload: { email: string; role: string; scope_type?: string; scope_id?: string }) {
        const { data } = await api.post('/team/invite', payload);
        return data;
    },
    async accept(inviteId: string) {
        const { data } = await api.post(`/team/accept/${inviteId}`);
        return data;
    },
    async remove(memberId: string) {
        await api.delete(`/team/members/${memberId}`);
    },
};
