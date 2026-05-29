import { useState } from 'react';
import { NavLink, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { FiBarChart2, FiBell, FiChevronDown, FiCpu, FiEdit3, FiFileText, FiKey, FiShield, FiUsers } from 'react-icons/fi';
import { clearUserData, getCurrentUser, isLoggedIn } from '../../api/user';
import './Admin.css';

const navItems = [
    { to: '/admin', label: '数据总览', end: true, icon: <FiBarChart2 /> },
    { to: '/admin/posts', label: '内容工作台', icon: <FiFileText /> },
    { to: '/admin/compose', label: '运营发帖', icon: <FiEdit3 /> },
    { to: '/admin/moderation', label: '反馈与举报', icon: <FiBell /> },
    { to: '/admin/audit', label: '审核设置', icon: <FiShield /> },
    { to: '/admin/assistant', label: 'e仔助手', icon: <FiCpu /> },
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
    '/admin': '先看待办、流量、互动和风险，再决定今天要发什么内容。',
    '/admin/posts': '管理首页内容流，处理置顶、精选、下架和内容状态。',
    '/admin/compose': '用深汕e仔发布官方攻略、问答和新生内容。',
    '/admin/moderation': '集中处理举报、用户反馈和评论状态，先清待办再看历史。',
    '/admin/audit': '设置普通用户发帖后是否直接展示、进入人工审核或交给 AI 初审。',
    '/admin/assistant': '维护 e仔资料、查看自动回复状态，并测试知识库命中效果。',
    '/admin/notifications': '给小程序消息中心发送内测公告、维护提醒和活动通知。',
    '/admin/ai-replies': '查看 @深汕e仔 自动回复链路是否启用、是否失败，并手动重试。',
    '/admin/knowledge': '维护 e仔回答校园问题时可引用的资料，上传文档、录入信息并测试命中效果。',
    '/admin/comments': '处理评论可见性，查看评论来源和关联原帖。',
    '/admin/reports': '按举报对象处理内容风险，必要时下架帖子或评论。',
    '/admin/feedback': '收集用户建议和问题反馈，标记跟进状态。',
    '/admin/security': '查看请求、限流、异常 IP 和接口访问情况。',
    '/admin/users': '查看用户画像、活跃、内容贡献和风险记录。',
    '/admin/permissions': '集中分配运营和管理员权限，避免在用户管理里误操作。',
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
                            <span>{user?.nickname || user?.name || '运营同学'}</span>
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
