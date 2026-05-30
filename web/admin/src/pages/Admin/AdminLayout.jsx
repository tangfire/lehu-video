import { useState } from 'react';
import { NavLink, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { FiBarChart2, FiBell, FiChevronDown, FiCpu, FiEdit3, FiFileText, FiGrid, FiKey, FiMessageCircle, FiShield, FiUsers } from 'react-icons/fi';
import { clearUserData, getCurrentUser, isLoggedIn } from '../../api/user';
import './Admin.css';

const navItems = [
    { to: '/admin', label: '数据总览', end: true, icon: <FiBarChart2 /> },
    { to: '/admin/posts', label: '内容工作台', icon: <FiFileText /> },
    { to: '/admin/compose', label: '运营发帖', icon: <FiEdit3 /> },
    { to: '/admin/moments', label: '朋友圈素材', icon: <FiGrid /> },
    { to: '/admin/moderation', label: '反馈与举报', icon: <FiBell /> },
    { to: '/admin/audit', label: '审核设置', icon: <FiShield /> },
    { to: '/admin/assistant', label: 'e仔助手', icon: <FiCpu /> },
    { to: '/admin/ai-replies', label: 'e仔回复', icon: <FiMessageCircle /> },
];

const advancedItems = [
    { to: '/admin/notifications', label: '系统通知', icon: <FiBell /> },
    { to: '/admin/security', label: '安全中心', icon: <FiShield /> },
    { to: '/admin/users', label: '用户管理', icon: <FiUsers /> },
    { to: '/admin/permissions', label: '权限管理', icon: <FiKey /> },
];

const navGroups = [
    { title: '日常运营', items: navItems },
    { title: '高级工具', items: advancedItems, advanced: true },
];

const titleMap = {
    '/admin': '数据总览',
    '/admin/posts': '内容工作台',
    '/admin/compose': '运营发帖',
    '/admin/moments': '朋友圈素材',
    '/admin/moderation': '反馈与举报',
    '/admin/audit': '审核设置',
    '/admin/assistant': 'e仔助手',
    '/admin/notifications': '通知中心',
    '/admin/ai-replies': 'e仔回复',
    '/admin/knowledge': 'e仔知识库',
    '/admin/comments': '评论管理',
    '/admin/reports': '举报处理',
    '/admin/feedback': '用户反馈',
    '/admin/security': '安全中心',
    '/admin/users': '用户管理',
    '/admin/permissions': '权限管理',
};

const subtitleMap = {
    '/admin': '待办、流量、互动、风险。',
    '/admin/posts': '置顶、精选、审核、下架。',
    '/admin/compose': '官方攻略、问答和公告。',
    '/admin/moments': '今日热帖九图素材。',
    '/admin/moderation': '举报、反馈、评论。',
    '/admin/audit': '不审、人工审、AI 初审。',
    '/admin/assistant': '人设、知识库、回复任务。',
    '/admin/notifications': '内测公告、维护提醒。',
    '/admin/ai-replies': '@e仔 回复任务。',
    '/admin/knowledge': '资料、片段、命中测试。',
    '/admin/comments': '评论可见性和来源。',
    '/admin/reports': '举报对象和处理状态。',
    '/admin/feedback': '用户问题和跟进状态。',
    '/admin/security': '请求、限流、异常 IP。',
    '/admin/users': '用户画像和风险记录。',
    '/admin/permissions': '后台角色和权限。',
};

const displayOperatorName = (user) => {
    const value = (user?.nickname || user?.name || '').trim();
    if (!value) return '运营同学';
    return /[ÃÂÆæÅåÇç]/.test(value) ? 'Admin' : value;
};

const AdminLayout = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const user = getCurrentUser();
    const advancedActive = advancedItems.some((item) => location.pathname.startsWith(item.to));
    const [advancedOpen, setAdvancedOpen] = useState(advancedActive);

    if (!isLoggedIn()) {
        return <Navigate to="/admin/login" replace />;
    }

    const logout = () => {
        clearUserData();
        navigate('/admin/login', { replace: true });
    };

    return (
        <div className="admin-shell">
            <div className="admin-layout">
                <aside className="admin-sidebar">
                    <div className="admin-brand">
                        <div className="admin-brand-mark">e</div>
                        <div className="admin-brand-title">深汕校园e站</div>
                        <div className="admin-brand-subtitle">运营控制台</div>
                    </div>
                    <nav className="admin-nav">
                        {navGroups.map((group) => {
                            if (group.advanced) {
                                return (
                                    <div className="admin-nav-group advanced" key={group.title}>
                                        <button
                                            className={`admin-nav-more ${advancedActive ? 'active' : ''}`}
                                            type="button"
                                            onClick={() => setAdvancedOpen((value) => !value)}
                                        >
                                            <span>{group.title}</span>
                                            <FiChevronDown className={advancedOpen ? 'rotate' : ''} />
                                        </button>
                                        {(advancedOpen || advancedActive) && group.items.map((item) => (
                                            <NavLink key={item.to} to={item.to} end={item.end}>
                                                {item.icon}
                                                {item.label}
                                            </NavLink>
                                        ))}
                                    </div>
                                );
                            }
                            return (
                                <div className="admin-nav-group" key={group.title}>
                                <span className="admin-nav-group-title">{group.title}</span>
                                {group.items.map((item) => (
                                    <NavLink key={item.to} to={item.to} end={item.end}>
                                        {item.icon}
                                        {item.label}
                                    </NavLink>
                                ))}
                                </div>
                            );
                        })}
                    </nav>
                </aside>
                <main className="admin-main">
                    <header className="admin-topbar">
                        <div>
                            <h1>{titleMap[location.pathname] || '后台管理'}</h1>
                            <p>{subtitleMap[location.pathname] || '深汕e仔官方内容、社区秩序和增长数据都在这里处理。'}</p>
                        </div>
                        <div className="admin-user">
                            <span>{displayOperatorName(user)}</span>
                            <button className="admin-button" onClick={logout}>退出</button>
                        </div>
                    </header>
                    <div className="admin-content">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    );
};

export default AdminLayout;
