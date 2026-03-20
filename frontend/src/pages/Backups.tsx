import { useState, useEffect } from 'react';
import { Archive, Plus, Trash2, Loader, Download, Upload, Database, FolderOpen, HardDrive, X, RotateCcw } from 'lucide-react';
import { backupService } from '../services/backups';
import { backupManagerService } from '../services/system';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface BackupEntry { id: string; type: string; storage: string; path: string; size_mb: number; status: string; created_at: string; }
interface Server { id: string; name: string; ip_address: string; }
type Tab = 'list' | 'create' | 'restore' | 'files';

const ENGINES = [
    { value: 'mysql', label: 'MySQL' },
    { value: 'postgresql', label: 'PostgreSQL' },
    { value: 'mongodb', label: 'MongoDB' },
];

export default function Backups() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('list');
    const [loading, setLoading] = useState(false);
    const [backups, setBackups] = useState<BackupEntry[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [output, setOutput] = useState('');
    const [filesOutput, setFilesOutput] = useState('');

    // Create backup form
    const [bkType, setBkType] = useState<'database' | 'site' | 'full'>('database');
    const [bkServer, setBkServer] = useState('');
    const [bkEngine, setBkEngine] = useState('mysql');
    const [bkDB, setBkDB] = useState('');
    const [bkSitePath, setBkSitePath] = useState('/var/www/html');
    const [bkSiteName, setBkSiteName] = useState('');

    // Restore form
    const [rType, setRType] = useState<'database' | 'site'>('database');
    const [rServer, setRServer] = useState('');
    const [rEngine, setREngine] = useState('mysql');
    const [rDB, setRDB] = useState('');
    const [rSitePath, setRSitePath] = useState('/var/www/html');
    const [rBackupPath, setRBackupPath] = useState('');

    const fetchBackups = async () => { setLoading(true); try { const r = await backupService.list(); setBackups(r.data || []); } catch { toast.error('Failed'); } setLoading(false); };
    useEffect(() => { serverService.list(1, 100).then(r => setServers(r.data || [])).catch(() => { }); fetchBackups(); }, []);

    const handleBackup = async () => {
        if (!bkServer) { toast.error('Select server'); return; }
        setOutput(''); toast.success('Starting backup...');
        try {
            let r;
            if (bkType === 'database') { r = await backupManagerService.backupDatabase(bkServer, bkEngine, bkDB); }
            else if (bkType === 'site') { r = await backupManagerService.backupSite(bkServer, bkSitePath, bkSiteName); }
            else { r = await backupManagerService.backupFull(bkServer); }
            setOutput(r.output || JSON.stringify(r, null, 2));
            toast.success('Backup completed!');
            fetchBackups();
        } catch { toast.error('Backup failed'); }
    };

    const handleRestore = async () => {
        if (!rServer || !rBackupPath) { toast.error('Fill required fields'); return; }
        if (!confirm('This will restore from backup. Existing data may be overwritten. Continue?')) return;
        setOutput('');
        try {
            let r;
            if (rType === 'database') { r = await backupManagerService.restoreDatabase(rServer, rEngine, rDB, rBackupPath); }
            else { r = await backupManagerService.restoreSite(rServer, rSitePath, rBackupPath); }
            setOutput(r.output || 'Restore completed');
            toast.success('Restore completed!');
        } catch { toast.error('Restore failed'); }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Delete this backup record?')) return;
        try { await backupService.remove(id); toast.success('Deleted'); fetchBackups(); } catch { toast.error('Failed'); }
    };

    const handleListFiles = async () => {
        if (!bkServer) { toast.error('Select server first'); return; }
        try { const r = await backupManagerService.listFiles(bkServer); setFilesOutput(r.output); } catch { toast.error('Failed'); }
    };

    const handleDeleteFile = async (filePath: string) => {
        if (!confirm(`Delete backup file: ${filePath}?`)) return;
        try { await backupManagerService.deleteFile(bkServer, filePath); toast.success('Deleted'); handleListFiles(); } catch { toast.error('Failed'); }
    };

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-violet-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-violet-600 to-violet-700 text-white text-sm font-medium hover:from-violet-500 hover:to-violet-600 transition-all shadow-lg shadow-violet-500/20";
    const TABS: { id: Tab; label: string; icon: any }[] = [
        { id: 'list', label: 'Backup History', icon: Archive },
        { id: 'create', label: 'Create Backup', icon: Download },
        { id: 'restore', label: 'Restore', icon: Upload },
        { id: 'files', label: 'Backup Files', icon: FolderOpen },
    ];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                    <div className="p-2.5 bg-gradient-to-br from-violet-500/20 to-purple-500/10 rounded-xl border border-violet-500/20"><Archive className="w-6 h-6 text-violet-400" /></div>
                    Backup & Restore
                </h1>
                <p className="text-surface-200/50 text-sm mt-1">Database backups, site backups, full server backups, and restore</p>
            </div>

            <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1">
                {TABS.map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)} className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all ${tab === t.id ? 'bg-violet-500/20 text-violet-400 border border-violet-500/30' : 'text-surface-200/50 hover:text-white border border-transparent'}`}><t.icon className="w-4 h-4" /> {t.label}</button>
                ))}
            </div>

            {/* LIST TAB */}
            {tab === 'list' && (
                <div className="space-y-2">
                    {loading && <div className="flex justify-center py-8"><Loader className="w-6 h-6 animate-spin text-violet-400" /></div>}
                    {backups.map(b => (
                        <div key={b.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                            <div className="flex items-center gap-3">
                                <div className={`p-2 rounded-lg border ${b.status === 'completed' ? 'bg-emerald-500/10 border-emerald-500/20' : 'bg-red-500/10 border-red-500/20'}`}>
                                    {b.type === 'database' ? <Database className="w-4 h-4 text-indigo-400" /> : b.type === 'site' ? <FolderOpen className="w-4 h-4 text-amber-400" /> : <HardDrive className="w-4 h-4 text-violet-400" />}
                                </div>
                                <div>
                                    <p className="text-white font-medium text-sm">{b.type} backup</p>
                                    <p className="text-xs text-surface-200/40">{b.path || 'N/A'} · {b.storage} · {b.status} · {new Date(b.created_at).toLocaleString()}</p>
                                </div>
                            </div>
                            <button onClick={() => handleDelete(b.id)} className="p-2 rounded-lg hover:bg-red-500/20 text-surface-200/40 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {!loading && backups.length === 0 && <p className="text-center text-surface-200/30 py-12">No backups yet</p>}
                </div>
            )}

            {/* CREATE BACKUP TAB */}
            {tab === 'create' && (
                <div className="space-y-4 max-w-lg">
                    <select value={bkServer} onChange={e => setBkServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                    <div className="grid grid-cols-3 gap-2">
                        {(['database', 'site', 'full'] as const).map(t => (
                            <button key={t} onClick={() => setBkType(t)} className={`px-4 py-3 rounded-lg text-sm font-medium border transition-all ${bkType === t ? 'bg-violet-500/20 text-violet-400 border-violet-500/30' : 'bg-surface-800/50 text-surface-200/50 border-surface-700/50'}`}>
                                {t === 'database' ? '🗄️ Database' : t === 'site' ? '📁 Site' : '💾 Full Server'}
                            </button>
                        ))}
                    </div>
                    {bkType === 'database' && (
                        <div className="grid grid-cols-2 gap-3">
                            <select value={bkEngine} onChange={e => setBkEngine(e.target.value)} className={inputCls + " bg-transparent"}>{ENGINES.map(e => <option key={e.value} value={e.value}>{e.label}</option>)}</select>
                            <input value={bkDB} onChange={e => setBkDB(e.target.value)} placeholder="Database name" className={inputCls} />
                        </div>
                    )}
                    {bkType === 'site' && (
                        <div className="grid grid-cols-2 gap-3">
                            <input value={bkSitePath} onChange={e => setBkSitePath(e.target.value)} placeholder="Site path" className={inputCls} />
                            <input value={bkSiteName} onChange={e => setBkSiteName(e.target.value)} placeholder="Site name (optional)" className={inputCls} />
                        </div>
                    )}
                    <button onClick={handleBackup} className={btnPrimary + " w-full flex items-center justify-center gap-2"}><Download className="w-4 h-4" /> Create Backup</button>
                    {output && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-48 overflow-auto border border-surface-700/30">{output}</pre>}
                </div>
            )}

            {/* RESTORE TAB */}
            {tab === 'restore' && (
                <div className="space-y-4 max-w-lg">
                    <select value={rServer} onChange={e => setRServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                    <div className="grid grid-cols-2 gap-2">
                        {(['database', 'site'] as const).map(t => (
                            <button key={t} onClick={() => setRType(t)} className={`px-4 py-3 rounded-lg text-sm font-medium border transition-all ${rType === t ? 'bg-violet-500/20 text-violet-400 border-violet-500/30' : 'bg-surface-800/50 text-surface-200/50 border-surface-700/50'}`}>
                                {t === 'database' ? '🗄️ Database' : '📁 Site'}
                            </button>
                        ))}
                    </div>
                    {rType === 'database' && (
                        <div className="grid grid-cols-2 gap-3">
                            <select value={rEngine} onChange={e => setREngine(e.target.value)} className={inputCls + " bg-transparent"}>{ENGINES.map(e => <option key={e.value} value={e.value}>{e.label}</option>)}</select>
                            <input value={rDB} onChange={e => setRDB(e.target.value)} placeholder="Database name" className={inputCls} />
                        </div>
                    )}
                    {rType === 'site' && (
                        <input value={rSitePath} onChange={e => setRSitePath(e.target.value)} placeholder="Restore to path" className={inputCls} />
                    )}
                    <input value={rBackupPath} onChange={e => setRBackupPath(e.target.value)} placeholder="Backup file path on server (e.g. /var/backups/novapanel/...)" className={inputCls + " font-mono"} />
                    <button onClick={handleRestore} className="w-full px-6 py-2.5 rounded-lg bg-gradient-to-r from-amber-600 to-amber-700 text-white text-sm font-medium flex items-center justify-center gap-2"><RotateCcw className="w-4 h-4" /> Restore from Backup</button>
                    {output && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-48 overflow-auto border border-surface-700/30">{output}</pre>}
                </div>
            )}

            {/* FILES TAB */}
            {tab === 'files' && (
                <div className="space-y-4">
                    <div className="flex gap-3">
                        <select value={bkServer} onChange={e => setBkServer(e.target.value)} className={inputCls + " max-w-xs bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                        <button onClick={handleListFiles} className={btnPrimary}>List Files</button>
                    </div>
                    {filesOutput && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-96 overflow-auto border border-surface-700/30">{filesOutput}</pre>}
                </div>
            )}
        </div>
    );
}
