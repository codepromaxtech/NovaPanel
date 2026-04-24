import { useState, useEffect } from 'react';
import { Users, Plus, X, Trash2, Edit2, BarChart2, Eye, EyeOff, RefreshCw, Globe, Database, Mail, HardDrive } from 'lucide-react';
import { resellerService, ResellerClient, ClientUsage } from '../services/reseller';
import { useToast } from '../components/ui/ToastProvider';

const inputCls = 'w-full px-3.5 py-2.5 rounded-xl bg-surface-700/50 border border-surface-600/50 text-white text-sm placeholder:text-surface-400/50 focus:outline-none focus:border-nova-500/70 focus:ring-1 focus:ring-nova-500/30 transition-all';

const statusBadge = (s: string) => {
    const map: Record<string, string> = {
        active:    'bg-success/10 text-success',
        suspended: 'bg-warning/10 text-warning',
        deleted:   'bg-danger/10 text-danger',
    };
    return map[s] || 'bg-surface-600/50 text-surface-400';
};

function UsageBar({ used, max, label }: { used: number; max: number; label: string }) {
    const pct = max > 0 ? Math.min((used / max) * 100, 100) : 0;
    const color = pct >= 90 ? 'bg-danger' : pct >= 70 ? 'bg-warning' : 'bg-nova-500';
    return (
        <div className="space-y-1">
            <div className="flex justify-between text-[10px] text-surface-400">
                <span>{label}</span><span>{used}/{max}</span>
            </div>
            <div className="h-1.5 rounded-full bg-surface-700">
                <div className={`h-full rounded-full ${color} transition-all`} style={{ width: `${pct}%` }} />
            </div>
        </div>
    );
}

export default function Reseller() {
    const toast = useToast();
    const [clients, setClients] = useState<ResellerClient[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [editClient, setEditClient] = useState<ResellerClient | null>(null);
    const [usageMap, setUsageMap] = useState<Record<string, ClientUsage>>({});
    const [showPass, setShowPass] = useState(false);
    const [saving, setSaving] = useState(false);
    const [form, setForm] = useState({
        email: '', password: '', first_name: '', last_name: '',
        max_domains: 5, max_databases: 2, max_email: 10, disk_gb: 5,
    });
    const [quotaForm, setQuotaForm] = useState({ max_domains: 5, max_databases: 2, max_email: 10, disk_gb: 5 });
    const [formErr, setFormErr] = useState('');

    const load = async () => {
        setLoading(true);
        try { setClients(await resellerService.listClients()); }
        catch { setClients([]); }
        finally { setLoading(false); }
    };

    useEffect(() => { load(); }, []);

    const loadUsage = async (clientId: string) => {
        try {
            const usage = await resellerService.getClientUsage(clientId);
            setUsageMap(m => ({ ...m, [clientId]: usage }));
        } catch {}
    };

    const handleCreate = async () => {
        if (!form.email || !form.password || !form.first_name) { setFormErr('Email, password and first name are required.'); return; }
        setSaving(true); setFormErr('');
        try {
            await resellerService.allocateClient(form);
            toast.success(`Client ${form.email} created`);
            setShowCreate(false);
            setForm({ email: '', password: '', first_name: '', last_name: '', max_domains: 5, max_databases: 2, max_email: 10, disk_gb: 5 });
            load();
        } catch (err: any) {
            setFormErr(err.response?.data?.error || 'Failed to create client');
        } finally { setSaving(false); }
    };

    const handleUpdateQuota = async () => {
        if (!editClient) return;
        setSaving(true);
        try {
            await resellerService.updateQuota(editClient.id, quotaForm);
            toast.success('Quota updated');
            setEditClient(null);
            load();
        } catch (err: any) {
            toast.error(err.response?.data?.error || 'Failed to update quota');
        } finally { setSaving(false); }
    };

    const handleSuspend = async (c: ResellerClient) => {
        if (!confirm(`Suspend ${c.email}?`)) return;
        try { await resellerService.suspendClient(c.id); toast.success('Client suspended'); load(); }
        catch { toast.error('Failed to suspend client'); }
    };

    const openEdit = (c: ResellerClient) => {
        setEditClient(c);
        setQuotaForm({ max_domains: c.quota.max_domains, max_databases: c.quota.max_databases, max_email: c.quota.max_email, disk_gb: c.quota.disk_gb });
        loadUsage(c.id);
    };

    const quotaField = (label: string, key: keyof typeof quotaForm, icon: any) => {
        const Icon = icon;
        return (
            <div>
                <label className="block text-xs font-medium text-surface-300 mb-1.5 flex items-center gap-1.5">
                    <Icon className="w-3.5 h-3.5 text-surface-400" />{label}
                </label>
                <input type="number" min={0} value={quotaForm[key]}
                    onChange={e => setQuotaForm(f => ({ ...f, [key]: Number(e.target.value) }))}
                    className={inputCls} />
            </div>
        );
    };

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Users className="w-7 h-7 text-nova-400" /> Reseller Clients
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        Manage your hosting clients — {clients.length} client{clients.length !== 1 ? 's' : ''}
                    </p>
                </div>
                <button onClick={() => setShowCreate(true)}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Plus className="w-4 h-4" /> Add Client
                </button>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-3 gap-4">
                {[
                    { label: 'Total Clients', value: clients.length, color: 'text-nova-400' },
                    { label: 'Active', value: clients.filter(c => c.status === 'active').length, color: 'text-success' },
                    { label: 'Suspended', value: clients.filter(c => c.status === 'suspended').length, color: 'text-warning' },
                ].map(s => (
                    <div key={s.label} className="rounded-xl border border-surface-700/50 bg-surface-800/40 p-4 text-center">
                        <p className={`text-2xl font-bold ${s.color}`}>{s.value}</p>
                        <p className="text-xs text-surface-400 mt-1">{s.label}</p>
                    </div>
                ))}
            </div>

            {/* Client table */}
            <div className="rounded-2xl border border-surface-700/50 bg-surface-800/40 backdrop-blur-sm overflow-hidden">
                {loading ? (
                    <div className="flex items-center justify-center py-16 text-surface-400">
                        <div className="w-6 h-6 border-2 border-surface-500 border-t-nova-400 rounded-full animate-spin mr-3" /> Loading…
                    </div>
                ) : clients.length === 0 ? (
                    <div className="text-center py-16 text-surface-400">
                        <Users className="w-12 h-12 mx-auto mb-4 opacity-30" />
                        <p className="font-medium">No clients yet</p>
                        <p className="text-sm mt-1 opacity-60">Add your first reseller client to get started</p>
                    </div>
                ) : (
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b border-surface-700/50 text-surface-400 text-xs uppercase tracking-wider">
                                <th className="text-left px-5 py-3 font-medium">Client</th>
                                <th className="text-left px-5 py-3 font-medium">Status</th>
                                <th className="text-left px-5 py-3 font-medium">Quota Limits</th>
                                <th className="text-left px-5 py-3 font-medium">Joined</th>
                                <th className="text-right px-5 py-3 font-medium">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-surface-700/30">
                            {clients.map(c => (
                                <tr key={c.id} className="hover:bg-surface-700/20 transition-colors">
                                    <td className="px-5 py-3.5">
                                        <p className="font-medium text-white">{c.first_name} {c.last_name}</p>
                                        <p className="text-xs text-surface-400 mt-0.5">{c.email}</p>
                                    </td>
                                    <td className="px-5 py-3.5">
                                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium capitalize ${statusBadge(c.status)}`}>
                                            <span className="w-1.5 h-1.5 rounded-full bg-current" />{c.status}
                                        </span>
                                    </td>
                                    <td className="px-5 py-3.5">
                                        <div className="flex items-center gap-4 text-xs text-surface-400">
                                            <span className="flex items-center gap-1"><Globe className="w-3 h-3" />{c.quota.max_domains} domains</span>
                                            <span className="flex items-center gap-1"><Database className="w-3 h-3" />{c.quota.max_databases} DB</span>
                                            <span className="flex items-center gap-1"><Mail className="w-3 h-3" />{c.quota.max_email} email</span>
                                            <span className="flex items-center gap-1"><HardDrive className="w-3 h-3" />{c.quota.disk_gb} GB</span>
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5 text-surface-400 text-xs">
                                        {new Date(c.created_at).toLocaleDateString()}
                                    </td>
                                    <td className="px-5 py-3.5 text-right">
                                        <div className="flex items-center justify-end gap-1">
                                            <button onClick={() => openEdit(c)}
                                                className="p-1.5 rounded-lg text-surface-400 hover:text-nova-400 hover:bg-nova-400/10 transition-colors" title="Edit quota">
                                                <Edit2 className="w-4 h-4" />
                                            </button>
                                            {c.status === 'active' && (
                                                <button onClick={() => handleSuspend(c)}
                                                    className="p-1.5 rounded-lg text-surface-400 hover:text-warning hover:bg-warning/10 transition-colors" title="Suspend">
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            )}
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>

            {/* Create client modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                    <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)} />
                    <div className="relative z-10 w-full max-w-lg rounded-2xl bg-surface-800 border border-surface-700/50 shadow-2xl p-6 max-h-[90vh] overflow-y-auto">
                        <div className="flex items-center justify-between mb-5">
                            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                                <Users className="w-5 h-5 text-nova-400" /> New Client
                            </h2>
                            <button onClick={() => setShowCreate(false)} className="text-surface-400 hover:text-white transition-colors"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-4">
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">First Name</label>
                                    <input value={form.first_name} onChange={e => setForm(f => ({ ...f, first_name: e.target.value }))}
                                        placeholder="Jane" className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">Last Name</label>
                                    <input value={form.last_name} onChange={e => setForm(f => ({ ...f, last_name: e.target.value }))}
                                        placeholder="Doe" className={inputCls} />
                                </div>
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Email</label>
                                <input type="email" value={form.email} onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                                    placeholder="client@example.com" className={inputCls} />
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Password</label>
                                <div className="relative">
                                    <input type={showPass ? 'text' : 'password'} value={form.password}
                                        onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                                        placeholder="Min 8 characters" className={inputCls + ' pr-10'} />
                                    <button type="button" onClick={() => setShowPass(p => !p)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 text-surface-400 hover:text-white">
                                        {showPass ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </button>
                                </div>
                            </div>
                            <p className="text-xs font-medium text-surface-300 pt-2 border-t border-surface-700/50">Resource Quotas</p>
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5 flex items-center gap-1"><Globe className="w-3 h-3" /> Max Domains</label>
                                    <input type="number" min={1} value={form.max_domains}
                                        onChange={e => setForm(f => ({ ...f, max_domains: Number(e.target.value) }))} className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5 flex items-center gap-1"><Database className="w-3 h-3" /> Max Databases</label>
                                    <input type="number" min={1} value={form.max_databases}
                                        onChange={e => setForm(f => ({ ...f, max_databases: Number(e.target.value) }))} className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5 flex items-center gap-1"><Mail className="w-3 h-3" /> Max Email Accounts</label>
                                    <input type="number" min={1} value={form.max_email}
                                        onChange={e => setForm(f => ({ ...f, max_email: Number(e.target.value) }))} className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5 flex items-center gap-1"><HardDrive className="w-3 h-3" /> Disk GB</label>
                                    <input type="number" min={1} value={form.disk_gb}
                                        onChange={e => setForm(f => ({ ...f, disk_gb: Number(e.target.value) }))} className={inputCls} />
                                </div>
                            </div>
                            {formErr && <p className="text-danger text-xs flex items-center gap-1.5"><span className="w-1 h-1 rounded-full bg-danger" /> {formErr}</p>}
                            <button onClick={handleCreate} disabled={saving}
                                className="w-full py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium disabled:opacity-60 flex items-center justify-center gap-2 hover:shadow-lg transition-all">
                                {saving ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Creating…</> : <><Plus className="w-4 h-4" /> Create Client</>}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Edit quota modal */}
            {editClient && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                    <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setEditClient(null)} />
                    <div className="relative z-10 w-full max-w-md rounded-2xl bg-surface-800 border border-surface-700/50 shadow-2xl p-6">
                        <div className="flex items-center justify-between mb-5">
                            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                                <BarChart2 className="w-5 h-5 text-nova-400" /> Edit Quota — {editClient.email}
                            </h2>
                            <button onClick={() => setEditClient(null)} className="text-surface-400 hover:text-white transition-colors"><X className="w-5 h-5" /></button>
                        </div>

                        {/* Usage bars */}
                        {usageMap[editClient.id] && (
                            <div className="mb-5 p-4 rounded-xl bg-surface-900/50 space-y-3">
                                <p className="text-xs font-medium text-surface-300 flex items-center gap-1.5">
                                    <BarChart2 className="w-3.5 h-3.5" /> Current Usage
                                </p>
                                <UsageBar used={usageMap[editClient.id].domains} max={editClient.quota.max_domains} label="Domains" />
                                <UsageBar used={usageMap[editClient.id].databases} max={editClient.quota.max_databases} label="Databases" />
                                <UsageBar used={usageMap[editClient.id].email_accounts} max={editClient.quota.max_email} label="Email Accounts" />
                            </div>
                        )}
                        {!usageMap[editClient.id] && (
                            <button onClick={() => loadUsage(editClient.id)}
                                className="w-full mb-4 py-2 rounded-lg border border-surface-700/50 text-surface-400 text-xs hover:text-white hover:bg-surface-700/30 transition-colors flex items-center justify-center gap-1.5">
                                <RefreshCw className="w-3.5 h-3.5" /> Load usage stats
                            </button>
                        )}

                        <div className="grid grid-cols-2 gap-3 space-y-0">
                            {quotaField('Max Domains', 'max_domains', Globe)}
                            {quotaField('Max Databases', 'max_databases', Database)}
                            {quotaField('Max Email', 'max_email', Mail)}
                            {quotaField('Disk GB', 'disk_gb', HardDrive)}
                        </div>
                        <button onClick={handleUpdateQuota} disabled={saving}
                            className="w-full mt-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium disabled:opacity-60 flex items-center justify-center gap-2 hover:shadow-lg transition-all">
                            {saving ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Saving…</> : 'Save Quota'}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
