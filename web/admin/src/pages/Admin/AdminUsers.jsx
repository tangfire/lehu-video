import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiAlertCircle, FiAward, FiClock, FiMessageCircle, FiRefreshCw, FiSearch, FiShield, FiUserCheck, FiUsers } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber, roleText } from './adminUtils';
import './Admin.css';

const pageSize = 20;

const roleOptions = [
    ['全部角色', ''],
    ['普通用户', 'user'],
    ['运营', 'operator'],
    ['管理员', 'admin'],
];

const authOptions = [
    ['全部认证', '-1'],
    ['未认证', '0'],
    ['已认证', '1'],
];

const AdminUsers = () => {
    const [users, setUsers] = useState([]);
    const [filters, setFilters] = useState({ keyword: '', role: '', authStatus: '-1' });
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const load = async (nextPage = page, nextFilters = filters) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listUsers({
                page: nextPage,
                size: pageSize,
                keyword: nextFilters.keyword,
                role: nextFilters.role,
                auth_status: nextFilters.authStatus,
            });
            setUsers(data.users || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取用户失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load(1);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const stats = useMemo(() => {
        const active = users.filter((item) => item.last_active_at).length;
        const operators = users.filter((item) => item.role === 'operator' || item.role === 'admin').length;
        const verified = users.filter((item) => Number(item.profile?.auth_status || 0) === 1).length;
        const risky = users.filter((item) => Number(item.report_count || 0) > 0 || Number(item.feedback_count || 0) > 0).length;
        return [
            { label: '当前页用户', value: users.length, hint: `共 ${compactNumber(total)} 人`, icon: <FiUsers /> },
            { label: '有活跃记录', value: active, hint: '最近一次请求/IP 可追踪', icon: <FiClock /> },
            { label: '后台成员', value: operators, hint: '运营 + 管理员', icon: <FiShield /> },
            { label: '已认证', value: verified, hint: '当前页认证状态', icon: <FiUserCheck /> },
            { label: '需关注', value: risky, hint: '有反馈或举报记录', icon: <FiAlertCircle /> },
        ];
    }, [users, total]);

    const updateFilter = (key, value) => {
        setFilters((prev) => ({ ...prev, [key]: value }));
    };

    const submitFilters = () => load(1);

    const resetFilters = () => {
        const next = { keyword: '', role: '', authStatus: '-1' };
        setFilters(next);
        load(1, next);
    };

    return (
        <>
            <section className="admin-users-head">
                <div>
                    <span className="admin-kicker">USER OPS</span>
                    <h2>用户工作台</h2>
                    <p>用来识别活跃同学、内容贡献和需要跟进的反馈/举报用户。角色调整请去权限管理。</p>
                </div>
                <div className="admin-head-actions">
                    <Link className="admin-button" to="/admin/permissions">
                        <FiShield />
                        权限管理
                    </Link>
                    <button className="admin-button" onClick={() => load(page)} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} /> 刷新
                    </button>
                </div>
            </section>

            <section className="admin-user-stat-grid">
                {stats.map((item) => (
                    <div className="admin-user-stat" key={item.label}>
                        <span>{item.icon}</span>
                        <div>
                            <em>{item.label}</em>
                            <strong>{item.value}</strong>
                            <small>{item.hint}</small>
                        </div>
                    </div>
                ))}
            </section>

            <div className="admin-user-filter">
                <div className="admin-search-box">
                    <FiSearch />
                    <input
                        value={filters.keyword}
                        onChange={(e) => updateFilter('keyword', e.target.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') submitFilters(); }}
                        placeholder="搜索昵称、姓名、学号、手机号、邮箱"
                    />
                </div>
                <select className="admin-select" value={filters.role} onChange={(e) => updateFilter('role', e.target.value)}>
                    {roleOptions.map(([label, value]) => <option value={value} key={value}>{label}</option>)}
                </select>
                <select className="admin-select" value={filters.authStatus} onChange={(e) => updateFilter('authStatus', e.target.value)}>
                    {authOptions.map(([label, value]) => <option value={value} key={value}>{label}</option>)}
                </select>
                <button className="admin-button primary" onClick={submitFilters}>查询</button>
                <button className="admin-button" onClick={resetFilters}>重置</button>
            </div>
            {error && <div className="admin-error">{error}</div>}
            {loading && <div className="admin-loading">用户加载中...</div>}
            {!loading && users.length === 0 && (
                <div className="admin-empty">
                    <FiUsers />
                    <span>暂无匹配用户</span>
                </div>
            )}
            {!loading && users.length > 0 && (
                <div className="admin-user-list">
                    {users.map((item) => (
                        <UserCard item={item} key={item.user.id} />
                    ))}
                </div>
            )}
            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条</span>
                <button className="admin-button" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={page * pageSize >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>
        </>
    );
};

const UserCard = ({ item }) => {
    const user = item.user || {};
    const profile = item.profile || {};
    const name = user.nickname || user.name || profile.real_name || '深汕同学';
    const role = item.role || 'user';
    const authStatus = Number(profile.auth_status || 0);
    const activeLabel = item.last_active_at ? `${item.last_active_at} · ${item.last_active_ip || '未知 IP'}` : '暂无活跃记录';
    const profileLines = [
        profile.real_name && `姓名：${profile.real_name}`,
        profile.student_no && `学号：${profile.student_no}`,
        profile.class_name && `班级：${profile.class_name}`,
        (profile.dorm_building || profile.room_no) && `宿舍：${profile.dorm_building || ''}${profile.room_no || ''}`,
    ].filter(Boolean);

    return (
        <article className="admin-user-card">
            <div className="admin-user-main">
                <div className="admin-user-avatar">
                    {user.avatar ? <img src={user.avatar} alt="" /> : <span>{name.slice(0, 1).toUpperCase()}</span>}
                </div>
                <div className="admin-user-info">
                    <div className="admin-user-title">
                        <h3>{name}</h3>
                        <span className={`admin-role-badge role-${role}`}>{roleText(role)}</span>
                        <span className={`admin-status-pill status-${authStatus}`}>{authStatus === 1 ? '已认证' : '未认证'}</span>
                    </div>
                    <div className="admin-user-sub">#{user.id} · {user.mobile || profile.mobile || user.email || '无联系方式'}</div>
                    <div className="admin-user-profile">
                        {profileLines.length ? profileLines.map((line) => <span key={line}>{line}</span>) : <span>校园资料暂未填写</span>}
                    </div>
                </div>
            </div>

            <div className="admin-user-metrics">
                <Metric label="发帖" value={item.post_count} />
                <Metric label="评论" value={item.comment_count} />
                <Metric label="点赞" value={item.like_count} />
                <Metric label="收藏" value={item.collection_count} />
                <Metric label="登录" value={item.login_count} />
                <Metric label="访问" value={item.visit_count} />
            </div>

            <div className="admin-user-side">
                <div className="admin-user-activity">
                    <strong>最近活跃</strong>
                    <span>{activeLabel}</span>
                    {item.last_active_path && <em>{item.last_active_path} · HTTP {item.last_active_status || 0}</em>}
                </div>
                <div className="admin-user-risk">
                    <span><FiMessageCircle /> 反馈 {compactNumber(item.feedback_count || 0)}</span>
                    <span><FiAlertCircle /> 举报 {compactNumber(item.report_count || 0)}</span>
                    {item.last_login_at && <span><FiAward /> 最近登录 {item.last_login_at}</span>}
                </div>
                <div className="admin-user-note">
                    <FiShield />
                    <span>当前角色：{roleText(role)}。权限分配已移到独立模块，避免在用户查看时误操作。</span>
                </div>
            </div>
        </article>
    );
};

const Metric = ({ label, value }) => (
    <div>
        <strong>{compactNumber(value || 0)}</strong>
        <span>{label}</span>
    </div>
);

export default AdminUsers;
