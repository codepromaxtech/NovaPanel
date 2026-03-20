import { useState, useEffect } from 'react';
import { Clock, Plus, Trash2, Edit3, Loader, RefreshCw, FileText, X, Terminal } from 'lucide-react';
import { cronService } from '../services/system';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface Server { id: string; name: string; ip_address: string; }

export default function CronJobs() {
    const toast = useToast();
    const [loading, setLoading] = useState(false);
    const [servers, setServers] = useState<Server[]>([]);
    const [server, setServer] = useState('');
    const [user, setUser] = useState('root');
    const [cronOutput, setCronOutput] = useState('');
    const [logOutput, setLogOutput] = useState('');
    const [showAdd, setShowAdd] = useState(false);
    const [showLogs, setShowLogs] = useState(false);
    const [form, setForm] = useState({ schedule: '', command: '', comment: '' });

    useEffect(() => { serverService.list(1, 100).then(r => setServers(r.data || [])).catch(() => { }); }, []);

    const fetchCron = async () => {
        if (!server) return;
        setLoading(true);
        try { const r = await cronService.list(server, user); setCronOutput(r.output || 'No cron jobs'); } catch { toast.error('Failed'); }
        setLoading(false);
    };
    useEffect(() => { if (server) fetchCron(); }, [server, user]);

    const handleAdd = async () => {
        if (!form.schedule || !form.command) { toast.error('Schedule and command required'); return; }
        try { await cronService.add(server, form.schedule, form.command, user, form.comment); toast.success('Added'); setShowAdd(false); setForm({ schedule: '', command: '', comment: '' }); fetchCron(); } catch { toast.error('Failed'); }
    };
    const handleDelete = async (lineNum: number) => {
        if (!confirm('Delete this cron job?')) return;
        try { await cronService.remove(server, lineNum, user); toast.success('Deleted'); fetchCron(); } catch { toast.error('Failed'); }
    };
    const handleLogs = async () => {
        try { const r = await cronService.logs(server, 100); setLogOutput(r.output); setShowLogs(true); } catch { toast.error('Failed'); }
    };

    const lines = cronOutput.split('\n').filter(l => l.trim());
    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-amber-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-amber-600 to-amber-700 text-white text-sm font-medium hover:from-amber-500 hover:to-amber-600 transition-all shadow-lg shadow-amber-500/20";

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-amber-500/20 to-orange-500/10 rounded-xl border border-amber-500/20"><Clock className="w-6 h-6 text-amber-400" /></div>
                        Cron Jobs
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Manage scheduled tasks on your servers</p>
                </div>
            </div>
            <div className="flex gap-3 items-end">
                <div className="flex-1"><label className="block text-sm text-surface-200/60 mb-1">Server</label><select value={server} onChange={e => setServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}</select></div>
                <div className="w-32"><label className="block text-sm text-surface-200/60 mb-1">User</label><input value={user} onChange={e => setUser(e.target.value)} className={inputCls} /></div>
                <button onClick={fetchCron} className="p-2.5 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white"><RefreshCw className="w-5 h-5" /></button>
                <button onClick={handleLogs} disabled={!server} className="p-2.5 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white"><FileText className="w-5 h-5" /></button>
                <button onClick={() => setShowAdd(true)} disabled={!server} className={btnPrimary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> Add</button>
            </div>
            {loading && <div className="flex justify-center py-8"><Loader className="w-6 h-6 animate-spin text-amber-400" /></div>}
            {server && !loading && (
                <div className="space-y-1">
                    {lines.map((line, i) => {
                        const isComment = line.startsWith('#');
                        return (
                            <div key={i} className={`group flex items-center justify-between px-4 py-2.5 rounded-lg ${isComment ? 'bg-surface-800/30' : 'bg-surface-800/50 border border-surface-700/50'}`}>
                                <pre className={`text-xs font-mono flex-1 ${isComment ? 'text-surface-200/30 italic' : 'text-surface-200/70'}`}>{line}</pre>
                                {!isComment && (
                                    <button onClick={() => handleDelete(i + 1)} className="p-1.5 rounded hover:bg-red-500/20 text-surface-200/30 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"><Trash2 className="w-3.5 h-3.5" /></button>
                                )}
                            </div>
                        );
                    })}
                    {lines.length === 0 && <p className="text-center text-surface-200/30 py-8">No cron jobs for user {user}</p>}
                </div>
            )}
            {!server && <p className="text-center text-surface-200/30 py-12">Select a server to view cron jobs</p>}

            {showAdd && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAdd(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-md shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><Plus className="w-5 h-5 text-amber-400" /> Add Cron Job</h2>
                        <div className="space-y-3">
                            <input value={form.comment} onChange={e => setForm({ ...form, comment: e.target.value })} placeholder="Comment (optional)" className={inputCls} />
                            <input value={form.schedule} onChange={e => setForm({ ...form, schedule: e.target.value })} placeholder="Schedule (e.g. */5 * * * *)" className={inputCls + " font-mono"} />
                            <input value={form.command} onChange={e => setForm({ ...form, command: e.target.value })} placeholder="Command to run" className={inputCls + " font-mono"} />
                            <div className="text-xs text-surface-200/30 bg-surface-900/50 rounded-lg p-3 font-mono space-y-1">
                                <p>┌───────── minute (0-59)</p>
                                <p>│ ┌─────── hour (0-23)</p>
                                <p>│ │ ┌───── day of month (1-31)</p>
                                <p>│ │ │ ┌─── month (1-12)</p>
                                <p>│ │ │ │ ┌─ day of week (0-7)</p>
                                <p>* * * * *</p>
                            </div>
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowAdd(false)} className="px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleAdd} className={btnPrimary}>Add Job</button>
                        </div>
                    </div>
                </div>
            )}
            {showLogs && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowLogs(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-bold text-white flex items-center gap-2"><Terminal className="w-5 h-5 text-amber-400" /> Cron Logs</h2><button onClick={() => setShowLogs(false)} className="p-1 hover:bg-surface-700 rounded-lg"><X className="w-5 h-5 text-surface-200/40" /></button></div>
                        <pre className="flex-1 overflow-auto bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono">{logOutput || 'No logs'}</pre>
                    </div>
                </div>
            )}
        </div>
    );
}
