import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Terminal as TerminalIcon, ArrowLeft, Wifi, WifiOff, Maximize2, Minimize2 } from 'lucide-react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';

export default function TerminalPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const termRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const fitRef = useRef<FitAddon | null>(null);
    const [connected, setConnected] = useState(false);
    const [serverName, setServerName] = useState('');
    const [fullscreen, setFullscreen] = useState(false);

    useEffect(() => {
        if (!termRef.current || !id) return;

        // Create xterm instance
        const term = new XTerm({
            cursorBlink: true,
            fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
            fontSize: 14,
            lineHeight: 1.2,
            theme: {
                background: '#0a0e1a',
                foreground: '#e4e8f0',
                cursor: '#a78bfa',
                cursorAccent: '#0a0e1a',
                selectionBackground: 'rgba(167, 139, 250, 0.3)',
                black: '#1a1e2e',
                red: '#f87171',
                green: '#4ade80',
                yellow: '#fbbf24',
                blue: '#60a5fa',
                magenta: '#c084fc',
                cyan: '#22d3ee',
                white: '#e4e8f0',
                brightBlack: '#4b5563',
                brightRed: '#fca5a5',
                brightGreen: '#86efac',
                brightYellow: '#fde68a',
                brightBlue: '#93c5fd',
                brightMagenta: '#d8b4fe',
                brightCyan: '#67e8f9',
                brightWhite: '#f9fafb',
            },
        });

        const fit = new FitAddon();
        term.loadAddon(fit);
        term.open(termRef.current);
        fit.fit();

        xtermRef.current = term;
        fitRef.current = fit;

        term.writeln('\x1b[1;36m╔═══════════════════════════════════════╗\x1b[0m');
        term.writeln('\x1b[1;36m║   \x1b[1;37mNovaPanel Web Terminal\x1b[1;36m              ║\x1b[0m');
        term.writeln('\x1b[1;36m╚═══════════════════════════════════════╝\x1b[0m');
        term.writeln('');
        term.writeln('\x1b[33mConnecting to server...\x1b[0m');

        // Connect WebSocket
        const token = localStorage.getItem('novapanel_token');
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/api/v1/servers/${id}/terminal?token=${token}`;

        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
            setConnected(true);
            term.writeln('\x1b[32m✓ Connected!\x1b[0m');
            term.writeln('');

            // Send initial resize
            const dims = fit.proposeDimensions();
            if (dims) {
                ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
            }
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'output' && msg.data) {
                    term.write(msg.data);
                }
            } catch {
                // Raw text
                term.write(event.data);
            }
        };

        ws.onclose = () => {
            setConnected(false);
            term.writeln('');
            term.writeln('\x1b[31m✕ Disconnected from server\x1b[0m');
        };

        ws.onerror = () => {
            setConnected(false);
            term.writeln('\x1b[31m✕ Connection error\x1b[0m');
        };

        // Terminal input → WebSocket
        term.onData((data) => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'input', data }));
            }
        });

        // Handle window resize
        const handleResize = () => {
            fit.fit();
            const dims = fit.proposeDimensions();
            if (dims && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
            }
        };
        window.addEventListener('resize', handleResize);

        // Fetch server name
        fetch(`/api/v1/servers/${id}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        }).then(r => r.json()).then((data: { name?: string }) => {
            if (data.name) setServerName(data.name);
        }).catch(() => { });

        return () => {
            window.removeEventListener('resize', handleResize);
            ws.close();
            term.dispose();
        };
    }, [id]);

    // Re-fit on fullscreen toggle
    useEffect(() => {
        if (fitRef.current) {
            setTimeout(() => fitRef.current?.fit(), 100);
        }
    }, [fullscreen]);

    return (
        <div className={`${fullscreen ? 'fixed inset-0 z-[60]' : ''} flex flex-col h-full`}
            style={{ background: '#0a0e1a' }}>
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-2"
                style={{ background: 'linear-gradient(135deg, #1a1e2e 0%, #0f1320 100%)', borderBottom: '1px solid rgba(139, 92, 246, 0.2)' }}>
                <div className="flex items-center gap-3">
                    <button onClick={() => navigate('/servers')}
                        className="p-1.5 rounded-lg transition-all hover:bg-white/5">
                        <ArrowLeft className="w-4 h-4 text-gray-400" />
                    </button>
                    <TerminalIcon className="w-5 h-5 text-violet-400" />
                    <div>
                        <h2 className="text-sm font-semibold text-white">{serverName || 'Server Terminal'}</h2>
                        <div className="flex items-center gap-1.5">
                            {connected ? (
                                <><Wifi className="w-3 h-3 text-emerald-400" /><span className="text-xs text-emerald-400">Connected</span></>
                            ) : (
                                <><WifiOff className="w-3 h-3 text-red-400" /><span className="text-xs text-red-400">Disconnected</span></>
                            )}
                        </div>
                    </div>
                </div>
                <button onClick={() => setFullscreen(!fullscreen)}
                    className="p-1.5 rounded-lg transition-all hover:bg-white/5">
                    {fullscreen ? <Minimize2 className="w-4 h-4 text-gray-400" /> : <Maximize2 className="w-4 h-4 text-gray-400" />}
                </button>
            </div>
            {/* Terminal */}
            <div ref={termRef} className="flex-1 p-2" style={{ background: '#0a0e1a' }} />
        </div>
    );
}
