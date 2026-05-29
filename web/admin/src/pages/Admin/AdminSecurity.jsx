import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiShield } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber } from './adminUtils';
import './Admin.css';

const AdminSecurity = () => {
    const [security, setSecurity] = useState(null);
    const [blockIP, setBlockIP] = useState('');
    const [reason, setReason] = useState('');
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);
    const [confirmAction, setConfirmAction] = useState(null);

    const load = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.security();
            setSecurity(data.security || {});
        } catch (err) {
            setError(err.message || '获取安全数据失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load();
    }, []);

    const stats = useMemo(() => {
        if (!security) return [];
        return [
            { label: '今日请求', value: security.today_requests || 0, hint: '后端真实请求' },
            { label: '独立 IP', value: security.today_unique_ips || 0, hint: '今日访问来源' },
            { label: '限流次数', value: security.today_rate_limited || 0, hint: '429 请求' },
            { label: '封禁 IP', value: security.active_blocked_ips || 0, hint: `拦截 ${compactNumber(security.today_blocked || 0)} 次` },
            { label: '错误请求', value: security.today_errors || 0, hint: '4xx / 5xx' },
        ];
    }, [security]);

    const openBlockConfirm = () => {
        const ip = blockIP.trim();
        if (!ip) {
            setError('请输入要封禁的 IP');
            return;
        }
        setError('');
        setConfirmAction({ type: 'block', ip, reason: reason.trim() || '后台手动封禁' });
    };

    const submitBlock = async () => {
        const ip = confirmAction?.ip || blockIP.trim();
        if (!ip) {
            setError('请输入要封禁的 IP');
            return;
        }
        try {
            await campusAdminApi.blockIP({ ip, reason: confirmAction?.reason || reason.trim() || '后台手动封禁' });
            setMessage('IP 已封禁');
            setBlockIP('');
            setReason('');
            setConfirmAction(null);
            window.setTimeout(() => setMessage(''), 2400);
            load();
        } catch (err) {
            setError(err.message || '封禁失败');
        }
    };

    const unblock = async (ip) => {
        try {
            await campusAdminApi.unblockIP(ip);
            setMessage('IP 已解封');
            setConfirmAction(null);
            window.setTimeout(() => setMessage(''), 2400);
            load();
        } catch (err) {
            setError(err.message || '解封失败');
        }
    };

    if (loading && !security) return <div className="admin-loading">安全数据加载中...</div>;

    return (
        <div className="admin-security-page">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}

            <section className="admin-key-grid security">
                {stats.map((item) => (
                    <div className="admin-key-stat" key={item.label}>
                        <span>{item.label}</span>
                        <strong>{compactNumber(item.value)}</strong>
                        <em>{item.hint}</em>
                    </div>
                ))}
            </section>

            <section className="admin-panel">
                <div className="admin-panel-head">
                    <h2>手动封禁 IP</h2>
                    <button className="admin-button" onClick={load}>刷新</button>
                </div>
                <div className="admin-toolbar security">
                    <input className="admin-input" value={blockIP} onChange={(e) => setBlockIP(e.target.value)} placeholder="例如 127.0.0.1" />
                    <input className="admin-input" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="封禁原因，可选" />
                    <button className="admin-button danger" onClick={openBlockConfirm}>封禁</button>
                </div>
            </section>

            <section className="admin-two-column security">
                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <h2>Top IP</h2>
                    </div>
                    <div className="admin-security-list">
                        {(security?.top_ips || []).map((item) => (
                            <div className="admin-security-row" key={item.ip}>
                                <div>
                                    <strong>{item.ip || 'unknown'}</strong>
                                    <span>{item.last_seen || ''}</span>
                                </div>
                                <div className="admin-security-count">
                                    <b>{compactNumber(item.request_count)}</b>
                                    <span>错误 {compactNumber(item.error_count)}</span>
                                </div>
                            </div>
                        ))}
                        {!(security?.top_ips || []).length && <div className="admin-empty compact">暂无访问记录</div>}
                    </div>
                </div>

                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <h2>Top 接口</h2>
                    </div>
                    <div className="admin-security-list">
                        {(security?.top_paths || []).map((item) => (
                            <div className="admin-security-row" key={item.path}>
                                <div>
                                    <strong>{item.path}</strong>
                                    <span>错误 {compactNumber(item.error_count)}</span>
                                </div>
                                <div className="admin-security-count">
                                    <b>{compactNumber(item.request_count)}</b>
                                    <span>请求</span>
                                </div>
                            </div>
                        ))}
                        {!(security?.top_paths || []).length && <div className="admin-empty compact">暂无接口记录</div>}
                    </div>
                </div>
            </section>

            <section className="admin-panel">
                <div className="admin-panel-head">
                    <h2>已封禁 IP</h2>
                </div>
                <div className="admin-security-list">
                    {(security?.blocked_ips || []).map((item) => (
                        <div className="admin-security-row" key={item.ip}>
                            <div>
                                <strong>{item.ip}</strong>
                                <span>{item.reason || '未填写原因'} · {item.updated_at}</span>
                            </div>
                            <button className="admin-button" onClick={() => setConfirmAction({ type: 'unblock', ip: item.ip })}>解封</button>
                        </div>
                    ))}
                    {!(security?.blocked_ips || []).length && <div className="admin-empty compact">暂无封禁 IP</div>}
                </div>
            </section>

            <section className="admin-panel">
                <div className="admin-panel-head">
                    <h2>最近请求</h2>
                </div>
                <div className="admin-table-wrap">
                    <table className="admin-table">
                        <thead>
                            <tr>
                                <th>时间</th>
                                <th>IP</th>
                                <th>接口</th>
                                <th>状态</th>
                                <th>耗时</th>
                            </tr>
                        </thead>
                        <tbody>
                            {(security?.recent_logs || []).map((item) => (
                                <tr key={item.id}>
                                    <td>{item.created_at}</td>
                                    <td>{item.ip}</td>
                                    <td className="admin-title-cell">
                                        <strong>{item.method} {item.path}</strong>
                                        <div className="admin-muted">user #{item.user_id || 0}</div>
                                    </td>
                                    <td>
                                        <span className={`admin-status ${Number(item.status_code) >= 400 ? 'status-0' : 'status-2'}`}>
                                            {item.status_code}
                                            {item.rate_limited ? ' 限流' : ''}
                                            {item.blocked ? ' 封禁' : ''}
                                        </span>
                                    </td>
                                    <td>{item.duration_ms}ms</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                    {!(security?.recent_logs || []).length && <div className="admin-empty compact"><FiShield /> 暂无请求记录</div>}
                </div>
            </section>

            {confirmAction && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal">
                        <div className="admin-modal-icon danger"><FiAlertTriangle /></div>
                        <h3>{confirmAction.type === 'block' ? '确认封禁 IP？' : '确认解封 IP？'}</h3>
                        <p>
                            {confirmAction.type === 'block'
                                ? `封禁后，来自 ${confirmAction.ip} 的请求会被拦截，可能影响同一出口网络下的正常用户。`
                                : `解封后，${confirmAction.ip} 可以重新访问小程序和后台接口。`}
                        </p>
                        {confirmAction.type === 'block' && <p className="admin-confirm-warning">建议只在明显恶意请求、刷接口或攻击行为时封禁，并保留原因。</p>}
                        <div className="admin-modal-actions">
                            <button className="admin-button" onClick={() => setConfirmAction(null)}>取消</button>
                            <button className={confirmAction.type === 'block' ? 'admin-button danger' : 'admin-button primary'} onClick={() => (confirmAction.type === 'block' ? submitBlock() : unblock(confirmAction.ip))}>
                                {confirmAction.type === 'block' ? '确认封禁' : '确认解封'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AdminSecurity;
