import { useState, useEffect } from 'react';
import { Bell, Plus, Trash2, X, Edit2, CheckCircle, AlertTriangle, Clock } from 'lucide-react';
import { alertsService, AlertRule, AlertIncident } from '../services/alerts';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface Server { id: string; name: string; ip_address: string; }

const METRICS = [
    { value: 'cpu_usage',      label: 'CPU Usage (%)'       },
    { value: 'memory_usage',   label: 'Memory Usage (%)'    },
    { value: 'disk_usage',     label: 'Disk Usage (%)'      },
    { value: 'load_average',   label: 'Load Average'        },
    { value: 'network_in',     label: 'Network In (MB/s)'   },
    { value: 'network_out',    label: 'Network Out (MB/s)'  },
];

const OPERATORS = ['>', '<', '>=', '<=', '='];
const CHANNELS  = ['email', 'webhook', 'slack'];

const inputCls = 'w-full px-3.5 py-2.5 rounded-xl bg-surface-700/50 border border-surface-600/50 text-white text-sm placeholder:text-surface-400/50 focus:outline-none focus:border-nova-500/70 focus:ring-1 focus:ring-nova-500/30 transition-all';

function formatDate(d?: string) {
    if (!d) return '—';
    return new Date(d).toLocaleString();
}

export default function Alerts() {
    const toast = useToast();
    const [tab, setTab] = useState<'rules' | 'incidents'>('rules');
    const [rules, setRules] = useState<AlertRule[]>([]);
    const [incidents, setIncidents] = useState<AlertIncident[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [editRule, setEditRule] = useState<AlertRule | null>(null);
    const [saving, setSaving] = useState(false);
    const [form, setForm] = useState({
        name: '', server_id: '', metric: 'cpu_usage', threshold: 80,
        operator: '>', duration_min: 5, channel: 'email', destination: '',
    });
    const [formErr, setFormErr] = useState('');

    const load = async () => {
        try {
            const [r, i, s] = await Promise.all([
                alertsService.listRules(),
                alertsService.listIncidents(),
                serverService.list(),
            ]);
            setRules(Array.isArray(r) ? r : []);
            setIncidents(Array.isArray(i) ? i : []);
            const srvList = Array.isArray(s) ? s : (s?.data || []);
            setServers(srvList);
        } catch {
            setRules([]); setIncidents([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, []);

    const serverName = (id?: string) => id ? (servers.find(s => s.id === id)?.name || id.slice(0, 8)) : 'All Servers';

    const openCreate = () => {
        setEditRule(null);
        setForm({ name: '', server_id: '', metric: 'cpu_usage', threshold: 80, operator: '>', duration_min: 5, channel: 'email', destination: '' });
        setFormErr('');
        setShowCreate(true);
    };

    const openEdit = (rule: AlertRule) => {
        setEditRule(rule);
        setForm({
            name: rule.name, server_id: rule.server_id || '',
            metric: rule.metric, threshold: rule.threshold,
            operator: rule.operator, duration_min: rule.duration_min,
            channel: rule.channel, destination: rule.destination,
        });
        setFormErr('');
        setShowCreate(true);
    };

    const handleSave = async () => {
        if (!form.name) { setFormErr('Rule name is required.'); return; }
        if (form.channel === 'email' && !form.destination) { setFormErr('Email destination is required.'); return; }
        if ((form.channel === 'webhook' || form.channel === 'slack') && !form.destination) { setFormErr('Webhook URL is required.'); return; }
        setSaving(true); setFormErr('');
        try {
            const payload = {
                ...form,
                server_id: form.server_id || undefined,
                threshold: Number(form.threshold),
                duration_min: Number(form.duration_min),
            };
            if (editRule) {
                await alertsService.updateRule(editRule.id, payload);
                toast.success('Alert rule updated');
            } else {
                await alertsService.createRule(payload as any);
                toast.success('Alert rule created');
            }
            setShowCreate(false);
            load();
        } catch (err: any) {
            setFormErr(err.response?.data?.error || 'Failed to save alert rule');
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Delete this alert rule?')) return;
        try {
            await alertsService.deleteRule(id);
            toast.success('Alert rule deleted');
            load();
        } catch {
            toast.error('Failed to delete alert rule');
        }
    };

    const toggleActive = async (rule: AlertRule) => {
        try {
            await alertsService.updateRule(rule.id, { is_active: !rule.is_active });
            load();
        } catch {
            toast.error('Failed to update rule');
        }
    };

    const openIncidents = incidents.filter(i => !i.resolved_at);
    const resolvedIncidents = incidents.filter(i => i.resolved_at);

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Bell className="w-7 h-7 text-nova-400" />
                        Alerts
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        {openIncidents.length > 0
                            ? <span className="text-warning">{openIncidents.length} active incident{openIncidents.length > 1 ? 's' : ''}</span>
                            : 'Monitor server metrics and get notified on threshold breaches'}
                    </p>
                </div>
                <button
                    onClick={openCreate}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all"
                >
                    <Plus className="w-4 h-4" /> New Rule
                </button>
            </div>

            {/* Active Incidents banner */}
            {openIncidents.length > 0 && (
                <div className="rounded-xl border border-warning/30 bg-warning/5 p-4 space-y-2">
                    <div className="flex items-center gap-2 text-warning font-medium text-sm">
                        <AlertTriangle className="w-4 h-4" />
                        Active Incidents
                    </div>
                    {openIncidents.map(inc => (
                        <div key={inc.id} className="flex items-center justify-between text-sm text-surface-300 pl-6">
                            <span>{inc.rule_name} — value: <strong className="text-warning">{inc.value?.toFixed(1)}</strong></span>
                            <span className="text-surface-500 text-xs">{formatDate(inc.fired_at)}</span>
                        </div>
                    ))}
                </div>
            )}

            {/* Tabs */}
            <div className="flex gap-1 p-1 rounded-xl bg-surface-800/60 border border-surface-700/50 w-fit">
                {(['rules', 'incidents'] as const).map(t => (
                    <button
                        key={t}
                        onClick={() => setTab(t)}
                        className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-all capitalize ${tab === t ? 'bg-nova-600 text-white' : 'text-surface-400 hover:text-white'}`}
                    >
                        {t === 'rules' ? `Rules (${rules.length})` : `Incidents (${incidents.length})`}
                    </button>
                ))}
            </div>

            {/* Rules tab */}
            {tab === 'rules' && (
                <div className="rounded-2xl border border-surface-700/50 bg-surface-800/40 backdrop-blur-sm overflow-hidden">
                    {loading ? (
                        <div className="flex items-center justify-center py-16 text-surface-400">
                            <div className="w-6 h-6 border-2 border-surface-500 border-t-nova-400 rounded-full animate-spin mr-3" />
                            Loading…
                        </div>
                    ) : rules.length === 0 ? (
                        <div className="text-center py-16 text-surface-400">
                            <Bell className="w-12 h-12 mx-auto mb-4 opacity-30" />
                            <p className="font-medium">No alert rules</p>
                            <p className="text-sm mt-1 opacity-60">Create a rule to be notified when a metric crosses a threshold</p>
                        </div>
                    ) : (
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="border-b border-surface-700/50 text-surface-400 text-xs uppercase tracking-wider">
                                    <th className="text-left px-5 py-3 font-medium">Rule</th>
                                    <th className="text-left px-5 py-3 font-medium">Condition</th>
                                    <th className="text-left px-5 py-3 font-medium">Server</th>
                                    <th className="text-left px-5 py-3 font-medium">Notify</th>
                                    <th className="text-left px-5 py-3 font-medium">Status</th>
                                    <th className="text-right px-5 py-3 font-medium">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-surface-700/30">
                                {rules.map(rule => (
                                    <tr key={rule.id} className="hover:bg-surface-700/20 transition-colors">
                                        <td className="px-5 py-3.5 text-white font-medium">{rule.name}</td>
                                        <td className="px-5 py-3.5 text-surface-300 font-mono text-xs">
                                            {METRICS.find(m => m.value === rule.metric)?.label || rule.metric}{' '}
                                            <span className="text-nova-400">{rule.operator} {rule.threshold}</span>
                                            {' '}for <span className="text-nova-400">{rule.duration_min}m</span>
                                        </td>
                                        <td className="px-5 py-3.5 text-surface-300 text-xs">{serverName(rule.server_id)}</td>
                                        <td className="px-5 py-3.5 text-surface-300 text-xs capitalize">{rule.channel}</td>
                                        <td className="px-5 py-3.5">
                                            <button onClick={() => toggleActive(rule)}>
                                                <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium cursor-pointer transition-colors ${rule.is_active ? 'bg-success/10 text-success hover:bg-success/20' : 'bg-surface-600/50 text-surface-400 hover:bg-surface-600'}`}>
                                                    <span className={`w-1.5 h-1.5 rounded-full ${rule.is_active ? 'bg-success' : 'bg-surface-400'}`} />
                                                    {rule.is_active ? 'Active' : 'Paused'}
                                                </span>
                                            </button>
                                        </td>
                                        <td className="px-5 py-3.5 text-right flex items-center justify-end gap-1">
                                            <button onClick={() => openEdit(rule)} className="p-1.5 rounded-lg text-surface-400 hover:text-nova-400 hover:bg-nova-400/10 transition-colors" title="Edit">
                                                <Edit2 className="w-4 h-4" />
                                            </button>
                                            <button onClick={() => handleDelete(rule.id)} className="p-1.5 rounded-lg text-surface-400 hover:text-danger hover:bg-danger/10 transition-colors" title="Delete">
                                                <Trash2 className="w-4 h-4" />
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>
            )}

            {/* Incidents tab */}
            {tab === 'incidents' && (
                <div className="rounded-2xl border border-surface-700/50 bg-surface-800/40 backdrop-blur-sm overflow-hidden">
                    {loading ? (
                        <div className="flex items-center justify-center py-16 text-surface-400">
                            <div className="w-6 h-6 border-2 border-surface-500 border-t-nova-400 rounded-full animate-spin mr-3" />
                            Loading…
                        </div>
                    ) : incidents.length === 0 ? (
                        <div className="text-center py-16 text-surface-400">
                            <CheckCircle className="w-12 h-12 mx-auto mb-4 opacity-30" />
                            <p className="font-medium">No incidents</p>
                            <p className="text-sm mt-1 opacity-60">All clear — no threshold breaches recorded</p>
                        </div>
                    ) : (
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="border-b border-surface-700/50 text-surface-400 text-xs uppercase tracking-wider">
                                    <th className="text-left px-5 py-3 font-medium">Rule</th>
                                    <th className="text-left px-5 py-3 font-medium">Value</th>
                                    <th className="text-left px-5 py-3 font-medium">Fired</th>
                                    <th className="text-left px-5 py-3 font-medium">Resolved</th>
                                    <th className="text-left px-5 py-3 font-medium">Notified</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-surface-700/30">
                                {[...openIncidents, ...resolvedIncidents].map(inc => (
                                    <tr key={inc.id} className="hover:bg-surface-700/20 transition-colors">
                                        <td className="px-5 py-3.5 text-white font-medium">{inc.rule_name}</td>
                                        <td className="px-5 py-3.5">
                                            <span className={`font-mono text-sm ${inc.resolved_at ? 'text-surface-400' : 'text-warning'}`}>
                                                {inc.value?.toFixed(2)}
                                            </span>
                                        </td>
                                        <td className="px-5 py-3.5 text-surface-400 text-xs">
                                            <div className="flex items-center gap-1.5">
                                                <AlertTriangle className={`w-3.5 h-3.5 ${inc.resolved_at ? 'text-surface-500' : 'text-warning'}`} />
                                                {formatDate(inc.fired_at)}
                                            </div>
                                        </td>
                                        <td className="px-5 py-3.5 text-xs">
                                            {inc.resolved_at ? (
                                                <div className="flex items-center gap-1.5 text-success">
                                                    <CheckCircle className="w-3.5 h-3.5" />
                                                    {formatDate(inc.resolved_at)}
                                                </div>
                                            ) : (
                                                <div className="flex items-center gap-1.5 text-warning">
                                                    <Clock className="w-3.5 h-3.5" />
                                                    Ongoing
                                                </div>
                                            )}
                                        </td>
                                        <td className="px-5 py-3.5">
                                            <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs ${inc.notified ? 'bg-success/10 text-success' : 'bg-surface-600/50 text-surface-400'}`}>
                                                {inc.notified ? 'Sent' : 'Pending'}
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>
            )}

            {/* Create / Edit Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                    <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)} />
                    <div className="relative z-10 w-full max-w-lg rounded-2xl bg-surface-800 border border-surface-700/50 shadow-2xl p-6 max-h-[90vh] overflow-y-auto">
                        <div className="flex items-center justify-between mb-5">
                            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                                <Bell className="w-5 h-5 text-nova-400" />
                                {editRule ? 'Edit Alert Rule' : 'New Alert Rule'}
                            </h2>
                            <button onClick={() => setShowCreate(false)} className="text-surface-400 hover:text-white transition-colors">
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Rule Name</label>
                                <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                                    placeholder="e.g. High CPU Alert" className={inputCls} />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Server <span className="text-surface-500">(optional — leave empty for all)</span></label>
                                <select value={form.server_id} onChange={e => setForm(f => ({ ...f, server_id: e.target.value }))}
                                    className={inputCls + ' bg-surface-700/50'}>
                                    <option value="">All Servers</option>
                                    {servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                                </select>
                            </div>

                            <div className="grid grid-cols-3 gap-3">
                                <div className="col-span-1">
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">Metric</label>
                                    <select value={form.metric} onChange={e => setForm(f => ({ ...f, metric: e.target.value }))}
                                        className={inputCls + ' bg-surface-700/50'}>
                                        {METRICS.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">Operator</label>
                                    <select value={form.operator} onChange={e => setForm(f => ({ ...f, operator: e.target.value }))}
                                        className={inputCls + ' bg-surface-700/50'}>
                                        {OPERATORS.map(op => <option key={op} value={op}>{op}</option>)}
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">Threshold</label>
                                    <input type="number" value={form.threshold}
                                        onChange={e => setForm(f => ({ ...f, threshold: Number(e.target.value) }))}
                                        className={inputCls} />
                                </div>
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">
                                    Duration — breach must persist for <span className="text-nova-400">{form.duration_min} min</span>
                                </label>
                                <input type="range" min={1} max={60} value={form.duration_min}
                                    onChange={e => setForm(f => ({ ...f, duration_min: Number(e.target.value) }))}
                                    className="w-full accent-nova-500" />
                                <div className="flex justify-between text-xs text-surface-500 mt-1"><span>1m</span><span>60m</span></div>
                            </div>

                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">Notification Channel</label>
                                    <select value={form.channel} onChange={e => setForm(f => ({ ...f, channel: e.target.value }))}
                                        className={inputCls + ' bg-surface-700/50'}>
                                        {CHANNELS.map(c => <option key={c} value={c} className="capitalize">{c.charAt(0).toUpperCase() + c.slice(1)}</option>)}
                                    </select>
                                </div>
                                <div>
                                    <label className="block text-xs font-medium text-surface-300 mb-1.5">
                                        {form.channel === 'email' ? 'Email Address' : 'Webhook URL'}
                                    </label>
                                    <input value={form.destination}
                                        onChange={e => setForm(f => ({ ...f, destination: e.target.value }))}
                                        placeholder={form.channel === 'email' ? 'admin@example.com' : 'https://hooks.example.com/…'}
                                        className={inputCls} />
                                </div>
                            </div>

                            {formErr && (
                                <p className="text-danger text-xs flex items-center gap-1.5">
                                    <span className="w-1 h-1 rounded-full bg-danger" /> {formErr}
                                </p>
                            )}

                            <button onClick={handleSave} disabled={saving}
                                className="w-full py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-60 flex items-center justify-center gap-2">
                                {saving
                                    ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Saving…</>
                                    : <><Plus className="w-4 h-4" /> {editRule ? 'Update Rule' : 'Create Rule'}</>}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
