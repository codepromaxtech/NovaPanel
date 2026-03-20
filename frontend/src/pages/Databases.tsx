import { useState, useEffect } from 'react';
import { Database, Plus, Trash2, Search, HardDrive, Play, Table, Users, Download, Upload, ExternalLink, StopCircle, Loader, X, Terminal, Wrench } from 'lucide-react';
import { databaseService } from '../services/databases';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface DBEntry { id: string; name: string; engine: string; db_user: string; size_mb: number; status: string; server_id: string; created_at: string; }
interface Server { id: string; name: string; ip_address: string; }
type Tab = 'databases' | 'query' | 'tools' | 'users' | 'export';

const TABS: { id: Tab; label: string; icon: any }[] = [
    { id: 'databases', label: 'Databases', icon: Database },
    { id: 'query', label: 'Query Runner', icon: Terminal },
    { id: 'tools', label: 'Web Tools', icon: Wrench },
    { id: 'users', label: 'Users', icon: Users },
    { id: 'export', label: 'Export / Import', icon: Download },
];
const ENGINES = [
    { value: 'mysql', label: 'MySQL / MariaDB', icon: '🗄️', tool: 'phpMyAdmin', port: '8081' },
    { value: 'postgresql', label: 'PostgreSQL', icon: '🐘', tool: 'Adminer', port: '8082' },
    { value: 'mongodb', label: 'MongoDB', icon: '🍃', tool: 'Mongo Express', port: '8083' },
    { value: 'redis', label: 'Redis', icon: '⚡', tool: 'Redis Commander', port: '8084' },
];

export default function Databases() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('databases');
    const [loading, setLoading] = useState(false);
    const [databases, setDatabases] = useState<DBEntry[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [showCreate, setShowCreate] = useState(false);
    const [newDB, setNewDB] = useState({ name: '', engine: 'mysql', server_id: '' });

    // Query runner state
    const [qServer, setQServer] = useState('');
    const [qEngine, setQEngine] = useState('mysql');
    const [qDB, setQDB] = useState('');
    const [qQuery, setQQuery] = useState('');
    const [qOutput, setQOutput] = useState('');
    const [qRunning, setQRunning] = useState(false);

    // Tools state
    const [toolServer, setToolServer] = useState('');
    const [toolStatuses, setToolStatuses] = useState<Record<string, any>>({});

    // Users state
    const [uServer, setUServer] = useState('');
    const [uEngine, setUEngine] = useState('mysql');
    const [uOutput, setUOutput] = useState('');
    const [showCreateUser, setShowCreateUser] = useState(false);
    const [newUser, setNewUser] = useState({ username: '', password: '', database: '' });

    // Export state
    const [eServer, setEServer] = useState('');
    const [eEngine, setEEngine] = useState('mysql');
    const [eDB, setEDB] = useState('');
    const [eOutput, setEOutput] = useState('');
    const [importPath, setImportPath] = useState('');

    const fetchDatabases = async () => { setLoading(true); try { const r = await databaseService.list(); setDatabases(r.data || []); } catch { toast.error('Failed'); } setLoading(false); };
    const fetchServers = async () => { try { const r = await serverService.list(1, 100); setServers(r.data || []); } catch { } };

    useEffect(() => { fetchServers(); fetchDatabases(); }, []);

    const handleCreate = async () => {
        if (!newDB.name) { toast.error('Name required'); return; }
        try { await databaseService.create(newDB); toast.success('Database created'); setShowCreate(false); setNewDB({ name: '', engine: 'mysql', server_id: '' }); fetchDatabases(); } catch { toast.error('Failed'); }
    };
    const handleDelete = async (id: string) => { if (!confirm('Delete this database?')) return; try { await databaseService.remove(id); toast.success('Deleted'); fetchDatabases(); } catch { toast.error('Failed'); } };

    const handleRunQuery = async () => {
        if (!qServer || !qDB || !qQuery) { toast.error('Fill all fields'); return; }
        setQRunning(true); setQOutput('');
        try { const r = await databaseService.runQuery(qServer, qEngine, qDB, qQuery); setQOutput(r.output || r.error || 'No output'); } catch { toast.error('Failed'); }
        setQRunning(false);
    };
    const handleListTables = async () => {
        if (!qServer || !qDB) { toast.error('Select server and database'); return; }
        setQRunning(true);
        try { const r = await databaseService.listTables(qServer, qEngine, qDB); setQOutput(r.output); } catch { toast.error('Failed'); }
        setQRunning(false);
    };

    const checkToolStatus = async (engine: string) => {
        if (!toolServer) return;
        try { const r = await databaseService.toolStatus(toolServer, engine); setToolStatuses(prev => ({ ...prev, [engine]: r })); } catch { }
    };
    const handleDeployTool = async (engine: string) => {
        if (!toolServer) { toast.error('Select server'); return; }
        toast.success('Deploying... this may take a minute');
        try { const r = await databaseService.deployTool(toolServer, engine); setToolStatuses(prev => ({ ...prev, [engine]: r })); toast.success(`${r.tool} deployed!`); } catch { toast.error('Failed to deploy'); }
    };
    const handleStopTool = async (engine: string) => {
        if (!confirm('Stop this tool?')) return;
        try { await databaseService.stopTool(toolServer, engine); setToolStatuses(prev => ({ ...prev, [engine]: { ...prev[engine], status: 'not_found' } })); toast.success('Stopped'); } catch { toast.error('Failed'); }
    };

    const handleListUsers = async () => {
        if (!uServer) { toast.error('Select server'); return; }
        try { const r = await databaseService.listUsers(uServer, uEngine); setUOutput(r.output); } catch { toast.error('Failed'); }
    };
    const handleCreateUser = async () => {
        if (!newUser.username || !newUser.password || !newUser.database) { toast.error('Fill all fields'); return; }
        try { await databaseService.createUser(uServer, uEngine, newUser.username, newUser.password, newUser.database); toast.success('User created'); setShowCreateUser(false); handleListUsers(); } catch { toast.error('Failed'); }
    };

    const handleExport = async () => {
        if (!eServer || !eDB) { toast.error('Select server and database'); return; }
        try { const r = await databaseService.exportDB(eServer, eEngine, eDB); setEOutput(r.output); toast.success('Export completed'); } catch { toast.error('Failed'); }
    };
    const handleImport = async () => {
        if (!eServer || !eDB || !importPath) { toast.error('Fill all fields'); return; }
        if (!confirm('Import will overwrite existing data. Continue?')) return;
        try { const r = await databaseService.importDB(eServer, eEngine, eDB, importPath); setEOutput(r.output); toast.success('Import completed'); } catch { toast.error('Failed'); }
    };

    useEffect(() => { if (toolServer) ENGINES.forEach(e => checkToolStatus(e.value)); }, [toolServer]);

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-indigo-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-indigo-600 to-indigo-700 text-white text-sm font-medium hover:from-indigo-500 hover:to-indigo-600 transition-all shadow-lg shadow-indigo-500/20";
    const btnSecondary = "px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm";

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-indigo-500/20 to-purple-500/10 rounded-xl border border-indigo-500/20"><Database className="w-6 h-6 text-indigo-400" /></div>
                        Database Manager
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Manage databases, run queries, deploy web tools (phpMyAdmin, Adminer, Mongo Express, Redis Commander)</p>
                </div>
            </div>

            <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1">
                {TABS.map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)}
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all ${tab === t.id ? 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/30' : 'text-surface-200/50 hover:text-white hover:bg-surface-700/30 border border-transparent'}`}>
                        <t.icon className="w-4 h-4" /> {t.label}
                    </button>
                ))}
            </div>

            {loading && <div className="flex justify-center py-12"><Loader className="w-6 h-6 animate-spin text-indigo-400" /></div>}

            {/* ════════ DATABASES TAB ════════ */}
            {tab === 'databases' && !loading && (
                <div className="space-y-4">
                    <div className="flex justify-end"><button onClick={() => setShowCreate(true)} className={btnPrimary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> Create Database</button></div>
                    <div className="space-y-2">
                        {databases.map(db => (
                            <div key={db.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                                <div className="flex items-center gap-3">
                                    <div className="p-2 rounded-lg bg-indigo-500/10 border border-indigo-500/20"><HardDrive className="w-4 h-4 text-indigo-400" /></div>
                                    <div>
                                        <p className="text-white font-medium">{db.name}</p>
                                        <p className="text-xs text-surface-200/40">{db.engine} · User: {db.db_user} · {db.size_mb}MB · {db.status}</p>
                                    </div>
                                </div>
                                <button onClick={() => handleDelete(db.id)} className="p-2 rounded-lg hover:bg-red-500/20 text-surface-200/40 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                            </div>
                        ))}
                        {databases.length === 0 && <p className="text-center text-surface-200/30 py-12">No databases yet</p>}
                    </div>
                </div>
            )}

            {/* ════════ QUERY RUNNER TAB ════════ */}
            {tab === 'query' && (
                <div className="space-y-4 max-w-3xl">
                    <div className="grid grid-cols-3 gap-3">
                        <select value={qServer} onChange={e => setQServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                        <select value={qEngine} onChange={e => setQEngine(e.target.value)} className={inputCls + " bg-transparent"}>{ENGINES.map(e => <option key={e.value} value={e.value}>{e.label}</option>)}</select>
                        <input value={qDB} onChange={e => setQDB(e.target.value)} placeholder="Database name" className={inputCls} />
                    </div>
                    <textarea value={qQuery} onChange={e => setQQuery(e.target.value)} placeholder={qEngine === 'redis' ? 'e.g. KEYS *' : 'SELECT * FROM users LIMIT 10;'} rows={4}
                        className={inputCls + " font-mono resize-y"} />
                    <div className="flex gap-3">
                        <button onClick={handleRunQuery} disabled={qRunning} className={btnPrimary + " flex items-center gap-2"}>{qRunning ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />} Run Query</button>
                        <button onClick={handleListTables} className={btnSecondary + " flex items-center gap-2"}><Table className="w-4 h-4" /> List Tables</button>
                    </div>
                    {qOutput && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-96 overflow-auto border border-surface-700/30">{qOutput}</pre>}
                </div>
            )}

            {/* ════════ WEB TOOLS TAB ════════ */}
            {tab === 'tools' && (
                <div className="space-y-4">
                    <div className="flex items-center gap-3 mb-2">
                        <select value={toolServer} onChange={e => setToolServer(e.target.value)} className={inputCls + " max-w-xs bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}</select>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        {ENGINES.map(eng => {
                            const st = toolStatuses[eng.value];
                            const isRunning = st?.status === 'running';
                            return (
                                <div key={eng.value} className={`bg-surface-800/50 border rounded-xl p-5 transition-colors ${isRunning ? 'border-emerald-500/30' : 'border-surface-700/50'}`}>
                                    <div className="flex items-center justify-between mb-3">
                                        <div className="flex items-center gap-3">
                                            <span className="text-2xl">{eng.icon}</span>
                                            <div>
                                                <p className="text-white font-semibold">{eng.tool}</p>
                                                <p className="text-xs text-surface-200/40">for {eng.label}</p>
                                            </div>
                                        </div>
                                        <span className={`px-2 py-0.5 rounded-full text-xs ${isRunning ? 'bg-emerald-500/20 text-emerald-400' : 'bg-surface-700/50 text-surface-200/40'}`}>
                                            {isRunning ? 'Running' : 'Stopped'}
                                        </span>
                                    </div>
                                    <div className="flex gap-2">
                                        {isRunning ? (
                                            <>
                                                <a href={st?.url} target="_blank" rel="noopener noreferrer" className={btnPrimary + " flex items-center gap-1 text-xs !px-3 !py-2"}><ExternalLink className="w-3 h-3" /> Open</a>
                                                <button onClick={() => handleStopTool(eng.value)} className="px-3 py-2 rounded-lg bg-red-500/10 text-red-400 text-xs border border-red-500/20 hover:bg-red-500/20"><StopCircle className="w-3 h-3 inline mr-1" />Stop</button>
                                            </>
                                        ) : (
                                            <button onClick={() => handleDeployTool(eng.value)} className={btnPrimary + " text-xs !px-3 !py-2"}>Deploy {eng.tool}</button>
                                        )}
                                    </div>
                                    {isRunning && st?.url && <p className="text-xs text-surface-200/30 mt-2 font-mono">{st.url}</p>}
                                </div>
                            );
                        })}
                    </div>
                    {!toolServer && <p className="text-center text-surface-200/30 py-8">Select a server above to manage DB tools</p>}
                </div>
            )}

            {/* ════════ USERS TAB ════════ */}
            {tab === 'users' && (
                <div className="space-y-4 max-w-2xl">
                    <div className="grid grid-cols-2 gap-3">
                        <select value={uServer} onChange={e => setUServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                        <select value={uEngine} onChange={e => setUEngine(e.target.value)} className={inputCls + " bg-transparent"}>{ENGINES.filter(e => e.value !== 'redis').map(e => <option key={e.value} value={e.value}>{e.label}</option>)}</select>
                    </div>
                    <div className="flex gap-3">
                        <button onClick={handleListUsers} className={btnPrimary + " flex items-center gap-2"}><Users className="w-4 h-4" /> List Users</button>
                        <button onClick={() => setShowCreateUser(true)} className={btnSecondary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> Create User</button>
                    </div>
                    {uOutput && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-96 overflow-auto border border-surface-700/30">{uOutput}</pre>}
                </div>
            )}

            {/* ════════ EXPORT/IMPORT TAB ════════ */}
            {tab === 'export' && (
                <div className="space-y-4 max-w-2xl">
                    <div className="grid grid-cols-3 gap-3">
                        <select value={eServer} onChange={e => setEServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select>
                        <select value={eEngine} onChange={e => setEEngine(e.target.value)} className={inputCls + " bg-transparent"}>{ENGINES.filter(e => e.value !== 'redis').map(e => <option key={e.value} value={e.value}>{e.label}</option>)}</select>
                        <input value={eDB} onChange={e => setEDB(e.target.value)} placeholder="Database name" className={inputCls} />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                            <h3 className="text-white font-semibold text-sm mb-2 flex items-center gap-2"><Download className="w-4 h-4 text-indigo-400" /> Export</h3>
                            <p className="text-xs text-surface-200/40 mb-3">Dump database to /tmp on the server</p>
                            <button onClick={handleExport} className={btnPrimary + " w-full text-xs"}>Export Database</button>
                        </div>
                        <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                            <h3 className="text-white font-semibold text-sm mb-2 flex items-center gap-2"><Upload className="w-4 h-4 text-emerald-400" /> Import</h3>
                            <input value={importPath} onChange={e => setImportPath(e.target.value)} placeholder="File path on server (e.g. /tmp/backup.sql)" className={inputCls + " mb-3"} />
                            <button onClick={handleImport} className="w-full px-4 py-2 rounded-lg bg-emerald-500/20 text-emerald-400 text-xs font-medium border border-emerald-500/30 hover:bg-emerald-500/30">Import Database</button>
                        </div>
                    </div>
                    {eOutput && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-60 overflow-auto border border-surface-700/30">{eOutput}</pre>}
                </div>
            )}

            {/* ════════ CREATE DB MODAL ════════ */}
            {showCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><Database className="w-5 h-5 text-indigo-400" /> Create Database</h2>
                        <div className="space-y-3">
                            <input value={newDB.name} onChange={e => setNewDB({ ...newDB, name: e.target.value })} placeholder="Database name" className={inputCls} />
                            <select value={newDB.engine} onChange={e => setNewDB({ ...newDB, engine: e.target.value })} className={inputCls + " bg-transparent"}>
                                {ENGINES.map(e => <option key={e.value} value={e.value}>{e.label}</option>)}
                            </select>
                            <select value={newDB.server_id} onChange={e => setNewDB({ ...newDB, server_id: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="">Select server</option>
                                {servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                            </select>
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowCreate(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreate} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}

            {/* ════════ CREATE USER MODAL ════════ */}
            {showCreateUser && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreateUser(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><Users className="w-5 h-5 text-indigo-400" /> Create DB User</h2>
                        <div className="space-y-3">
                            <input value={newUser.username} onChange={e => setNewUser({ ...newUser, username: e.target.value })} placeholder="Username" className={inputCls} />
                            <input type="password" value={newUser.password} onChange={e => setNewUser({ ...newUser, password: e.target.value })} placeholder="Password" className={inputCls} />
                            <input value={newUser.database} onChange={e => setNewUser({ ...newUser, database: e.target.value })} placeholder="Grant access to database" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowCreateUser(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateUser} className={btnPrimary}>Create User</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
