import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Header from './Header';

export default function MainLayout() {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

    return (
        <div className="min-h-screen bg-surface-900 text-white">
            <Sidebar
                collapsed={sidebarCollapsed}
                onToggle={() => setSidebarCollapsed(c => !c)}
            />
            <div
                className="transition-all duration-300 min-h-screen flex flex-col"
                style={{ marginLeft: sidebarCollapsed ? '68px' : '250px' }}
            >
                <Header />
                <main className="flex-1 p-6 animate-fade-in">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
