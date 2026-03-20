import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Header from './Header';

export default function MainLayout() {
    return (
        <div className="min-h-screen bg-surface-900 text-white">
            <Sidebar />
            <div className="ml-[250px] transition-all duration-300">
                <Header />
                <main className="p-6 animate-fade-in">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
