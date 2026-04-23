import { useState, useEffect } from 'react';
import {
    Users, UserPlus, Mail, Shield, X, Loader, Trash2,
    CheckCircle, Clock, ChevronDown,
} from 'lucide-react';
import { teamService, TeamMember, PendingInvite } from '../services/team';
import { useToast } from '../components/ui/ToastProvider';

const ROLES = ['viewer', 'developer', 'admin'];
const SCOPE_TYPES = [
    { id: 'all', label: 'All Resources' },
    { id: 'server', label: 'Specific Server' },
    { id: 'domain', label: 'Specific Domain' },
];

const roleBadge = (role: string) => {
    const styles: Record<string, string> = {
        admin: 'bg-red-500/10 text-red-400 border-red-500/20',
        developer: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
        viewer: 'bg-surface-600/40 text-surface-200/60 border-white/10',
    };
    return (
        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${styles[role] || styles.viewer}`}>
            {role}
        </span>
    );
};

export default function TeamManagement() {
    const toast = useToast();
    const [members, setMembers] = useState<TeamMember[]>([]);
    const [invites, setInvites] = useState<PendingInvite[]>([]);
    const [loading, setLoading] = useState(true);
    const [showInvite, setShowInvite] = useState(false);
    const [inviteForm, setInviteForm] = useState({ email: '', role: 'viewer', scope_type: 'all', scope_id: '' });
    const [inviting, setInviting] = useState(false);
    const [removing, setRemoving] = useState<string | null>(null);
    const [accepting, setAccepting] = useState<string | null>(null);

    const load = async () => {
        setLoading(true);
        try {
            const [m, i] = await Promise.all([teamService.listMembers(), teamService.listInvites()]);
            setMembers(m);
            setInvites(i);
        } catch {
            setMembers([]);
            setInvites([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, []);

    const handleInvite = async () => {
        if (!inviteForm.email) return;
        setInviting(true);
        try {
            await teamService.invite(inviteForm);
            toast.success(`Invitation sent to ${inviteForm.email}`);
            setShowInvite(false);
            setInviteForm({ email: '', role: 'viewer', scope_type: 'all', scope_id: '' });
            load();
        } catch (err: any) {
            toast.error(err?.response?.data?.error || 'Failed to send invitation');
        } finally {
            setInviting(false);
        }
    };

    const handleRemove = async (id: string, email: string) => {
        if (!confirm(`Remove ${email} from your team?`)) return;
        setRemoving(id);
        try {
            await teamService.remove(id);
            toast.success('Member removed');
            load();
        } catch {
            toast.error('Failed to remove member');
        } finally {
            setRemoving(null);
        }
    };

    const handleAccept = async (id: string) => {
        setAccepting(id);
        try {
            await teamService.accept(id);
            toast.success('Invitation accepted');
            load();
        } catch {
            toast.error('Failed to accept invitation');
        } finally {
            setAccepting(null);
        }
    };

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Users className="w-7 h-7 text-nova-400" /> Team Management
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        Invite team members and manage their access scopes
                    </p>
                </div>
                <button
                    onClick={() => setShowInvite(true)}
                    className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all"
                >
                    <UserPlus className="w-4 h-4" /> Invite Member
                </button>
            </div>

            {/* Pending invites for current user */}
            {invites.length > 0 && (
                <div className="glass-card rounded-xl p-5">
                    <h2 className="text-sm font-semibold text-surface-200/60 mb-3 flex items-center gap-2">
                        <Clock className="w-4 h-4" /> Pending Invitations for You
                    </h2>
                    <div className="space-y-2">
                        {invites.map(inv => (
                            <div key={inv.id} className="flex items-center justify-between p-3 rounded-xl border border-white/5 bg-white/[0.02]">
                                <div>
                                    <p className="text-sm font-medium text-white">{inv.owner_name || inv.owner_email}</p>
                                    <p className="text-xs text-surface-200/40 mt-0.5">
                                        Invited you as {roleBadge(inv.role)} · {inv.scope_type === 'all' ? 'All resources' : inv.scope_type}
                                    </p>
                                </div>
                                <button
                                    onClick={() => handleAccept(inv.id)}
                                    disabled={accepting === inv.id}
                                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-green-500/10 text-green-400 hover:bg-green-500/20 text-xs font-medium transition-colors disabled:opacity-50"
                                >
                                    {accepting === inv.id ? <Loader className="w-3 h-3 animate-spin" /> : <CheckCircle className="w-3 h-3" />}
                                    Accept
                                </button>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Members table */}
            <div className="glass-card rounded-xl overflow-hidden">
                <div className="px-5 py-4 border-b border-white/5 flex items-center gap-2">
                    <Shield className="w-4 h-4 text-nova-400" />
                    <h2 className="text-sm font-semibold text-white">Team Members</h2>
                    <span className="ml-auto text-xs text-surface-200/40">{members.length} member{members.length !== 1 ? 's' : ''}</span>
                </div>
                {loading ? (
                    <div className="flex items-center justify-center py-16">
                        <Loader className="w-6 h-6 text-nova-400 animate-spin" />
                    </div>
                ) : members.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-16 text-surface-200/40">
                        <Users className="w-12 h-12 mb-3 opacity-30" />
                        <p className="text-sm">No team members yet</p>
                        <p className="text-xs mt-1">Invite someone to get started</p>
                    </div>
                ) : (
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-white/5">
                                <th className="text-left px-5 py-3 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Member</th>
                                <th className="text-left px-5 py-3 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Role</th>
                                <th className="text-left px-5 py-3 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Scope</th>
                                <th className="text-left px-5 py-3 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Status</th>
                                <th className="text-right px-5 py-3 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {members.map(m => (
                                <tr key={m.id} className="border-b border-white/[0.04] hover:bg-white/[0.02] transition-colors">
                                    <td className="px-5 py-3.5">
                                        <div className="flex items-center gap-3">
                                            <div className="w-8 h-8 rounded-full bg-nova-500/20 flex items-center justify-center text-nova-400 text-xs font-bold">
                                                {(m.first_name?.[0] || m.email[0]).toUpperCase()}
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-white">
                                                    {m.first_name || m.last_name ? `${m.first_name} ${m.last_name}`.trim() : m.email}
                                                </p>
                                                <p className="text-xs text-surface-200/40 flex items-center gap-1">
                                                    <Mail className="w-3 h-3" /> {m.email}
                                                </p>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5">{roleBadge(m.role)}</td>
                                    <td className="px-5 py-3.5">
                                        <span className="text-xs text-surface-200/50 capitalize">
                                            {m.scope_type === 'all' ? 'All Resources' : m.scope_type}
                                        </span>
                                    </td>
                                    <td className="px-5 py-3.5">
                                        {m.status === 'active' ? (
                                            <span className="inline-flex items-center gap-1 text-xs text-green-400">
                                                <CheckCircle className="w-3 h-3" /> Active
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center gap-1 text-xs text-yellow-400">
                                                <Clock className="w-3 h-3" /> Pending
                                            </span>
                                        )}
                                    </td>
                                    <td className="px-5 py-3.5 text-right">
                                        <button
                                            onClick={() => handleRemove(m.id, m.email)}
                                            disabled={removing === m.id}
                                            className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/50 hover:text-danger transition-colors disabled:opacity-50"
                                        >
                                            {removing === m.id ? <Loader className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>

            {/* Invite Modal */}
            {showInvite && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowInvite(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white flex items-center gap-2">
                                <UserPlus className="w-5 h-5 text-nova-400" /> Invite Team Member
                            </h2>
                            <button onClick={() => setShowInvite(false)} className="text-surface-200/40 hover:text-white">
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Email Address</label>
                                <input
                                    type="email"
                                    value={inviteForm.email}
                                    onChange={e => setInviteForm(p => ({ ...p, email: e.target.value }))}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="colleague@company.com"
                                />
                                <p className="text-xs text-surface-200/50 mt-1">Must be a registered NovaPanel user</p>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Role</label>
                                <div className="relative">
                                    <select
                                        value={inviteForm.role}
                                        onChange={e => setInviteForm(p => ({ ...p, role: e.target.value }))}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent appearance-none"
                                    >
                                        {ROLES.map(r => (
                                            <option key={r} value={r} className="bg-surface-800 capitalize">{r}</option>
                                        ))}
                                    </select>
                                    <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/40 pointer-events-none" />
                                </div>
                                <div className="mt-2 text-xs text-surface-200/40 space-y-0.5">
                                    <p><span className="text-surface-200/70">viewer</span> — read-only access</p>
                                    <p><span className="text-surface-200/70">developer</span> — deploy & manage apps</p>
                                    <p><span className="text-surface-200/70">admin</span> — full access</p>
                                </div>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Access Scope</label>
                                <div className="relative">
                                    <select
                                        value={inviteForm.scope_type}
                                        onChange={e => setInviteForm(p => ({ ...p, scope_type: e.target.value }))}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent appearance-none"
                                    >
                                        {SCOPE_TYPES.map(s => (
                                            <option key={s.id} value={s.id} className="bg-surface-800">{s.label}</option>
                                        ))}
                                    </select>
                                    <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/40 pointer-events-none" />
                                </div>
                            </div>

                            {inviteForm.scope_type !== 'all' && (
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">
                                        {inviteForm.scope_type === 'server' ? 'Server' : 'Domain'} ID
                                    </label>
                                    <input
                                        value={inviteForm.scope_id}
                                        onChange={e => setInviteForm(p => ({ ...p, scope_id: e.target.value }))}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20 font-mono"
                                        placeholder="UUID of the resource"
                                    />
                                </div>
                            )}

                            <div className="flex gap-3 pt-2">
                                <button
                                    onClick={() => setShowInvite(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={handleInvite}
                                    disabled={inviting || !inviteForm.email}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                                >
                                    {inviting ? <Loader className="w-4 h-4 animate-spin" /> : <UserPlus className="w-4 h-4" />}
                                    {inviting ? 'Sending...' : 'Send Invite'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
