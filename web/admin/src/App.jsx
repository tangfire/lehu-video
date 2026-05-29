import { BrowserRouter as Router, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import AdminLogin from './pages/Admin/AdminLogin.jsx';
import AdminLayout from './pages/Admin/AdminLayout.jsx';
import AdminDashboard from './pages/Admin/AdminDashboard.jsx';
import AdminPosts from './pages/Admin/AdminPosts.jsx';
import AdminCompose from './pages/Admin/AdminCompose.jsx';
import AdminUsers from './pages/Admin/AdminUsers.jsx';
import AdminSecurity from './pages/Admin/AdminSecurity.jsx';
import AdminPermissions from './pages/Admin/AdminPermissions.jsx';
import AdminNotifications from './pages/Admin/AdminNotifications.jsx';
import AdminModeration from './pages/Admin/AdminModeration.jsx';
import AdminAssistant from './pages/Admin/AdminAssistant.jsx';
import AdminAuditSettings from './pages/Admin/AdminAuditSettings.jsx';
import './App.css';

function AdminLegacyRedirect({ to, tab }) {
    const location = useLocation();
    const params = new URLSearchParams(location.search);
    params.set('tab', tab);
    return <Navigate to={`${to}?${params.toString()}`} replace />;
}

function App() {
    return (
        <Router>
            <Routes>
                <Route path="/" element={<Navigate to="/admin" replace />} />
                <Route path="/admin/login" element={<AdminLogin />} />
                <Route path="/admin" element={<AdminLayout />}>
                    <Route index element={<AdminDashboard />} />
                    <Route path="posts" element={<AdminPosts />} />
                    <Route path="compose" element={<AdminCompose />} />
                    <Route path="moderation" element={<AdminModeration />} />
                    <Route path="audit" element={<AdminAuditSettings />} />
                    <Route path="assistant" element={<AdminAssistant />} />
                    <Route path="notifications" element={<AdminNotifications />} />
                    <Route path="ai-replies" element={<AdminLegacyRedirect to="/admin/assistant" tab="status" />} />
                    <Route path="knowledge" element={<AdminLegacyRedirect to="/admin/assistant" tab="knowledge" />} />
                    <Route path="comments" element={<AdminLegacyRedirect to="/admin/moderation" tab="comments" />} />
                    <Route path="reports" element={<AdminLegacyRedirect to="/admin/moderation" tab="reports" />} />
                    <Route path="feedback" element={<AdminLegacyRedirect to="/admin/moderation" tab="feedback" />} />
                    <Route path="security" element={<AdminSecurity />} />
                    <Route path="users" element={<AdminUsers />} />
                    <Route path="permissions" element={<AdminPermissions />} />
                </Route>
                <Route path="*" element={<Navigate to="/admin" replace />} />
            </Routes>
        </Router>
    );
}

export default App;
