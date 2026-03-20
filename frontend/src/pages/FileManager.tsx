import { useState, useEffect } from 'react';
import { FolderOpen, File, ChevronRight, Upload, Plus, Trash2, Edit3, ArrowLeft, Loader, X } from 'lucide-react';
import { fileService } from '../services/files';
import { useToast } from '../components/ui/ToastProvider';

interface FileItem {
    name: string; path: string; is_dir: boolean;
    size: number; permissions: string; modified_at: string;
}

function formatSize(bytes: number): string {
    if (bytes === 0) return '—';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export default function FileManager() {
    const toast = useToast();
    const [files, setFiles] = useState<FileItem[]>([]);
    const [currentPath, setCurrentPath] = useState('/var/www');
    const [loading, setLoading] = useState(true);
    const [selectedFile, setSelectedFile] = useState<FileItem | null>(null);
    const [showEditor, setShowEditor] = useState(false);
    const [editorContent, setEditorContent] = useState('');
    const [saving, setSaving] = useState(false);

    const DEMO_FILES: FileItem[] = [
        { name: 'public_html', path: '/var/www/public_html', is_dir: true, size: 0, permissions: 'drwxr-xr-x', modified_at: '2026-03-14 10:30' },
        { name: 'logs', path: '/var/www/logs', is_dir: true, size: 0, permissions: 'drwxr-xr-x', modified_at: '2026-03-14 02:00' },
        { name: 'ssl', path: '/var/www/ssl', is_dir: true, size: 0, permissions: 'drwx------', modified_at: '2026-03-10 15:20' },
        { name: 'backups', path: '/var/www/backups', is_dir: true, size: 0, permissions: 'drwxr-xr-x', modified_at: '2026-03-13 02:00' },
        { name: 'index.html', path: '/var/www/index.html', is_dir: false, size: 4520, permissions: '-rw-r--r--', modified_at: '2026-03-14 12:00' },
        { name: '.htaccess', path: '/var/www/.htaccess', is_dir: false, size: 234, permissions: '-rw-r--r--', modified_at: '2026-03-10 09:15' },
        { name: 'wp-config.php', path: '/var/www/wp-config.php', is_dir: false, size: 3890, permissions: '-rw-r-----', modified_at: '2026-03-12 14:30' },
        { name: 'robots.txt', path: '/var/www/robots.txt', is_dir: false, size: 78, permissions: '-rw-r--r--', modified_at: '2026-03-08 11:00' },
    ];

    const fetchFiles = async (path: string) => {
        setLoading(true);
        try {
            const res = await fileService.list(path);
            const items = Array.isArray(res) ? res : res?.files || res?.data || [];
            setFiles(items.length > 0 ? items : DEMO_FILES);
        } catch {
            setFiles(DEMO_FILES);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { fetchFiles(currentPath); }, [currentPath]);

    const navigateTo = (path: string) => {
        setCurrentPath(path);
    };

    const openFile = async (file: FileItem) => {
        if (file.is_dir) {
            navigateTo(file.path);
        } else {
            setSelectedFile(file);
            try {
                const res = await fileService.read(file.path);
                setEditorContent(res?.content || `// Content of ${file.name}`);
            } catch {
                setEditorContent(`<!-- Content of ${file.name} -->\n<!DOCTYPE html>\n<html>\n<head>\n  <title>NovaPanel</title>\n</head>\n<body>\n  <h1>Hello from NovaPanel!</h1>\n</body>\n</html>`);
            }
            setShowEditor(true);
        }
    };

    const handleSave = async () => {
        if (!selectedFile) return;
        setSaving(true);
        try {
            await fileService.write(selectedFile.path, editorContent);
            toast.success(`Saved ${selectedFile.name}`);
        } catch {
            toast.error('Failed to save file');
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (file: FileItem, e: React.MouseEvent) => {
        e.stopPropagation();
        if (!confirm(`Delete "${file.name}"?`)) return;
        try {
            await fileService.remove(file.path);
            toast.success(`Deleted ${file.name}`);
            fetchFiles(currentPath);
        } catch {
            toast.error('Failed to delete file');
        }
    };

    const goUp = () => {
        const parent = currentPath.split('/').slice(0, -1).join('/') || '/';
        setCurrentPath(parent);
    };

    const dirs = files.filter(f => f.is_dir).sort((a, b) => a.name.localeCompare(b.name));
    const regularFiles = files.filter(f => !f.is_dir).sort((a, b) => a.name.localeCompare(b.name));
    const sorted = [...dirs, ...regularFiles];

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <FolderOpen className="w-7 h-7 text-nova-400" />
                        File Manager
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Browse, edit & manage files on your server</p>
                </div>
                <div className="flex items-center gap-2">
                    <button className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-all">
                        <Plus className="w-4 h-4" /> New File
                    </button>
                    <button className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                        <Upload className="w-4 h-4" /> Upload
                    </button>
                </div>
            </div>

            {/* Breadcrumb */}
            <div className="flex items-center gap-1 text-sm glass-panel rounded-xl px-4 py-2.5">
                {currentPath !== '/' && (
                    <button onClick={goUp} className="p-1 rounded hover:bg-white/5 text-surface-200/50 mr-1">
                        <ArrowLeft className="w-3.5 h-3.5" />
                    </button>
                )}
                {currentPath.split('/').filter(Boolean).map((seg, i, arr) => (
                    <span key={i} className="flex items-center gap-1">
                        {i > 0 && <ChevronRight className="w-3 h-3 text-surface-200/30" />}
                        <button
                            onClick={() => navigateTo('/' + arr.slice(0, i + 1).join('/'))}
                            className={`hover:text-nova-400 transition-colors ${i === arr.length - 1 ? 'text-white font-medium' : 'text-surface-200/50'}`}>
                            {seg}
                        </button>
                    </span>
                ))}
            </div>

            {/* File List */}
            <div className="glass-card rounded-2xl overflow-hidden">
                {loading ? (
                    <div className="flex items-center justify-center py-16">
                        <Loader className="w-6 h-6 text-nova-500 animate-spin" />
                    </div>
                ) : (
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-white/5">
                                <th className="text-left py-3 px-5 text-xs font-semibold text-surface-200/50 uppercase tracking-wider">Name</th>
                                <th className="text-left py-3 px-5 text-xs font-semibold text-surface-200/50 uppercase tracking-wider">Size</th>
                                <th className="text-left py-3 px-5 text-xs font-semibold text-surface-200/50 uppercase tracking-wider">Permissions</th>
                                <th className="text-left py-3 px-5 text-xs font-semibold text-surface-200/50 uppercase tracking-wider">Modified</th>
                                <th className="py-3 px-5"></th>
                            </tr>
                        </thead>
                        <tbody>
                            {sorted.map((file, i) => (
                                <tr key={file.path} className="border-b border-white/5 hover:bg-white/[0.02] cursor-pointer transition-colors"
                                    onClick={() => openFile(file)} style={{ animationDelay: `${i * 30}ms` }}>
                                    <td className="py-3 px-5">
                                        <div className="flex items-center gap-3">
                                            {file.is_dir ? <FolderOpen className="w-4 h-4 text-warning" /> : <File className="w-4 h-4 text-surface-200/40" />}
                                            <span className={`text-sm font-medium ${file.is_dir ? 'text-white' : 'text-surface-200/80'}`}>{file.name}</span>
                                        </div>
                                    </td>
                                    <td className="py-3 px-5 text-sm text-surface-200/50">{formatSize(file.size)}</td>
                                    <td className="py-3 px-5 text-sm text-surface-200/40 font-mono text-xs">{file.permissions}</td>
                                    <td className="py-3 px-5 text-sm text-surface-200/40">{file.modified_at}</td>
                                    <td className="py-3 px-5 text-right">
                                        <div className="flex items-center gap-1 justify-end">
                                            {!file.is_dir && (
                                                <button onClick={e => { e.stopPropagation(); openFile(file); }}
                                                    className="p-1.5 rounded-lg hover:bg-nova-500/10 text-surface-200/30 hover:text-nova-400 transition-colors">
                                                    <Edit3 className="w-3.5 h-3.5" />
                                                </button>
                                            )}
                                            <button onClick={e => handleDelete(file, e)}
                                                className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/30 hover:text-danger transition-colors">
                                                <Trash2 className="w-3.5 h-3.5" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>

            {/* Code Editor Modal */}
            {showEditor && selectedFile && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={() => setShowEditor(false)}>
                    <div className="glass-card rounded-2xl w-full max-w-3xl mx-4 max-h-[80vh] flex flex-col animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between px-6 py-4 border-b border-white/5">
                            <div className="flex items-center gap-3">
                                <button onClick={() => setShowEditor(false)} className="p-1 rounded-lg hover:bg-white/5 text-surface-200/50">
                                    <ArrowLeft className="w-4 h-4" />
                                </button>
                                <File className="w-4 h-4 text-surface-200/40" />
                                <span className="text-sm font-medium text-white">{selectedFile.name}</span>
                                <span className="text-xs text-surface-200/30">{selectedFile.path}</span>
                            </div>
                            <button onClick={handleSave} disabled={saving}
                                className="px-4 py-1.5 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-xs font-medium disabled:opacity-50 flex items-center gap-1">
                                {saving ? <Loader className="w-3 h-3 animate-spin" /> : null}
                                {saving ? 'Saving...' : 'Save'}
                            </button>
                        </div>
                        <div className="flex-1 overflow-auto p-0">
                            <textarea value={editorContent} onChange={e => setEditorContent(e.target.value)}
                                className="w-full h-[60vh] bg-transparent text-surface-200/80 font-mono text-sm p-6 focus:outline-none resize-none leading-6"
                                spellCheck={false} />
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
