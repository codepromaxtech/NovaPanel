import { useState, useEffect } from 'react';
import { Shield, Plus, Trash2, X, Loader } from 'lucide-react';
import { securityService } from '../services/security';
import { useToast } from '../components/ui/ToastProvider';

interface FWRule {
    id: string; port: string; protocol: string; source_ip: string;
    action: string; direction: string; description: string; is_active: boolean;
}
interface SecEvent {
    id: string; event_type: string; source_ip: string; severity: string;
    details: string; created_at: string;
}

export default function Security() {
    const toast = useToast();
    const [rules, setRules] = useState<FWRule[]>([]);
    const [events, setEvents] = useState<SecEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddRule, setShowAddRule] = useState(false);
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const [form, setForm] = useState({ port: '', protocol: 'tcp', action: 'allow', source_ip: 'any', description: '', server_id: '' });

    const fetchData = async () => {
        try {
            const [rulesRes, eventsRes] = await Promise.all([
                securityService.listRules(),
                securityService.listEvents(),
            ]);
            setRules(Array.isArray(rulesRes) ? rulesRes : []);
            setEvents(Array.isArray(eventsRes) ? eventsRes : eventsRes?.data || []);
        } catch {
            setRules([]); setEvents([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { fetchData(); }, []);

    const handleCreate = async () => {
        if (!form.port) { setError('Port is required.'); return; }
        setCreating(true); setError('');
        try {
            await securityService.createRule({ ...form, server_id: form.server_id || 'default' });
            setShowAddRule(false);
            setForm({ port: '', protocol: 'tcp', action: 'allow', source_ip: 'any', description: '', server_id: '' });
            toast.success('Firewall rule added');
            fetchData();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to add rule');
            toast.error('Failed to add rule');
        } finally {
            setCreating(false);
        }
    };

    const handleDeleteRule = async (id: string) => {
        if (!confirm('Delete this firewall rule?')) return;
        try { await securityService.deleteRule(id); toast.success('Rule deleted'); fetchData(); }
        catch { toast.error('Failed to delete rule'); }
    };

    const severityColors: Record<string, string> = {
        info: 'text-info bg-info/10', warning: 'text-warning bg-warning/10', danger: 'text-danger bg-danger/10',
    };

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Shield className="w-7 h-7 text-nova-400" />
                        Security
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Firewall rules & threat monitoring</p>
                </div>
                <button onClick={() => setShowAddRule(true)}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Plus className="w-4 h-4" /> Add Rule
                </button>
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-20"><Loader className="w-6 h-6 text-nova-400 animate-spin" /></div>
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    <div className="lg:col-span-2 glass-card rounded-2xl overflow-hidden">
                        <div className="px-5 py-3 border-b border-white/5">
                            <h3 className="text-sm font-semibold text-white">Firewall Rules (UFW)</h3>
                        </div>
                        {rules.length === 0 ? (
                            <div className="flex flex-col items-center justify-center py-12 text-surface-200/40">
                                <Shield className="w-8 h-8 mb-2 opacity-40" />
                                <p className="text-sm">No firewall rules configured</p>
                            </div>
                        ) : (
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-white/5">
                                        <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Port</th>
                                        <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Proto</th>
                                        <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Source</th>
                                        <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Action</th>
                                        <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Description</th>
                                        <th className="py-2 px-5"></th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {rules.map(r => (
                                        <tr key={r.id} className="border-b border-white/5 hover:bg-white/[0.02] transition-colors">
                                            <td className="py-2.5 px-5 text-sm text-white font-mono">{r.port}</td>
                                            <td className="py-2.5 px-5 text-sm text-surface-200/60 uppercase">{r.protocol}</td>
                                            <td className="py-2.5 px-5 text-sm text-surface-200/50 font-mono">{r.source_ip}</td>
                                            <td className="py-2.5 px-5">
                                                <span className={`text-xs px-2 py-0.5 rounded-full ${r.action === 'allow' ? 'bg-success/10 text-success border border-success/20' : 'bg-danger/10 text-danger border border-danger/20'}`}>
                                                    {r.action}
                                                </span>
                                            </td>
                                            <td className="py-2.5 px-5 text-sm text-surface-200/40">{r.description}</td>
                                            <td className="py-2.5 px-5">
                                                <button onClick={() => handleDeleteRule(r.id)} className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/50 hover:text-danger transition-colors">
                                                    <Trash2 className="w-3.5 h-3.5" />
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        )}
                    </div>

                    <div className="glass-card rounded-2xl overflow-hidden">
                        <div className="px-5 py-3 border-b border-white/5">
                            <h3 className="text-sm font-semibold text-white">Recent Events</h3>
                        </div>
                        {events.length === 0 ? (
                            <div className="flex items-center justify-center py-12 text-surface-200/40 text-sm">No events recorded</div>
                        ) : (
                            <div className="divide-y divide-white/5">
                                {events.map(e => (
                                    <div key={e.id} className="px-5 py-3 hover:bg-white/[0.02] transition-colors">
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className={`text-xs px-2 py-0.5 rounded-full ${severityColors[e.severity] || ''}`}>{e.severity}</span>
                                            <span className="text-xs text-surface-200/50">{e.created_at ? new Date(e.created_at).toLocaleString() : ''}</span>
                                        </div>
                                        <p className="text-sm text-surface-200/70">{e.details}</p>
                                        <p className="text-xs text-surface-200/50 font-mono mt-0.5">{e.source_ip}</p>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            )}

            {showAddRule && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAddRule(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white">Add Firewall Rule</h2>
                            <button onClick={() => setShowAddRule(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        {error && <div className="mb-4 p-3 rounded-xl border border-danger/30 bg-danger/10 text-danger text-sm">{error}</div>}
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Port *</label>
                                <input value={form.port} onChange={e => setForm({ ...form, port: e.target.value })}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="e.g. 8080" />
                            </div>
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Protocol</label>
                                    <select value={form.protocol} onChange={e => setForm({ ...form, protocol: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30">
                                        <option value="tcp">TCP</option><option value="udp">UDP</option>
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Action</label>
                                    <select value={form.action} onChange={e => setForm({ ...form, action: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30">
                                        <option value="allow">Allow</option><option value="deny">Deny</option>
                                    </select>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Source IP</label>
                                <input value={form.source_ip} onChange={e => setForm({ ...form, source_ip: e.target.value })}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Description</label>
                                <input value={form.description} onChange={e => setForm({ ...form, description: e.target.value })}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="e.g. Web server port" />
                            </div>
                            <div className="flex gap-3 pt-2">
                                <button onClick={() => setShowAddRule(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-all">Cancel</button>
                                <button onClick={handleCreate} disabled={creating}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium disabled:opacity-50 flex items-center justify-center gap-2">
                                    {creating ? <Loader className="w-4 h-4 animate-spin" /> : null}
                                    {creating ? 'Adding...' : 'Add Rule'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
