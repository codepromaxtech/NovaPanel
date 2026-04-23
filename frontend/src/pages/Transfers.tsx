import { useState, useEffect } from 'react';
import { ArrowRightLeft, Plus, Trash2, RotateCcw, X, Loader, Eye, XCircle, CheckCircle2, Clock, PlayCircle, Calendar, ToggleLeft, ToggleRight, HardDrive, RefreshCw } from 'lucide-react';
import { transferService } from '../services/transfers';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface Server { id: string; name: string; ip_address: string; }
interface TransferJob {
    id: string; source_server_id: string; dest_server_id: string;
    source_path: string; dest_path: string; direction: string;
    rsync_options: string; exclude_patterns: string; bandwidth_limit: number;
    delete_extra: boolean; dry_run: boolean; status: string;
    bytes_transferred: number; files_transferred: number; progress: number;
    output: string; started_at: string; completed_at: string; created_at: string;
}
interface Schedule {
    id: string; name: string; source_server_id: string; dest_server_id: string;
    source_path: string; dest_path: string; direction: string;
    cron_expression: string; is_active: boolean; last_run: string; next_run: string; created_at: string;
}

type Tab = 'jobs' | 'new' | 'schedules';

export default function Transfers() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('jobs');
    const [loading, setLoading] = useState(false);
    const [jobs, setJobs] = useState<TransferJob[]>([]);
    const [schedules, setSchedules] = useState<Schedule[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [viewJob, setViewJob] = useState<TransferJob | null>(null);

    const [form, setForm] = useState({
        source_server_id: '', dest_server_id: '', source_path: '', dest_path: '',
        direction: 'push', rsync_options: '-avzh --progress', exclude_patterns: '',
        bandwidth_limit: 0, delete_extra: false, dry_run: false,
    });
    const [schedForm, setSchedForm] = useState({
        name: '', source_server_id: '', dest_server_id: '', source_path: '', dest_path: '',
        direction: 'push', rsync_options: '-avzh --progress', exclude_patterns: '',
        bandwidth_limit: 0, delete_extra: false, cron_expression: '',
    });
    const [showNewSchedule, setShowNewSchedule] = useState(false);

    const fetchJobs = async () => { setLoading(true); try { const r = await transferService.list(); setJobs(r.data || []); } catch { toast.error('Failed to load'); } setLoading(false); };
    const fetchSchedules = async () => { setLoading(true); try { const r = await transferService.listSchedules(); setSchedules(r.data || []); } catch { toast.error('Failed'); } setLoading(false); };
    const fetchServers = async () => { try { const r = await serverService.list(1, 100); setServers(r.data || []); } catch { } };

    useEffect(() => { fetchServers(); }, []);
    useEffect(() => {
        if (tab === 'jobs') fetchJobs();
        else if (tab === 'schedules') fetchSchedules();
    }, [tab]);

    // Auto-refresh running jobs
    useEffect(() => {
        if (tab !== 'jobs') return;
        const hasRunning = jobs.some(j => j.status === 'running' || j.status === 'pending');
        if (!hasRunning) return;
        const iv = setInterval(fetchJobs, 5000);
        return () => clearInterval(iv);
    }, [tab, jobs]);

    const handleCreate = async () => {
        if (!form.source_path || !form.dest_path) { toast.error('Source and dest paths required'); return; }
        try { await transferService.create(form); toast.success('Transfer started'); setTab('jobs'); fetchJobs(); } catch { toast.error('Failed'); }
    };
    const handleCancel = async (id: string) => { if (!confirm('Cancel this transfer?')) return; try { await transferService.cancel(id); toast.success('Cancelled'); fetchJobs(); } catch { toast.error('Failed'); } };
    const handleRetry = async (id: string) => { try { await transferService.retry(id); toast.success('Retrying'); fetchJobs(); } catch { toast.error('Failed'); } };
    const handleDelete = async (id: string) => { if (!confirm('Delete this transfer job?')) return; try { await transferService.remove(id); toast.success('Deleted'); fetchJobs(); } catch { toast.error('Failed'); } };
    const handleCreateSchedule = async () => {
        if (!schedForm.name || !schedForm.cron_expression) { toast.error('Name and cron required'); return; }
        try { await transferService.createSchedule(schedForm); toast.success('Schedule created'); setShowNewSchedule(false); fetchSchedules(); } catch { toast.error('Failed'); }
    };
    const handleDeleteSchedule = async (id: string) => { if (!confirm('Delete this schedule?')) return; try { await transferService.deleteSchedule(id); toast.success('Deleted'); fetchSchedules(); } catch { toast.error('Failed'); } };
    const handleToggleSchedule = async (id: string, active: boolean) => { try { await transferService.toggleSchedule(id, !active); fetchSchedules(); } catch { toast.error('Failed'); } };

    const statusBadge = (s: string) => {
        const m: Record<string, string> = { pending: 'bg-yellow-500/20 text-yellow-400', running: 'bg-blue-500/20 text-blue-400', completed: 'bg-emerald-500/20 text-emerald-400', failed: 'bg-red-500/20 text-red-400', cancelled: 'bg-surface-700/50 text-surface-200/50' };
        return <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${m[s] || m.pending}`}>{s}</span>;
    };
    const serverName = (id: string) => servers.find(s => s.id === id)?.name || id?.slice(0, 8) || '—';

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-cyan-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-cyan-600 to-cyan-700 text-white text-sm font-medium hover:from-cyan-500 hover:to-cyan-600 transition-all shadow-lg shadow-cyan-500/20";
    const btnSecondary = "px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm";

    const TABS: { id: Tab; label: string; icon: any }[] = [
        { id: 'jobs', label: 'Transfer Jobs', icon: ArrowRightLeft },
        { id: 'new', label: 'New Transfer', icon: Plus },
        { id: 'schedules', label: 'Schedules', icon: Calendar },
    ];

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-cyan-500/20 to-blue-500/10 rounded-xl border border-cyan-500/20"><ArrowRightLeft className="w-6 h-6 text-cyan-400" /></div>
                        File Transfers — Rsync
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Transfer files between servers using rsync with scheduling, bandwidth control, and exclusion patterns</p>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1">
                {TABS.map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)}
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all ${tab === t.id ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30' : 'text-surface-200/50 hover:text-white hover:bg-surface-700/30 border border-transparent'}`}>
                        <t.icon className="w-4 h-4" /> {t.label}
                    </button>
                ))}
            </div>

            {loading && <div className="flex justify-center py-12"><Loader className="w-6 h-6 animate-spin text-cyan-400" /></div>}

            {/* ════════════ JOBS TAB ════════════ */}
            {tab === 'jobs' && !loading && (
                <div className="space-y-3">
                    {jobs.map(j => (
                        <div key={j.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-4">
                                    <div className={`p-2.5 rounded-xl border ${j.status === 'running' ? 'bg-blue-500/10 border-blue-500/20' : j.status === 'completed' ? 'bg-emerald-500/10 border-emerald-500/20' : j.status === 'failed' ? 'bg-red-500/10 border-red-500/20' : 'bg-surface-700/30 border-surface-700/50'}`}>
                                        {j.status === 'running' ? <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" /> : j.status === 'completed' ? <CheckCircle2 className="w-5 h-5 text-emerald-400" /> : j.status === 'failed' ? <XCircle className="w-5 h-5 text-red-400" /> : <Clock className="w-5 h-5 text-surface-200/40" />}
                                    </div>
                                    <div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-white font-medium text-sm">{serverName(j.source_server_id)}</span>
                                            <ArrowRightLeft className="w-3.5 h-3.5 text-surface-200/50" />
                                            <span className="text-white font-medium text-sm">{serverName(j.dest_server_id)}</span>
                                            {statusBadge(j.status)}
                                            {j.dry_run && <span className="px-2 py-0.5 rounded-full text-xs bg-purple-500/20 text-purple-400">dry-run</span>}
                                        </div>
                                        <p className="text-xs text-surface-200/40 mt-0.5">{j.source_path} → {j.dest_path}</p>
                                        <p className="text-xs text-surface-200/50">{j.direction} · {new Date(j.created_at).toLocaleString()}</p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <button onClick={() => setViewJob(j)} className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors"><Eye className="w-4 h-4" /></button>
                                    {j.status === 'running' && <button onClick={() => handleCancel(j.id)} className="p-2 rounded-lg hover:bg-red-500/20 text-surface-200/40 hover:text-red-400 transition-colors"><XCircle className="w-4 h-4" /></button>}
                                    {j.status === 'failed' && <button onClick={() => handleRetry(j.id)} className="p-2 rounded-lg hover:bg-blue-500/20 text-surface-200/40 hover:text-blue-400 transition-colors"><RotateCcw className="w-4 h-4" /></button>}
                                    {(j.status === 'completed' || j.status === 'failed' || j.status === 'cancelled') && <button onClick={() => handleDelete(j.id)} className="p-2 rounded-lg hover:bg-red-500/20 text-surface-200/40 hover:text-red-400 transition-colors"><Trash2 className="w-4 h-4" /></button>}
                                </div>
                            </div>
                            {j.status === 'running' && j.progress > 0 && (
                                <div className="mt-3 h-1.5 bg-surface-900 rounded-full overflow-hidden">
                                    <div className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 rounded-full transition-all" style={{ width: `${j.progress}%` }} />
                                </div>
                            )}
                        </div>
                    ))}
                    {jobs.length === 0 && <p className="text-center text-surface-200/50 py-16">No transfer jobs yet — start one from the "New Transfer" tab</p>}
                </div>
            )}

            {/* ════════════ NEW TRANSFER TAB ════════════ */}
            {tab === 'new' && (
                <div className="space-y-5 max-w-2xl">
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Source Server</label>
                            <select value={form.source_server_id} onChange={e => setForm({ ...form, source_server_id: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="">Select source server</option>
                                {servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}
                            </select>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Destination Server</label>
                            <select value={form.dest_server_id} onChange={e => setForm({ ...form, dest_server_id: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="">Select dest server</option>
                                {servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}
                            </select>
                        </div>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div><label className="block text-sm font-medium text-surface-200/60 mb-1.5">Source Path</label><input value={form.source_path} onChange={e => setForm({ ...form, source_path: e.target.value })} placeholder="/var/www/html/" className={inputCls} /></div>
                        <div><label className="block text-sm font-medium text-surface-200/60 mb-1.5">Destination Path</label><input value={form.dest_path} onChange={e => setForm({ ...form, dest_path: e.target.value })} placeholder="/backup/www/" className={inputCls} /></div>
                    </div>
                    <div className="grid grid-cols-3 gap-4">
                        <div>
                            <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Direction</label>
                            <select value={form.direction} onChange={e => setForm({ ...form, direction: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="push">Push (src → dest)</option>
                                <option value="pull">Pull (dest ← src)</option>
                                <option value="local">Local (same server)</option>
                            </select>
                        </div>
                        <div><label className="block text-sm font-medium text-surface-200/60 mb-1.5">Bandwidth Limit (KB/s)</label><input type="number" value={form.bandwidth_limit} onChange={e => setForm({ ...form, bandwidth_limit: parseInt(e.target.value) || 0 })} placeholder="0 = unlimited" className={inputCls} /></div>
                        <div><label className="block text-sm font-medium text-surface-200/60 mb-1.5">Rsync Options</label><input value={form.rsync_options} onChange={e => setForm({ ...form, rsync_options: e.target.value })} className={inputCls} /></div>
                    </div>
                    <div><label className="block text-sm font-medium text-surface-200/60 mb-1.5">Exclude Patterns (comma separated)</label><input value={form.exclude_patterns} onChange={e => setForm({ ...form, exclude_patterns: e.target.value })} placeholder="*.log, .git/, node_modules/, *.tmp" className={inputCls} /></div>
                    <div className="flex items-center gap-6">
                        <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={form.delete_extra} onChange={e => setForm({ ...form, delete_extra: e.target.checked })} className="accent-cyan-500" /><span className="text-sm text-surface-200/60">Delete extra files at destination (--delete)</span></label>
                        <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={form.dry_run} onChange={e => setForm({ ...form, dry_run: e.target.checked })} className="accent-purple-500" /><span className="text-sm text-surface-200/60">Dry run (preview only)</span></label>
                    </div>
                    <div className="flex gap-3 pt-2">
                        <button onClick={handleCreate} className={btnPrimary + " flex items-center gap-2"}><PlayCircle className="w-4 h-4" /> {form.dry_run ? 'Preview Transfer' : 'Start Transfer'}</button>
                    </div>
                </div>
            )}

            {/* ════════════ SCHEDULES TAB ════════════ */}
            {tab === 'schedules' && !loading && (
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <p className="text-sm text-surface-200/50">Recurring rsync transfers on a cron schedule</p>
                        <button onClick={() => setShowNewSchedule(true)} className={btnPrimary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> New Schedule</button>
                    </div>
                    <div className="space-y-2">
                        {schedules.map(s => (
                            <div key={s.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                                <div className="flex items-center gap-3">
                                    <div className={`p-2 rounded-lg border ${s.is_active ? 'bg-emerald-500/10 border-emerald-500/20' : 'bg-surface-700/30 border-surface-700/50'}`}>
                                        <Calendar className={`w-4 h-4 ${s.is_active ? 'text-emerald-400' : 'text-surface-200/50'}`} />
                                    </div>
                                    <div>
                                        <p className="text-white font-medium text-sm">{s.name}</p>
                                        <p className="text-xs text-surface-200/40">{s.source_path} → {s.dest_path} · cron: <code className="text-cyan-400">{s.cron_expression}</code></p>
                                        {s.last_run && <p className="text-xs text-surface-200/50">Last: {new Date(s.last_run).toLocaleString()}</p>}
                                    </div>
                                </div>
                                <div className="flex items-center gap-2">
                                    <button onClick={() => handleToggleSchedule(s.id, s.is_active)} className="p-1" aria-label={s.is_active ? 'Disable schedule' : 'Enable schedule'} aria-pressed={s.is_active}>
                                        {s.is_active ? <ToggleRight className="w-7 h-7 text-emerald-400" /> : <ToggleLeft className="w-7 h-7 text-surface-200/50" />}
                                    </button>
                                    <button onClick={() => handleDeleteSchedule(s.id)} className="p-2 rounded-lg hover:bg-red-500/20 text-surface-200/40 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                                </div>
                            </div>
                        ))}
                        {schedules.length === 0 && <p className="text-center text-surface-200/50 py-12">No scheduled transfers</p>}
                    </div>
                </div>
            )}

            {/* ════════ VIEW OUTPUT MODAL ════════ */}
            {viewJob && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setViewJob(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-4">
                            <h2 className="text-lg font-bold text-white flex items-center gap-2"><Eye className="w-5 h-5 text-cyan-400" /> Transfer Output</h2>
                            <button onClick={() => setViewJob(null)} className="p-1 hover:bg-surface-700 rounded-lg"><X className="w-5 h-5 text-surface-200/40" /></button>
                        </div>
                        <div className="grid grid-cols-2 gap-3 mb-4 text-xs">
                            <div className="bg-surface-900/50 rounded-lg p-2"><span className="text-surface-200/40">Status:</span> {statusBadge(viewJob.status)}</div>
                            <div className="bg-surface-900/50 rounded-lg p-2"><span className="text-surface-200/40">Direction:</span> <span className="text-white ml-1">{viewJob.direction}</span></div>
                            <div className="bg-surface-900/50 rounded-lg p-2"><span className="text-surface-200/40">Source:</span> <span className="text-white ml-1">{viewJob.source_path}</span></div>
                            <div className="bg-surface-900/50 rounded-lg p-2"><span className="text-surface-200/40">Dest:</span> <span className="text-white ml-1">{viewJob.dest_path}</span></div>
                        </div>
                        <div className="flex-1 overflow-auto">
                            <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono">{viewJob.output || 'No output yet…'}</pre>
                        </div>
                    </div>
                </div>
            )}

            {/* ════════ NEW SCHEDULE MODAL ════════ */}
            {showNewSchedule && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowNewSchedule(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-lg shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><Calendar className="w-5 h-5 text-cyan-400" /> New Scheduled Transfer</h2>
                        <div className="space-y-3">
                            <input value={schedForm.name} onChange={e => setSchedForm({ ...schedForm, name: e.target.value })} placeholder="Schedule name" className={inputCls} />
                            <div className="grid grid-cols-2 gap-3">
                                <select value={schedForm.source_server_id} onChange={e => setSchedForm({ ...schedForm, source_server_id: e.target.value })} className={inputCls + " bg-transparent"}>
                                    <option value="">Source server</option>
                                    {servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                                </select>
                                <select value={schedForm.dest_server_id} onChange={e => setSchedForm({ ...schedForm, dest_server_id: e.target.value })} className={inputCls + " bg-transparent"}>
                                    <option value="">Dest server</option>
                                    {servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                                </select>
                            </div>
                            <div className="grid grid-cols-2 gap-3">
                                <input value={schedForm.source_path} onChange={e => setSchedForm({ ...schedForm, source_path: e.target.value })} placeholder="Source path" className={inputCls} />
                                <input value={schedForm.dest_path} onChange={e => setSchedForm({ ...schedForm, dest_path: e.target.value })} placeholder="Dest path" className={inputCls} />
                            </div>
                            <input value={schedForm.cron_expression} onChange={e => setSchedForm({ ...schedForm, cron_expression: e.target.value })} placeholder="Cron expression (e.g. 0 2 * * *)" className={inputCls} />
                            <select value={schedForm.direction} onChange={e => setSchedForm({ ...schedForm, direction: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="push">Push</option><option value="pull">Pull</option><option value="local">Local</option>
                            </select>
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowNewSchedule(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateSchedule} className={btnPrimary}>Create Schedule</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
