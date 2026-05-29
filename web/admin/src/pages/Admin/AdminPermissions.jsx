import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiRefreshCw, FiSearch, FiShield, FiUserCheck, FiUsers } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { roleText } from './adminUtils';
import './Admin.css';

const pageSize = 20;

const roleOptions = [
    ['全部角色', ''],
    ['普通用户', 'user'],
    ['运营', 'operator'],
    ['管理员', 'admin'],
];

const roleActions = [
    ['operator', '升级为运营', '可进入后台处理内容、评论、举报和反馈'],
    ['admin', '升级为管理员', '可调整其他用户后台权限，建议只给核心成员'],
    ['user', '移除后台权限', '恢复为普通用户，不能再进入运营后台'],
];

const AdminPermissions = () => {
    const [users, setUsers] = useState([]);
    const [filters, setFilters] = useState({ keyword: '', role: '' });
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [confirmAction, setConfirmAction] = useState(null);
    const [actionLoading, setActionLoading] = useState(false);

    const load = async (nextPage = page, nextFilters = filters) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listUsers({
                page: nextPage,
                size: pageSize,
                keyword: nextFilters.keyword,
                role: nextFilters.role,
                auth_status: -1,
            });
            setUsers(data.users || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取权限列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load(1);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const counts = useMemo(() => {
        const admins = users.filter((item) => item.role === 'admin').length;
        const operators = users.filter((item) => item.role === 'operator').length;
        const normal = users.filter((item) => !item.role || item.role === 'user').length;
        return { admins, operators, normal };
    }, [users]);

    const updateFilter = (key, value) => {
        setFilters((prev) => ({ ...prev, [key]: value }));
    };

    const submitFilters = () => load(1);

    const resetFilters = () => {
        const next = { keyword: '', role: '' };
        setFilters(next);
        load(1, next);
    };

    const updateRole = async (item, role) => {
        if (actionLoading) return;
        setActionLoading(true);
        setError('');
        try {
            await campusAdminApi.updateUserRole(item.user.id, role);
            setMessage(`已将 ${displayName(item)} 更新为${roleText(role)}`);
            window.setTimeout(() => setMessage(''), 2600);
            setConfirmAction(null);
            load(page);
        } catch (err) {
            setError(err.message || '更新权限失败');
        } finally {
            setActionLoading(false);
        }
    };

    return (
        <div className="admin-permission-page">
            {message && <div className="admin-toast success">{message}</div>}
            <section className="admin-users-head permission">
                <div>
                    <span className="admin-kicker">ACCESS CONTROL</span>
                    <h2>权限管理</h2>
                    <p>只在这里分配运营和管理员权限。用户画像、活跃和内容贡献请回到用户管理查看。</p>
                </div>
                <button className="admin-button" onClick={() => load(page)} disabled={loading}>
                    <FiRefreshCw className={loading ? 'spin' : ''} /> 刷新
                </button>
            </section>

            <section className="admin-permission-warning">
                <FiAlertTriangle />
                <div>
                    <strong>谨慎授予管理员权限</strong>
                    <span>管理员可以继续分配权限；运营只负责内容、反馈、举报等日常工作。正式上线前必须关闭本地开发的全员管理员开关。</span>
                </div>
            </section>

            <section className="admin-user-stat-grid permission">
                <div className="admin-user-stat">
                    <span><FiShield /></span>
                    <div><em>本页管理员</em><strong>{counts.admins}</strong><small>最高权限</small></div>
                </div>
                <div className="admin-user-stat">
                    <span><FiUserCheck /></span>
                    <div><em>本页运营</em><strong>{counts.operators}</strong><small>日常后台权限</small></div>
                </div>
                <div className="admin-user-stat">
                    <span><FiUsers /></span>
                    <div><em>本页普通用户</em><strong>{counts.normal}</strong><small>无后台权限</small></div>
                </div>
            </section>

            <div className="admin-user-filter permission">
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
                <button className="admin-button primary" onClick={submitFilters}>查询</button>
                <button className="admin-button" onClick={resetFilters}>重置</button>
            </div>

            {error && <div className="admin-error">{error}</div>}
            {loading && <div className="admin-loading">权限列表加载中...</div>}
            {!loading && users.length === 0 && <div className="admin-empty"><FiShield /><span>暂无匹配用户</span></div>}

            {!loading && users.length > 0 && (
                <div className="admin-permission-list">
                    {users.map((item) => (
                        <PermissionRow item={item} key={item.user.id} actionLoading={actionLoading} onRole={(role) => setConfirmAction({ item, role })} />
                    ))}
                </div>
            )}

            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条</span>
                <button className="admin-button" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={page * pageSize >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>

            {confirmAction && (
                <RoleConfirmModal
                    item={confirmAction.item}
                    role={confirmAction.role}
                    loading={actionLoading}
                    onCancel={() => setConfirmAction(null)}
                    onConfirm={() => updateRole(confirmAction.item, confirmAction.role)}
                />
            )}
        </div>
    );
};

const PermissionRow = ({ item, actionLoading, onRole }) => {
    const user = item.user || {};
    const role = item.role || 'user';
    const name = displayName(item);
    const contact = user.mobile || item.profile?.mobile || user.email || '无联系方式';
    return (
        <article className="admin-permission-row">
            <div className="admin-user-avatar">
                {user.avatar ? <img src={user.avatar} alt="" /> : <span>{name.slice(0, 1).toUpperCase()}</span>}
            </div>
            <div className="admin-permission-info">
                <div>
                    <h3>{name}</h3>
                    <span className={`admin-role-badge role-${role}`}>{roleText(role)}</span>
                </div>
                <p>#{user.id} · {contact}</p>
                <p>{item.profile?.real_name || '未填写实名'} {item.profile?.student_no || ''}</p>
            </div>
            <div className="admin-permission-actions">
                {roleActions.map(([nextRole, label, title]) => (
                    <button
                        className={`admin-button ${nextRole === 'user' ? 'danger' : ''}`}
                        disabled={actionLoading || nextRole === role}
                        title={title}
                        key={nextRole}
                        onClick={() => onRole(nextRole)}
                    >
                        {label}
                    </button>
                ))}
            </div>
        </article>
    );
};

const RoleConfirmModal = ({ item, role, loading, onCancel, onConfirm }) => {
    const action = roleActions.find(([value]) => value === role);
    const isAdmin = role === 'admin';
    return (
        <div className="admin-modal-backdrop" role="presentation">
            <div className="admin-confirm-modal">
                <div className="admin-modal-icon"><FiShield /></div>
                <h3>确认调整权限？</h3>
                <p>将「{displayName(item)}」更新为「{roleText(role)}」。{action ? action[2] : ''}</p>
                {isAdmin && <p className="admin-confirm-warning">管理员权限很高，建议只给核心成员和技术负责人。</p>}
                <div className="admin-modal-actions">
                    <button className="admin-button" disabled={loading} onClick={onCancel}>取消</button>
                    <button className={`admin-button ${role === 'user' ? 'danger' : 'primary'}`} disabled={loading} onClick={onConfirm}>
                        {loading ? '处理中...' : '确认'}
                    </button>
                </div>
            </div>
        </div>
    );
};

const displayName = (item) => item.user?.nickname || item.user?.name || item.profile?.real_name || '深汕同学';

export default AdminPermissions;
