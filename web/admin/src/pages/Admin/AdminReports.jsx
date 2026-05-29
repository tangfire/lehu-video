import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FiExternalLink, FiFlag, FiTrash2 } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const reportStatusText = (status) => {
    const map = { 0: '待处理', 1: '已处理', 2: '已驳回' };
    return map[Number(status)] || '未知';
};

const AdminReports = () => {
    const [searchParams] = useSearchParams();
    const statusParam = searchParams.get('status') || '-1';
    const [reports, setReports] = useState([]);
    const [status, setStatus] = useState(statusParam);
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);
    const [actionLoading, setActionLoading] = useState(false);
    const [confirmAction, setConfirmAction] = useState(null);

    const load = async (nextPage = page, nextStatus = status) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listReports({ page: nextPage, size: 20, status: nextStatus });
            setReports(data.reports || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取举报失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        setStatus(statusParam);
        load(1, statusParam);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [statusParam]);

    const review = async (report, action) => {
        setActionLoading(true);
        setError('');
        try {
            await campusAdminApi.reviewReport(report.id, { action, reason: action === 'resolve' ? '已处理' : '已驳回' });
            setMessage(action === 'resolve' ? '举报已设为已处理' : '举报已驳回');
            window.setTimeout(() => setMessage(''), 2400);
            load(page);
        } catch (err) {
            setError(err.message || '处理举报失败');
        } finally {
            setActionLoading(false);
        }
    };

    const targetPostID = (report) => {
        if (!report) return '';
        if (report.target_type === 'post') return report.target_id;
        return report.comment?.post_id || report.comment?.post?.id || '';
    };

    const openTarget = (report) => {
        const postID = targetPostID(report);
        if (!postID) {
            setError('找不到关联帖子');
            return;
        }
        if (report.target_type === 'comment') {
            window.open(`/admin/moderation?tab=comments&post_id=${encodeURIComponent(String(postID))}`, '_blank', 'noopener,noreferrer');
            return;
        }
        window.open(`/admin/posts?keyword=${encodeURIComponent(String(postID))}`, '_blank', 'noopener,noreferrer');
    };

    const takedownTarget = async () => {
        if (!confirmAction?.report) return;
        const { report } = confirmAction;
        setActionLoading(true);
        setError('');
        try {
            if (report.target_type === 'post') {
                const target = report.target;
                if (target) {
                    await campusAdminApi.updatePost(report.target_id, {
                        category_code: target.category_code,
                        title: target.title,
                        content: target.content,
                        images: target.images || [],
                        media_type: target.media_type || 'text',
                        post_type: target.post_type || 'note',
                        extra: target.extra || {},
                        cover_url: target.cover_url || '',
                        status: 3,
                        audit_reason: '举报处理下架',
                        is_official: Boolean(target.is_official),
                        is_featured: Boolean(target.is_featured),
                        is_pinned: Boolean(target.is_pinned),
                        sort_weight: Number(target.sort_weight || 0),
                    });
                } else {
                    await campusAdminApi.deletePost(report.target_id);
                }
            } else {
                await campusAdminApi.deleteComment(report.target_id);
            }
            await campusAdminApi.reviewReport(report.id, { action: 'resolve', reason: '已下架举报对象' });
            setMessage('举报对象已下架，举报已设为已处理');
            window.setTimeout(() => setMessage(''), 2400);
            setConfirmAction(null);
            load(page);
        } catch (err) {
            setError(err.message || '下架举报对象失败');
        } finally {
            setActionLoading(false);
        }
    };

    return (
        <>
            {message && <div className="admin-toast success">{message}</div>}
            <div className="admin-toolbar">
                <select className="admin-select" value={status} onChange={(e) => setStatus(e.target.value)}>
                    <option value="-1">全部举报</option>
                    <option value="0">待处理</option>
                    <option value="1">已处理</option>
                    <option value="2">已驳回</option>
                </select>
                <button className="admin-button primary" onClick={() => load(1)}>查询</button>
            </div>
            {error && <div className="admin-error">{error}</div>}
            {loading && <div className="admin-loading">举报加载中...</div>}
            {!loading && reports.length === 0 && (
                <div className="admin-empty">
                    <FiFlag />
                    <span>暂无举报事项</span>
                </div>
            )}
            {!loading && reports.length > 0 && <div className="admin-table-wrap">
                <table className="admin-table">
                    <thead>
                        <tr>
                            <th>举报对象</th>
                            <th>原因</th>
                            <th>举报人</th>
                            <th>状态</th>
                            <th>时间</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        {reports.map((report) => (
                            <tr key={report.id}>
                                <td className="admin-title-cell">
                                    <strong>{report.target_type === 'post' ? report.target?.title : report.comment?.content}</strong>
                                    <div className="admin-muted">#{report.target_id}</div>
                                </td>
                                <td>
                                    <div>{report.reason}</div>
                                    <div className="admin-muted">{report.detail}</div>
                                </td>
                                <td>{report.reporter?.name || report.reporter?.nickname || '同学'}</td>
                                <td>{reportStatusText(report.status)}</td>
                                <td>{report.created_at}</td>
                                <td>
                                    <div className="admin-actions">
                                        <button className="admin-button" onClick={() => openTarget(report)}><FiExternalLink /> 查看</button>
                                        <button className="admin-button danger" disabled={actionLoading} onClick={() => setConfirmAction({ report })}><FiTrash2 /> 下架对象</button>
                                        <button className="admin-button" disabled={actionLoading || Number(report.status) === 1} onClick={() => review(report, 'resolve')}>设为已处理</button>
                                        <button className="admin-button" disabled={actionLoading || Number(report.status) === 2} onClick={() => review(report, 'reject')}>驳回举报</button>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>}
            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条</span>
                <button className="admin-button" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={page * 20 >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>
            {confirmAction && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal">
                        <div className="admin-modal-icon"><FiTrash2 /></div>
                        <h3>下架举报对象</h3>
                        <p>确认下架这条举报对应的{confirmAction.report?.target_type === 'post' ? '帖子' : '评论'}吗？下架后前台不再展示，举报会同时设为已处理。</p>
                        <div className="admin-modal-actions">
                            <button className="admin-button" onClick={() => setConfirmAction(null)}>取消</button>
                            <button className="admin-button danger" disabled={actionLoading} onClick={takedownTarget}>确认下架</button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default AdminReports;
