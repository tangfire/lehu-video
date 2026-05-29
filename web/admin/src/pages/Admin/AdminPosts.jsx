import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { FiCheckCircle, FiChevronDown, FiImage, FiMoreHorizontal, FiSearch, FiShield, FiTrash2 } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber, excerpt, mediaTypeText, postCover, postTypeText, statusText } from './adminUtils';
import './Admin.css';

const pageSize = 20;

const initialFilters = {
    keyword: '',
    status: '-1',
    postType: '',
    categoryCode: '',
    opsFilter: '',
};

const postTypeOptions = [
    ['全部类型', ''],
    ['普通笔记', 'note'],
    ['攻略', 'guide'],
    ['问答', 'question'],
    ['失物', 'lost'],
    ['社团', 'club'],
];

const opsOptions = [
    ['全部运营状态', ''],
    ['官方内容', 'official'],
    ['精选内容', 'featured'],
    ['置顶内容', 'pinned'],
    ['有待处理举报', 'reported'],
];

const statusOptions = [
    ['全部状态', '-1'],
    ['正常展示', '1'],
    ['待审核', '0'],
    ['已拒绝', '2'],
    ['已下架', '3'],
];

const batchActions = [
    ['pin', '批量置顶'],
    ['unpin', '取消置顶'],
    ['feature', '设为精选'],
    ['unfeature', '取消精选'],
    ['official', '设为官方'],
    ['unofficial', '取消官方'],
    ['visible', '恢复可见'],
    ['delete', '下架'],
];

const aiRiskText = (risk) => {
    const map = { low: '低风险', medium: '需复核', high: '高风险' };
    return map[risk] || '';
};

const aiDecisionText = (decision) => {
    const map = { pass: '建议通过', review: '建议复核', reject: '疑似违规' };
    return map[decision] || '';
};

const buildPostPayload = (post, patch) => ({
    category_code: post.category_code,
    title: post.title,
    content: post.content,
    images: post.images || [],
    media_type: post.media_type || 'text',
    post_type: post.post_type || 'note',
    extra: post.extra || {},
    cover_url: post.cover_url || '',
    status: Number(post.status ?? 1),
    audit_reason: post.audit_reason || '',
    is_official: Boolean(post.is_official),
    is_featured: Boolean(post.is_featured),
    is_pinned: Boolean(post.is_pinned),
    sort_weight: Number(post.sort_weight || 0),
    ...patch,
});

const AdminPosts = () => {
    const [searchParams] = useSearchParams();
    const initialStatus = searchParams.get('status') || initialFilters.status;
    const [posts, setPosts] = useState([]);
    const [categories, setCategories] = useState([]);
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [filters, setFilters] = useState({ ...initialFilters, status: initialStatus });
    const [selectedIds, setSelectedIds] = useState([]);
    const [weightValue, setWeightValue] = useState('100');
    const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);
    const [showBatchTools, setShowBatchTools] = useState(false);
    const [confirmAction, setConfirmAction] = useState(null);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [actionLoading, setActionLoading] = useState(false);

    const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);
    const allCurrentSelected = posts.length > 0 && posts.every((post) => selectedSet.has(String(post.id)));

    const load = async (nextPage = page, nextFilters = filters) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listPosts({
                page: nextPage,
                size: pageSize,
                keyword: nextFilters.keyword,
                status: nextFilters.status,
                post_type: nextFilters.postType,
                category_code: nextFilters.categoryCode,
                ops_filter: nextFilters.opsFilter,
            });
            setPosts(data.posts || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
            setSelectedIds([]);
        } catch (err) {
            setError(err.message || '获取内容失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        campusAdminApi.listCategories()
            .then((data) => setCategories(data.categories || []))
            .catch(() => setCategories([]));
        const nextFilters = { ...initialFilters, status: initialStatus };
        setFilters(nextFilters);
        load(1, nextFilters);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [initialStatus]);

    const updateFilter = (key, value, autoLoad = false) => {
        const next = { ...filters, [key]: value };
        setFilters(next);
        if (autoLoad) load(1, next);
    };

    const showMessage = (text) => {
        setMessage(text);
        window.setTimeout(() => setMessage(''), 2600);
    };

    const updatePost = async (post, patch, successText = '内容已更新') => {
        setActionLoading(true);
        setError('');
        try {
            await campusAdminApi.updatePost(post.id, buildPostPayload(post, patch));
            showMessage(successText);
            await load(page);
        } catch (err) {
            setError(err.message || '操作失败');
        } finally {
            setActionLoading(false);
        }
    };

    const runBatch = async (action, extra = {}) => {
        if (selectedIds.length === 0) {
            setError('请先选择内容');
            return;
        }
        setActionLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.batchPosts({
                ids: selectedIds,
                action,
                ...extra,
            });
            showMessage(`已更新 ${data.updated_count || selectedIds.length} 条内容`);
            setConfirmAction(null);
            await load(page);
        } catch (err) {
            setError(err.message || '批量操作失败');
        } finally {
            setActionLoading(false);
        }
    };

    const toggleSelected = (id) => {
        const key = String(id);
        setSelectedIds((prev) => prev.includes(key) ? prev.filter((item) => item !== key) : [...prev, key]);
    };

    const toggleAllCurrent = () => {
        if (allCurrentSelected) {
            setSelectedIds([]);
            return;
        }
        setSelectedIds(posts.map((post) => String(post.id)));
    };

    const saveWeight = (post, value) => {
        const weight = Number(value || 0);
        updatePost(post, { sort_weight: weight }, '权重已保存');
    };

    return (
        <div className="admin-workbench">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}

            <section className="admin-panel admin-filter-panel">
                <div className="admin-panel-head">
                    <div>
                        <h2>内容工作台</h2>
                        <p>默认只处理最常用的搜索、审核、置顶和精选；高级筛选需要时再展开。</p>
                    </div>
                    <button className="admin-button primary" onClick={() => load(1)} disabled={loading}>
                        <FiSearch />
                        查询
                    </button>
                    <Link className="admin-button" to="/admin/audit">
                        <FiShield />
                        审核设置
                    </Link>
                </div>
                <div className="admin-simple-filter-row">
                    <input className="admin-input" value={filters.keyword} onChange={(e) => updateFilter('keyword', e.target.value)} placeholder="搜索标题 / 正文" />
                    <select className="admin-select" value={filters.status} onChange={(e) => updateFilter('status', e.target.value, true)}>
                        {statusOptions.map(([label, value]) => <option key={value} value={value}>{label}</option>)}
                    </select>
                    <button className={filters.opsFilter === 'pinned' ? 'admin-button active' : 'admin-button'} type="button" onClick={() => updateFilter('opsFilter', filters.opsFilter === 'pinned' ? '' : 'pinned', true)}>只看置顶</button>
                    <button className={filters.opsFilter === 'featured' ? 'admin-button active' : 'admin-button'} type="button" onClick={() => updateFilter('opsFilter', filters.opsFilter === 'featured' ? '' : 'featured', true)}>只看精选</button>
                    <button className={filters.opsFilter === 'reported' ? 'admin-button active' : 'admin-button'} type="button" onClick={() => updateFilter('opsFilter', filters.opsFilter === 'reported' ? '' : 'reported', true)}>有举报</button>
                    <button className="admin-button subtle" type="button" onClick={() => setShowAdvancedFilters((value) => !value)}>
                        高级筛选
                        <FiChevronDown className={showAdvancedFilters ? 'rotate' : ''} />
                    </button>
                </div>
                {showAdvancedFilters && (
                    <div className="admin-filter-grid advanced">
                        <select className="admin-select" value={filters.postType} onChange={(e) => updateFilter('postType', e.target.value, true)}>
                            {postTypeOptions.map(([label, value]) => <option key={value} value={value}>{label}</option>)}
                        </select>
                        <select className="admin-select" value={filters.categoryCode} onChange={(e) => updateFilter('categoryCode', e.target.value, true)}>
                            <option value="">全部版块</option>
                            {categories.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
                        </select>
                        <select className="admin-select" value={filters.opsFilter} onChange={(e) => updateFilter('opsFilter', e.target.value, true)}>
                            {opsOptions.map(([label, value]) => <option key={value} value={value}>{label}</option>)}
                        </select>
                    </div>
                )}
            </section>

            <section className="admin-batch-bar">
                <label className="admin-check-line">
                    <input type="checkbox" checked={allCurrentSelected} onChange={toggleAllCurrent} />
                    已选 {selectedIds.length} 条
                </label>
                <div className="admin-batch-actions compact">
                    <button className="admin-button" type="button" disabled={actionLoading || selectedIds.length === 0} onClick={() => setConfirmAction({ type: 'batch', action: 'pin', label: '批量置顶' })}>置顶</button>
                    <button className="admin-button" type="button" disabled={actionLoading || selectedIds.length === 0} onClick={() => setConfirmAction({ type: 'batch', action: 'feature', label: '设为精选' })}>精选</button>
                    <button className="admin-button" type="button" disabled={actionLoading || selectedIds.length === 0} onClick={() => setConfirmAction({ type: 'batch', action: 'visible', label: '恢复可见' })}>恢复可见</button>
                    <button className="admin-button subtle" type="button" onClick={() => setShowBatchTools((value) => !value)}>
                        更多
                        <FiChevronDown className={showBatchTools ? 'rotate' : ''} />
                    </button>
                </div>
            </section>
            {showBatchTools && (
                <section className="admin-advanced-strip">
                    {batchActions.filter(([action]) => !['pin', 'feature', 'visible'].includes(action)).map(([action, label]) => {
                        const needsConfirm = action === 'delete';
                        return (
                            <button
                                className={action === 'delete' ? 'admin-button danger' : 'admin-button'}
                                type="button"
                                key={action}
                                disabled={actionLoading || selectedIds.length === 0}
                                onClick={() => needsConfirm ? setConfirmAction({ type: 'batch', action, label }) : setConfirmAction({ type: 'batch', action, label })}
                            >
                                {label}
                            </button>
                        );
                    })}
                    <input className="admin-input mini" type="number" value={weightValue} onChange={(e) => setWeightValue(e.target.value)} />
                    <button className="admin-button" disabled={actionLoading || selectedIds.length === 0} onClick={() => setConfirmAction({ type: 'batch', action: 'set_weight', label: `设权重为 ${Number(weightValue || 0)}`, extra: { sort_weight: Number(weightValue || 0) } })}>设权重</button>
                </section>
            )}

            <section className="admin-content-list">
                {loading && <div className="admin-loading">内容加载中...</div>}
                {!loading && posts.length === 0 && <div className="admin-empty">没有匹配的内容，换个筛选条件试试。</div>}
                {!loading && posts.map((post) => (
                    <article className="admin-post-card" key={post.id}>
                        <label className="admin-post-select">
                            <input type="checkbox" checked={selectedSet.has(String(post.id))} onChange={() => toggleSelected(post.id)} />
                        </label>
                        <div className="admin-post-cover">
                            {postCover(post) ? (
                                <img src={postCover(post)} alt="" />
                            ) : (
                                <div className="admin-cover-placeholder">
                                    <FiImage />
                                    <span>{mediaTypeText(post.media_type)}</span>
                                </div>
                            )}
                        </div>
                        <div className="admin-post-body">
                            <div className="admin-post-title-row">
                                <div>
                                    <h3>{post.title || '未命名内容'}</h3>
                                    <p>{excerpt(post.content, 96) || '暂无正文'}</p>
                                </div>
                                <span className={`admin-status-pill status-${post.status}`}>{statusText(post.status)}</span>
                            </div>
                            <div className="admin-tag-row">
                                {post.is_pinned && <span className="admin-tag hot">置顶</span>}
                                {post.is_featured && <span className="admin-tag warn">精选</span>}
                                {post.is_official && <span className="admin-tag">官方</span>}
                                <span className="admin-tag neutral">{postTypeText(post.post_type)}</span>
                                <span className="admin-tag neutral">{post.category_name || post.category_code || '未分组'}</span>
                            </div>
                            <div className="admin-post-meta">
                                <span>{compactNumber(post.like_count)} 赞</span>
                                <span>{compactNumber(post.comment_count)} 评</span>
                                <span>{compactNumber(post.collected_count)} 藏</span>
                                <span>权重 {post.sort_weight || 0}</span>
                                <span>{post.created_at}</span>
                            </div>
                            {(post.audit_reason || post.ai_audit_reason || post.ai_audit_error) && (
                                <div className={`admin-audit-note ${post.ai_audit_risk ? `risk-${post.ai_audit_risk}` : ''}`}>
                                    <FiShield />
                                    <span>
                                        {post.ai_audit_status && `AI ${aiDecisionText(post.ai_audit_decision) || post.ai_audit_status}${aiRiskText(post.ai_audit_risk) ? ` · ${aiRiskText(post.ai_audit_risk)}` : ''}：`}
                                        {post.ai_audit_reason || post.ai_audit_error || post.audit_reason}
                                    </span>
                                </div>
                            )}
                        </div>
                        <div className="admin-post-ops">
                            <div className="admin-actions">
                                <button className={post.is_pinned ? 'admin-button active' : 'admin-button'} disabled={actionLoading} onClick={() => updatePost(post, { is_pinned: !post.is_pinned })}>{post.is_pinned ? '置顶中' : '置顶'}</button>
                                <button className={post.is_featured ? 'admin-button active' : 'admin-button'} disabled={actionLoading} onClick={() => updatePost(post, { is_featured: !post.is_featured })}>{post.is_featured ? '精选中' : '精选'}</button>
                                {Number(post.status) === 1 ? (
                                    <button className="admin-button quiet" type="button" disabled>
                                        <FiCheckCircle />
                                        正常展示
                                    </button>
                                ) : (
                                    <button className="admin-button" disabled={actionLoading} onClick={() => updatePost(post, { status: 1, audit_reason: '' }, '内容已恢复可见')}>
                                        <FiCheckCircle />
                                        恢复可见
                                    </button>
                                )}
                                <button className={post.is_official ? 'admin-button active quiet' : 'admin-button quiet'} disabled={actionLoading} onClick={() => updatePost(post, { is_official: !post.is_official })}>{post.is_official ? '官方' : '设官方'}</button>
                            </div>
                            <div className="admin-weight-editor">
                                <input
                                    className="admin-input mini"
                                    type="number"
                                    defaultValue={post.sort_weight || 0}
                                    onBlur={(e) => {
                                        if (Number(e.target.value || 0) !== Number(post.sort_weight || 0)) saveWeight(post, e.target.value);
                                    }}
                                />
                                <button className="admin-button danger" disabled={actionLoading || Number(post.status) === 3} onClick={() => setConfirmAction({ type: 'single', post, label: '下架内容' })}>
                                    <FiTrash2 />
                                    {Number(post.status) === 3 ? '已下架' : '下架'}
                                </button>
                            </div>
                        </div>
                    </article>
                ))}
            </section>

            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条，第 {page} 页</span>
                <button className="admin-button" disabled={loading || page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={loading || page * pageSize >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>

            {confirmAction && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal">
                        <div className="admin-modal-icon"><FiMoreHorizontal /></div>
                        <h3>{confirmAction.label}</h3>
                        <p>
                            {confirmAction.type === 'batch'
                                ? `将对 ${selectedIds.length} 条内容执行「${confirmAction.label}」。涉及置顶、精选、官方、权重或恢复可见时，会影响小程序首页曝光。`
                                : `确认下架「${confirmAction.post?.title || '这条内容'}」吗？下架后前台不再展示，可通过「恢复可见」重新上架。`}
                        </p>
                        <div className="admin-modal-actions">
                            <button className="admin-button" onClick={() => setConfirmAction(null)}>取消</button>
                            <button
                                className="admin-button danger"
                                disabled={actionLoading}
                                onClick={() => {
                                    if (confirmAction.type === 'batch') {
                                        runBatch(confirmAction.action, confirmAction.extra || {});
                                    } else {
                                        updatePost(confirmAction.post, { status: 3, audit_reason: '运营下架' }, '内容已下架');
                                        setConfirmAction(null);
                                    }
                                }}
                            >
                                确认
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AdminPosts;
