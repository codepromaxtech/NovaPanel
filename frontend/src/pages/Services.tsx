import { useState, useEffect } from 'react';
import { Cog, Play, Square, RotateCcw, RefreshCw, Loader, Search, X, FileText, CheckCircle2, XCircle, AlertTriangle, Clock } from 'lucide-react';
import { systemctlService } from '../services/system';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface Server { id: string; name: string; ip_address: string; }

export default function Services() {
    const toast = useToast();
    const [loading, setLoading] = useState(false);
    const [servers, setServers] = useState<Server[]>([]);
    const [server, setServer] = useState('');
    const [filter, setFilter] = useState('');
    const [output, setOutput] = useState('');
    const [statusOutput, setStatusOutput] = useState('');
    const [logOutput, setLogOutput] = useState('');
    const [selectedSvc, setSelectedSvc] = useState('');
    const [showStatus, setShowStatus] = useState(false);
    const [showLogs, setShowLogs] = useState(false);
    const [tab, setTab] = useState<'services' | 'failed' | 'timers'>('services');

    useEffect(() => { serverService.list(1, 100).then(r => setServers(r.data || [])).catch(() => { }); }, []);

    const fetchServices = async () => {
        if (!server) return;
        setLoading(true);
        try {
            if (tab === 'services') { const r = await systemctlService.list(server, filter); setOutput(r.output); }
            else if (tab === 'failed') { const r = await systemctlService.failed(server); setOutput(r.output); }
            else { const r = await systemctlService.timers(server); setOutput(r.output); }
        } catch { toast.error('Failed'); }
        setLoading(false);
    };
    useEffect(() => { if (server) fetchServices(); }, [server, tab]);

    const doAction = async (action: string, svc: string) => {
        if ((action === 'stop' || action === 'disable') && !confirm(`${action} ${svc}?`)) return;
        try {
            let r;
            switch (action) {
                case 'start': r = await systemctlService.start(server, svc); break;
                case 'stop': r = await systemctlService.stop(server, svc); break;
                case 'restart': r = await systemctlService.restart(server, svc); break;
                case 'reload': r = await systemctlService.reload(server, svc); break;
                case 'enable': r = await systemctlService.enable(server, svc); break;
                case 'disable': r = await systemctlService.disable(server, svc); break;
            }
            toast.success(r?.message || `${action} OK`);
            fetchServices();
        } catch { toast.error('Failed'); }
    };
    const viewStatus = async (svc: string) => {
        setSelectedSvc(svc);
        try { const r = await systemctlService.status(server, svc); setStatusOutput(r.output); setShowStatus(true); } catch { toast.error('Failed'); }
    };
    const viewLogs = async (svc: string) => {
        setSelectedSvc(svc);
        try { const r = await systemctlService.logs(server, svc, 100); setLogOutput(r.output); setShowLogs(true); } catch { toast.error('Failed'); }
    };
    const handleDaemonReload = async () => {
        try { await systemctlService.daemonReload(server); toast.success('Daemon reloaded'); } catch { toast.error('Failed'); }
    };

    const lines = output.split('\n').filter(l => l.trim());
    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-teal-500/50 text-sm";

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                    <div className="p-2.5 bg-gradient-to-br from-teal-500/20 to-emerald-500/10 rounded-xl border border-teal-500/20"><Cog className="w-6 h-6 text-teal-400" /></div>
                    System Services
                </h1>
                <p className="text-surface-200/50 text-sm mt-1">Manage systemd services — start, stop, restart, enable, disable, view logs</p>
            </div>

            <div className="flex gap-3 items-end flex-wrap">
                <div className="flex-1 min-w-48"><select value={server} onChange={e => setServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}</select></div>
                <div className="w-48"><input value={filter} onChange={e => setFilter(e.target.value)} placeholder="Filter services..." className={inputCls} /></div>
                <button onClick={fetchServices} className="p-2.5 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white"><Search className="w-5 h-5" /></button>
                <button onClick={handleDaemonReload} disabled={!server} className="p-2.5 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white" title="Daemon Reload"><RefreshCw className="w-5 h-5" /></button>
            </div>

            <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1">
                {[
                    { id: 'services' as const, icon: Cog, label: 'All Services' },
                    { id: 'failed' as const, icon: XCircle, label: 'Failed' },
                    { id: 'timers' as const, icon: Clock, label: 'Timers' },
                ].map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)} className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${tab === t.id ? 'bg-teal-500/20 text-teal-400 border border-teal-500/30' : 'text-surface-200/50 hover:text-white border border-transparent'}`}><t.icon className="w-4 h-4" /> {t.label}</button>
                ))}
            </div>

            {loading && <div className="flex justify-center py-8"><Loader className="w-6 h-6 animate-spin text-teal-400" /></div>}

            {server && !loading && tab === 'services' && (
                <div className="space-y-1">
                    {lines.map((line, i) => {
                        const parts = line.trim().split(/\s+/);
                        const svcName = parts[0] || '';
                        const isRunning = line.includes('running');
                        const isFailed = line.includes('failed');
                        return (
                            <div key={i} className="group flex items-center justify-between px-4 py-2 rounded-lg bg-surface-800/50 border border-surface-700/50 hover:border-surface-700 transition-colors">
                                <div className="flex items-center gap-2 flex-1 min-w-0">
                                    {isRunning ? <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" /> : isFailed ? <XCircle className="w-3.5 h-3.5 text-red-400 shrink-0" /> : <AlertTriangle className="w-3.5 h-3.5 text-surface-200/30 shrink-0" />}
                                    <span className="text-xs font-mono text-white truncate">{svcName}</span>
                                    <span className={`text-xs px-1.5 py-0.5 rounded ${isRunning ? 'text-emerald-400' : isFailed ? 'text-red-400' : 'text-surface-200/40'}`}>{isRunning ? 'running' : isFailed ? 'failed' : 'inactive'}</span>
                                </div>
                                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <button onClick={() => viewStatus(svcName)} className="px-2 py-1 text-xs rounded bg-surface-700/50 text-surface-200/60 hover:text-white">Status</button>
                                    <button onClick={() => viewLogs(svcName)} className="px-2 py-1 text-xs rounded bg-surface-700/50 text-surface-200/60 hover:text-white">Logs</button>
                                    {!isRunning && <button onClick={() => doAction('start', svcName)} className="px-2 py-1 text-xs rounded bg-emerald-500/20 text-emerald-400"><Play className="w-3 h-3 inline" /> Start</button>}
                                    {isRunning && <button onClick={() => doAction('restart', svcName)} className="px-2 py-1 text-xs rounded bg-blue-500/20 text-blue-400"><RotateCcw className="w-3 h-3 inline" /> Restart</button>}
                                    {isRunning && <button onClick={() => doAction('stop', svcName)} className="px-2 py-1 text-xs rounded bg-red-500/20 text-red-400"><Square className="w-3 h-3 inline" /> Stop</button>}
                                </div>
                            </div>
                        );
                    })}
                    {lines.length === 0 && <p className="text-center text-surface-200/30 py-8">No services found</p>}
                </div>
            )}
            {server && !loading && (tab === 'failed' || tab === 'timers') && (
                <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-96 overflow-auto border border-surface-700/30">{output || 'None'}</pre>
            )}
            {!server && <p className="text-center text-surface-200/30 py-12">Select a server to manage services</p>}

            {showStatus && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowStatus(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-bold text-white">Service: {selectedSvc}</h2><button onClick={() => setShowStatus(false)} className="p-1 hover:bg-surface-700 rounded-lg"><X className="w-5 h-5 text-surface-200/40" /></button></div>
                        <pre className="flex-1 overflow-auto bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono">{statusOutput}</pre>
                        <div className="flex gap-2 mt-4">
                            <button onClick={() => { doAction('start', selectedSvc); setShowStatus(false); }} className="px-3 py-2 text-xs rounded-lg bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">Start</button>
                            <button onClick={() => { doAction('restart', selectedSvc); setShowStatus(false); }} className="px-3 py-2 text-xs rounded-lg bg-blue-500/20 text-blue-400 border border-blue-500/30">Restart</button>
                            <button onClick={() => { doAction('stop', selectedSvc); setShowStatus(false); }} className="px-3 py-2 text-xs rounded-lg bg-red-500/20 text-red-400 border border-red-500/30">Stop</button>
                            <button onClick={() => { doAction('enable', selectedSvc); }} className="px-3 py-2 text-xs rounded-lg bg-surface-700/50 text-surface-200/60">Enable</button>
                            <button onClick={() => { doAction('disable', selectedSvc); }} className="px-3 py-2 text-xs rounded-lg bg-surface-700/50 text-surface-200/60">Disable</button>
                        </div>
                    </div>
                </div>
            )}
            {showLogs && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowLogs(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-bold text-white flex items-center gap-2"><FileText className="w-5 h-5 text-teal-400" /> Logs: {selectedSvc}</h2><button onClick={() => setShowLogs(false)} className="p-1 hover:bg-surface-700 rounded-lg"><X className="w-5 h-5 text-surface-200/40" /></button></div>
                        <pre className="flex-1 overflow-auto bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono">{logOutput}</pre>
                    </div>
                </div>
            )}
        </div>
    );
}
