import { useState, useEffect, useRef, useCallback } from 'react';
import { FolderOpen, File, ChevronRight, Plus, Trash2, ArrowLeft, Loader, X, Save, Search, Copy, Scissors, FileText, FolderPlus, RefreshCw, Archive, MoreVertical, Edit3, Eye, Lock, Download } from 'lucide-react';
import { fileService } from '../services/files';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface FileItem {
    name: string; path: string; is_dir: boolean; is_link?: boolean;
    size: number; permissions: string; modified_at: string; owner?: string; group?: string;
}

const EXT_LANG: Record<string, string> = {
    js: 'javascript', jsx: 'javascript', ts: 'typescript', tsx: 'typescript', py: 'python',
    go: 'go', rs: 'rust', php: 'php', rb: 'ruby', java: 'java', c: 'c', cpp: 'cpp', h: 'c',
    html: 'html', htm: 'html', css: 'css', scss: 'scss', less: 'less',
    json: 'json', xml: 'xml', yaml: 'yaml', yml: 'yaml', toml: 'toml',
    md: 'markdown', sql: 'sql', sh: 'bash', bash: 'bash', zsh: 'bash',
    conf: 'conf', env: 'env', ini: 'ini', nginx: 'nginx', dockerfile: 'dockerfile',
    log: 'log', txt: 'text',
};
const EXT_ICONS: Record<string, string> = {
    js: '🟨', ts: '🔷', py: '🐍', go: '🔵', php: '🐘', html: '🌐', css: '🎨', json: '📋',
    md: '📝', sh: '⚡', yml: '⚙️', yaml: '⚙️', sql: '🗃️', log: '📄', env: '🔒',
};

function formatSize(bytes: number): string {
    if (bytes === 0) return '—';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`;
    return `${(bytes / 1073741824).toFixed(1)} GB`;
}

function getExt(name: string): string { return name.split('.').pop()?.toLowerCase() || ''; }

export default function FileManager() {
    const toast = useToast();
    const [serverId, setServerId] = useState('');
    const [servers, setServers] = useState<any[]>([]);
    const [files, setFiles] = useState<FileItem[]>([]);
    const [currentPath, setCurrentPath] = useState('/');
    const [loading, setLoading] = useState(false);

    // Editor
    const [openTabs, setOpenTabs] = useState<{ path: string; name: string; content: string; modified: boolean }[]>([]);
    const [activeTab, setActiveTab] = useState('');
    const editorRef = useRef<HTMLTextAreaElement>(null);

    // Modals
    const [showCreate, setShowCreate] = useState<'file' | 'dir' | null>(null);
    const [createName, setCreateName] = useState('');
    const [showRename, setShowRename] = useState<FileItem | null>(null);
    const [renameName, setRenameName] = useState('');
    const [showSearch, setShowSearch] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const [searchResults, setSearchResults] = useState<string[]>([]);
    const [showChmod, setShowChmod] = useState<FileItem | null>(null);
    const [chmodValue, setChmodValue] = useState('');
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; file: FileItem } | null>(null);

    useEffect(() => { serverService.list(1, 100).then(r => setServers(r.data || [])).catch(() => { }); }, []);

    const fetchFiles = useCallback(async (path: string) => {
        if (!serverId) return;
        setLoading(true);
        try {
            const res = await fileService.list(serverId, path);
            setFiles(res.files || []);
        } catch { setFiles([]); toast.error('Failed to list files'); }
        setLoading(false);
    }, [serverId]);

    useEffect(() => { if (serverId) fetchFiles(currentPath); }, [currentPath, serverId]);

    const navigateTo = (path: string) => setCurrentPath(path);
    const goUp = () => { const p = currentPath.split('/').slice(0, -1).join('/') || '/'; setCurrentPath(p); };

    // Open file in editor tab
    const openFile = async (file: FileItem) => {
        if (file.is_dir) { navigateTo(file.path); return; }
        if (openTabs.some(t => t.path === file.path)) { setActiveTab(file.path); return; }
        try {
            const res = await fileService.read(serverId, file.path);
            if (res.is_binary) { toast.error('Binary file — cannot edit'); return; }
            const content = res.content || '';
            setOpenTabs(prev => [...prev, { path: file.path, name: file.name, content, modified: false }]);
            setActiveTab(file.path);
        } catch { toast.error('Failed to read file'); }
    };

    const closeTab = (path: string) => {
        const tab = openTabs.find(t => t.path === path);
        if (tab?.modified && !confirm('Discard changes?')) return;
        setOpenTabs(prev => prev.filter(t => t.path !== path));
        if (activeTab === path) setActiveTab(openTabs.find(t => t.path !== path)?.path || '');
    };

    const updateContent = (content: string) => {
        setOpenTabs(prev => prev.map(t => t.path === activeTab ? { ...t, content, modified: true } : t));
    };

    const saveFile = async () => {
        const tab = openTabs.find(t => t.path === activeTab);
        if (!tab) return;
        try {
            await fileService.write(serverId, tab.path, tab.content);
            setOpenTabs(prev => prev.map(t => t.path === activeTab ? { ...t, modified: false } : t));
            toast.success('Saved');
        } catch { toast.error('Failed to save'); }
    };

    // Keyboard shortcuts
    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 's') { e.preventDefault(); saveFile(); }
            if ((e.ctrlKey || e.metaKey) && e.key === 'f') { e.preventDefault(); setShowSearch(true); }
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, [activeTab, openTabs]);

    // File operations
    const handleCreate = async () => {
        if (!createName) return;
        const path = currentPath === '/' ? `/${createName}` : `${currentPath}/${createName}`;
        try {
            await fileService.create(serverId, path, showCreate === 'dir');
            toast.success(`Created ${createName}`);
            setShowCreate(null); setCreateName(''); fetchFiles(currentPath);
        } catch { toast.error('Failed'); }
    };
    const handleRename = async () => {
        if (!showRename || !renameName) return;
        const dir = showRename.path.substring(0, showRename.path.lastIndexOf('/'));
        const newPath = `${dir}/${renameName}`;
        try {
            await fileService.rename(serverId, showRename.path, newPath);
            toast.success('Renamed'); setShowRename(null); fetchFiles(currentPath);
        } catch { toast.error('Failed'); }
    };
    const handleDelete = async (file: FileItem) => {
        if (!confirm(`Delete "${file.name}"?`)) return;
        try { await fileService.remove(serverId, file.path); toast.success('Deleted'); fetchFiles(currentPath); } catch { toast.error('Failed'); }
    };
    const handleSearch = async () => {
        if (!searchQuery) return;
        try { const r = await fileService.search(serverId, currentPath, searchQuery); setSearchResults(r.results || []); } catch { toast.error('Search failed'); }
    };
    const handleGrep = async () => {
        if (!searchQuery) return;
        try { const r = await fileService.grep(serverId, currentPath, searchQuery); setSearchResults(r.results || []); } catch { toast.error('Grep failed'); }
    };
    const handleChmod = async () => {
        if (!showChmod || !chmodValue) return;
        try { await fileService.chmod(serverId, showChmod.path, chmodValue); toast.success('Permissions changed'); setShowChmod(null); fetchFiles(currentPath); } catch { toast.error('Failed'); }
    };
    const handleExtract = async (file: FileItem) => {
        try { await fileService.extract(serverId, file.path); toast.success('Extracted'); fetchFiles(currentPath); } catch { toast.error('Failed'); }
    };
    const handleCompress = async (file: FileItem) => {
        try { await fileService.compress(serverId, file.path); toast.success('Compressed'); fetchFiles(currentPath); } catch { toast.error('Failed'); }
    };

    const activeContent = openTabs.find(t => t.path === activeTab);
    const lineCount = activeContent ? activeContent.content.split('\n').length : 0;
    const ext = activeTab ? getExt(activeTab) : '';
    const lang = EXT_LANG[ext] || 'text';

    const dirs = files.filter(f => f.is_dir).sort((a, b) => a.name.localeCompare(b.name));
    const regularFiles = files.filter(f => !f.is_dir).sort((a, b) => a.name.localeCompare(b.name));
    const sorted = [...dirs, ...regularFiles];

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-nova-500/50 text-sm";

    // ─── No server selected ───
    if (!serverId) {
        return (
            <div className="space-y-6 max-w-md mx-auto mt-12">
                <div className="text-center">
                    <div className="inline-flex p-4 bg-gradient-to-br from-nova-500/20 to-blue-500/10 rounded-2xl border border-nova-500/20 mb-4"><FolderOpen className="w-10 h-10 text-nova-400" /></div>
                    <h1 className="text-2xl font-bold text-white">File Manager</h1>
                    <p className="text-surface-200/50 text-sm mt-2">Select a server to browse files</p>
                </div>
                <div className="space-y-2">{servers.map(s => (
                    <button key={s.id} onClick={() => setServerId(s.id)} className="w-full text-left bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-nova-500/30 transition-colors">
                        <p className="text-white font-semibold">{s.name}</p>
                        <p className="text-xs text-surface-200/40">{s.ip_address} · {s.hostname}</p>
                    </button>
                ))}{servers.length === 0 && <p className="text-center text-surface-200/30 py-8">No servers found</p>}</div>
            </div>
        );
    }

    // ─── Main Layout ───
    return (
        <div className="h-[calc(100vh-4rem)] flex flex-col" onClick={() => setContextMenu(null)}>
            {/* Top Bar */}
            <div className="flex items-center justify-between px-4 py-2 bg-surface-800/80 border-b border-surface-700/30">
                <div className="flex items-center gap-3">
                    <button onClick={() => setServerId('')} className="text-xs text-surface-200/40 hover:text-white">← Servers</button>
                    <FolderOpen className="w-5 h-5 text-nova-400" />
                    <span className="text-white font-semibold text-sm">File Manager</span>
                    <span className="text-surface-200/30 text-xs">{servers.find(s => s.id === serverId)?.name}</span>
                </div>
                <div className="flex items-center gap-2">
                    <button onClick={() => setShowSearch(true)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white" title="Search (Ctrl+F)"><Search className="w-4 h-4" /></button>
                    <button onClick={() => { setShowCreate('file'); setCreateName(''); }} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white" title="New File"><FileText className="w-4 h-4" /></button>
                    <button onClick={() => { setShowCreate('dir'); setCreateName(''); }} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white" title="New Folder"><FolderPlus className="w-4 h-4" /></button>
                    <button onClick={() => fetchFiles(currentPath)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white"><RefreshCw className="w-4 h-4" /></button>
                </div>
            </div>

            {/* Breadcrumb */}
            <div className="flex items-center gap-1 text-xs px-4 py-1.5 bg-surface-900/50 border-b border-surface-700/20 min-h-[32px]">
                {currentPath !== '/' && <button onClick={goUp} className="p-0.5 rounded hover:bg-surface-700/50 text-surface-200/50"><ArrowLeft className="w-3 h-3" /></button>}
                <button onClick={() => navigateTo('/')} className="text-surface-200/50 hover:text-white">/</button>
                {currentPath.split('/').filter(Boolean).map((seg, i, arr) => (
                    <span key={i} className="flex items-center gap-1">
                        <ChevronRight className="w-2.5 h-2.5 text-surface-200/20" />
                        <button onClick={() => navigateTo('/' + arr.slice(0, i + 1).join('/'))}
                            className={`hover:text-nova-400 ${i === arr.length - 1 ? 'text-white font-medium' : 'text-surface-200/50'}`}>{seg}</button>
                    </span>
                ))}
            </div>

            <div className="flex-1 flex overflow-hidden">
                {/* File List Panel */}
                <div className="w-80 border-r border-surface-700/30 overflow-auto bg-surface-900/30 flex-shrink-0">
                    {loading ? (
                        <div className="flex items-center justify-center py-16"><Loader className="w-5 h-5 text-nova-500 animate-spin" /></div>
                    ) : (
                        <div className="py-1">{sorted.map(file => {
                            const fext = getExt(file.name);
                            const icon = file.is_dir ? '📁' : (EXT_ICONS[fext] || '📄');
                            return (
                                <div key={file.path}
                                    className="group flex items-center gap-2 px-3 py-1.5 hover:bg-surface-700/30 cursor-pointer text-sm transition-colors"
                                    onClick={() => openFile(file)}
                                    onContextMenu={e => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, file }); }}>
                                    <span className="text-xs w-5 text-center flex-shrink-0">{icon}</span>
                                    <span className={`truncate flex-1 ${file.is_dir ? 'text-white font-medium' : 'text-surface-200/70'}`}>{file.name}</span>
                                    <span className="text-[10px] text-surface-200/20 flex-shrink-0">{file.is_dir ? '' : formatSize(file.size)}</span>
                                    <button onClick={e => { e.stopPropagation(); setContextMenu({ x: e.clientX, y: e.clientY, file }); }}
                                        className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-surface-700/50 text-surface-200/30"><MoreVertical className="w-3 h-3" /></button>
                                </div>
                            );
                        })}{sorted.length === 0 && <p className="text-center text-surface-200/30 py-8 text-sm">Empty directory</p>}</div>
                    )}
                </div>

                {/* Editor Panel */}
                <div className="flex-1 flex flex-col bg-surface-950 overflow-hidden">
                    {/* Tabs */}
                    {openTabs.length > 0 && (
                        <div className="flex border-b border-surface-700/30 bg-surface-900/50 overflow-x-auto">
                            {openTabs.map(tab => (
                                <div key={tab.path} onClick={() => setActiveTab(tab.path)}
                                    className={`flex items-center gap-2 px-3 py-2 text-xs cursor-pointer border-r border-surface-700/20 min-w-[120px] max-w-[200px] ${activeTab === tab.path ? 'bg-surface-950 text-white border-b-2 border-b-nova-500' : 'text-surface-200/50 hover:bg-surface-800/50'}`}>
                                    <span className="truncate flex-1">{tab.modified ? '● ' : ''}{tab.name}</span>
                                    <button onClick={e => { e.stopPropagation(); closeTab(tab.path); }} className="p-0.5 rounded hover:bg-surface-700/50"><X className="w-3 h-3" /></button>
                                </div>
                            ))}
                        </div>
                    )}

                    {activeContent ? (
                        <div className="flex-1 flex overflow-hidden">
                            {/* Line numbers */}
                            <div className="py-3 px-2 bg-surface-900/50 text-right select-none border-r border-surface-700/20 overflow-hidden">
                                {Array.from({ length: lineCount }, (_, i) => (
                                    <div key={i} className="text-[11px] leading-[20px] text-surface-200/20 font-mono pr-1">{i + 1}</div>
                                ))}
                            </div>
                            {/* Editor */}
                            <textarea ref={editorRef} value={activeContent.content} onChange={e => updateContent(e.target.value)}
                                className="flex-1 bg-transparent text-surface-200/80 font-mono text-[13px] leading-[20px] p-3 focus:outline-none resize-none overflow-auto"
                                spellCheck={false} style={{ tabSize: 4 }} />
                        </div>
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-surface-200/20">
                            <div className="text-center">
                                <FileText className="w-12 h-12 mx-auto mb-3 opacity-30" />
                                <p className="text-sm">Select a file to edit</p>
                                <p className="text-xs mt-1 text-surface-200/10">Ctrl+S to save · Ctrl+F to search</p>
                            </div>
                        </div>
                    )}

                    {/* Status bar */}
                    {activeContent && (
                        <div className="flex items-center justify-between px-3 py-1 bg-surface-800/80 border-t border-surface-700/20 text-[10px] text-surface-200/30">
                            <div className="flex items-center gap-3">
                                <span>{lang.toUpperCase()}</span>
                                <span>Lines: {lineCount}</span>
                                <span>Size: {formatSize(new Blob([activeContent.content]).size)}</span>
                            </div>
                            <div className="flex items-center gap-2">
                                {activeContent.modified && <span className="text-amber-400">Modified</span>}
                                <button onClick={saveFile} className="flex items-center gap-1 px-2 py-0.5 rounded bg-nova-500/20 text-nova-400 hover:bg-nova-500/30 text-[10px]">
                                    <Save className="w-3 h-3" /> Save
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Context Menu */}
            {contextMenu && (
                <div className="fixed z-[70] bg-surface-800 border border-surface-700/50 rounded-xl shadow-2xl py-1 min-w-[160px]"
                    style={{ left: contextMenu.x, top: contextMenu.y }}>
                    {!contextMenu.file.is_dir && <button onClick={() => { openFile(contextMenu.file); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-surface-200/70 hover:bg-surface-700/50 flex items-center gap-2"><Edit3 className="w-3 h-3" /> Edit</button>}
                    <button onClick={() => { setShowRename(contextMenu.file); setRenameName(contextMenu.file.name); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-surface-200/70 hover:bg-surface-700/50 flex items-center gap-2"><Edit3 className="w-3 h-3" /> Rename</button>
                    <button onClick={() => { setShowChmod(contextMenu.file); setChmodValue(''); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-surface-200/70 hover:bg-surface-700/50 flex items-center gap-2"><Lock className="w-3 h-3" /> Permissions</button>
                    {/\.(tar|gz|zip|bz2|xz|tgz)$/i.test(contextMenu.file.name) && <button onClick={() => { handleExtract(contextMenu.file); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-surface-200/70 hover:bg-surface-700/50 flex items-center gap-2"><Archive className="w-3 h-3" /> Extract</button>}
                    {contextMenu.file.is_dir && <button onClick={() => { handleCompress(contextMenu.file); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-surface-200/70 hover:bg-surface-700/50 flex items-center gap-2"><Archive className="w-3 h-3" /> Compress</button>}
                    <div className="border-t border-surface-700/30 my-1" />
                    <button onClick={() => { handleDelete(contextMenu.file); setContextMenu(null); }} className="w-full text-left px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 flex items-center gap-2"><Trash2 className="w-3 h-3" /> Delete</button>
                </div>
            )}

            {/* Create File/Dir Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4">{showCreate === 'dir' ? 'New Folder' : 'New File'}</h2>
                        <input value={createName} onChange={e => setCreateName(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleCreate()}
                            placeholder={showCreate === 'dir' ? 'folder-name' : 'filename.ext'} className={inputCls} autoFocus />
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowCreate(null)} className="px-4 py-2 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleCreate} className="px-4 py-2 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium">Create</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Rename Modal */}
            {showRename && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowRename(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4">Rename</h2>
                        <input value={renameName} onChange={e => setRenameName(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleRename()} className={inputCls} autoFocus />
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowRename(null)} className="px-4 py-2 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleRename} className="px-4 py-2 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium">Rename</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Chmod Modal */}
            {showChmod && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowChmod(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-2">Change Permissions</h2>
                        <p className="text-xs text-surface-200/40 mb-3">{showChmod.path} — current: {showChmod.permissions}</p>
                        <input value={chmodValue} onChange={e => setChmodValue(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleChmod()}
                            placeholder="755" className={inputCls} autoFocus />
                        <div className="flex gap-2 mt-2 flex-wrap">{['644', '755', '775', '777', '600', '700'].map(m =>
                            <button key={m} onClick={() => setChmodValue(m)} className={`px-2 py-1 rounded text-xs ${chmodValue === m ? 'bg-nova-500/20 text-nova-400' : 'bg-surface-700/30 text-surface-200/50'}`}>{m}</button>
                        )}</div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowChmod(null)} className="px-4 py-2 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleChmod} className="px-4 py-2 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium">Apply</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Search Modal */}
            {showSearch && (
                <div className="fixed inset-0 z-[60] flex items-start justify-center pt-20 bg-black/40 backdrop-blur-sm" onClick={() => setShowSearch(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-4 w-full max-w-lg shadow-2xl" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center gap-2 mb-3">
                            <Search className="w-4 h-4 text-surface-200/40" />
                            <input value={searchQuery} onChange={e => setSearchQuery(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleSearch()}
                                placeholder="Search files or grep in content..." className="flex-1 bg-transparent text-white text-sm focus:outline-none" autoFocus />
                            <button onClick={handleSearch} className="px-2 py-1 rounded-lg bg-nova-500/20 text-nova-400 text-xs">Find Files</button>
                            <button onClick={handleGrep} className="px-2 py-1 rounded-lg bg-surface-700/50 text-surface-200/50 text-xs">Grep</button>
                        </div>
                        {searchResults.length > 0 && (
                            <div className="max-h-60 overflow-auto space-y-0.5">{searchResults.map((r, i) =>
                                <div key={i} className="px-2 py-1 text-xs text-surface-200/60 font-mono hover:bg-surface-700/30 rounded cursor-pointer truncate"
                                    onClick={() => { setShowSearch(false); const parts = r.split(':'); if (parts.length > 0) navigateTo(parts[0].substring(0, parts[0].lastIndexOf('/'))); }}>{r}</div>
                            )}</div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
