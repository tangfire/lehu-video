import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCalendar, FiCheckCircle, FiCheckSquare, FiCopy, FiDownload, FiGrid, FiRefreshCw, FiSquare } from 'react-icons/fi';
import { campusAdminApi, downloadBlob } from '../../api/admin';
import { compactNumber, excerpt, postCover } from './adminUtils';
import './Admin.css';

const today = () => {
    const date = new Date();
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
};

const revokePreviews = (items) => {
    items.forEach((item) => {
        if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
    });
};

const AdminMoments = () => {
    const [date, setDate] = useState(today());
    const [packageData, setPackageData] = useState(null);
    const [candidates, setCandidates] = useState([]);
    const [selectedIds, setSelectedIds] = useState([]);
    const [previews, setPreviews] = useState([]);
    const [loading, setLoading] = useState(false);
    const [candidateLoading, setCandidateLoading] = useState(false);
    const [downloadLoading, setDownloadLoading] = useState('');
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    useEffect(() => () => revokePreviews(previews), [previews]);

    useEffect(() => {
        let active = true;
        const load = async () => {
            setCandidateLoading(true);
            setError('');
            try {
                const data = await campusAdminApi.listMomentsCandidates({ date });
                if (!active) return;
                const posts = data.posts || [];
                setCandidates(posts);
                setSelectedIds(posts.slice(0, 9).map((post) => String(post.id)));
                setPackageData(null);
                setPreviews((old) => {
                    revokePreviews(old);
                    return [];
                });
            } catch (err) {
                if (!active) return;
                setCandidates([]);
                setSelectedIds([]);
                setError(err.message || '获取朋友圈候选失败');
            } finally {
                if (active) setCandidateLoading(false);
            }
        };
        load();
        return () => {
            active = false;
        };
    }, [date]);

    const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);

    const statsText = useMemo(() => {
        if (!packageData) return '等待生成';
        return `${packageData.count || 0}/9 张`;
    }, [packageData]);

    const loadPreviewImages = async (pkg) => {
        const posts = pkg.posts || [];
        const next = await Promise.all(posts.map(async (post) => {
            const blob = await campusAdminApi.downloadMomentsImage(pkg.package_id, post.slot);
            return {
                ...post,
                blob,
                previewUrl: URL.createObjectURL(blob),
            };
        }));
        setPreviews((old) => {
            revokePreviews(old);
            return next;
        });
    };

    const handleGenerate = async () => {
        setLoading(true);
        setError('');
        setMessage('');
        try {
            const payload = {
                date,
                ...(selectedIds.length > 0 ? { post_ids: selectedIds.map((id) => Number(id)).filter(Boolean) } : {}),
            };
            const data = await campusAdminApi.createMomentsPackage(payload);
            const pkg = data.package;
            setPackageData(pkg);
            await loadPreviewImages(pkg);
            const warning = (pkg.warnings || []).join('；');
            setMessage(warning || '朋友圈素材包已生成');
        } catch (err) {
            setError(err.message || '生成朋友圈素材失败');
        } finally {
            setLoading(false);
        }
    };

    const toggleCandidate = (postId) => {
        const id = String(postId);
        setError('');
        setSelectedIds((current) => {
            if (current.includes(id)) {
                return current.filter((item) => item !== id);
            }
            if (current.length >= 9) {
                setError('最多选择 9 个帖子');
                return current;
            }
            return [...current, id];
        });
    };

    const useTopNine = () => {
        setSelectedIds(candidates.slice(0, 9).map((post) => String(post.id)));
        setMessage('已按热度选择前 9 个帖子');
        setError('');
    };

    const refreshCandidates = async () => {
        setCandidateLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listMomentsCandidates({ date });
            const posts = data.posts || [];
            setCandidates(posts);
            setSelectedIds((current) => current.filter((id) => posts.some((post) => String(post.id) === id)).slice(0, 9));
            setMessage('候选列表已刷新');
        } catch (err) {
            setError(err.message || '刷新候选失败');
        } finally {
            setCandidateLoading(false);
        }
    };

    const handleDownloadZip = async () => {
        if (!packageData) return;
        setDownloadLoading('zip');
        setError('');
        try {
            const blob = await campusAdminApi.downloadMomentsZip(packageData.package_id);
            downloadBlob(blob, `ezai-moments-${date.replaceAll('-', '')}.zip`);
        } catch (err) {
            setError(err.message || '下载 ZIP 失败');
        } finally {
            setDownloadLoading('');
        }
    };

    const handleDownloadImage = (item) => {
        if (!item?.blob) return;
        downloadBlob(item.blob, `ezai-moments-${date.replaceAll('-', '')}-${String(item.slot).padStart(2, '0')}.png`);
    };

    const copyCaption = async () => {
        if (!packageData?.caption) return;
        try {
            await navigator.clipboard.writeText(packageData.caption);
            setMessage('朋友圈文案已复制');
        } catch {
            setError('复制失败，请手动选中文案复制');
        }
    };

    return (
        <div className="admin-moments-page">
            <section className="admin-panel admin-moments-toolbar">
                <div className="admin-moments-status">
                    <div className="admin-moments-status-icon">
                        <FiGrid />
                    </div>
                    <div>
                        <strong>朋友圈九图包</strong>
                        <span>{packageData ? statsText : `${selectedIds.length}/9 已选`}</span>
                    </div>
                </div>
                <div className="admin-moments-actions">
                    <label className="admin-date-control">
                        <FiCalendar />
                        <input type="date" value={date} onChange={(event) => setDate(event.target.value)} />
                    </label>
                    <button className="admin-button primary" type="button" onClick={handleGenerate} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                        {loading ? '生成中' : '生成已选素材'}
                    </button>
                    <button className="admin-button" type="button" onClick={handleDownloadZip} disabled={!packageData || downloadLoading === 'zip'}>
                        <FiDownload />
                        下载 ZIP
                    </button>
                </div>
            </section>

            <section className="admin-panel admin-moments-picker">
                <div className="admin-panel-head">
                    <div>
                        <h2>选择帖子</h2>
                        <p>{candidateLoading ? '正在加载候选' : `候选 ${candidates.length} 条，已选 ${selectedIds.length} 条`}</p>
                    </div>
                    <div className="admin-head-actions">
                        <button className="admin-button" type="button" onClick={useTopNine} disabled={candidateLoading || candidates.length === 0}>
                            <FiCheckSquare />
                            热度前 9
                        </button>
                        <button className="admin-button" type="button" onClick={refreshCandidates} disabled={candidateLoading}>
                            <FiRefreshCw className={candidateLoading ? 'spin' : ''} />
                            刷新候选
                        </button>
                    </div>
                </div>
                {candidates.length === 0 && (
                    <div className="admin-empty compact">当天还没有可用于朋友圈的图片帖。</div>
                )}
                {candidates.length > 0 && (
                    <div className="admin-moments-candidates">
                        {candidates.map((post) => {
                            const id = String(post.id);
                            const selected = selectedSet.has(id);
                            const order = selected ? selectedIds.indexOf(id) + 1 : 0;
                            return (
                                <button
                                    className={`admin-moments-candidate ${selected ? 'selected' : ''}`}
                                    type="button"
                                    key={id}
                                    onClick={() => toggleCandidate(id)}
                                >
                                    <div className="admin-moments-candidate-cover">
                                        {postCover(post) ? <img src={postCover(post)} alt="" /> : <FiGrid />}
                                        {selected && <span>{order}</span>}
                                    </div>
                                    <div className="admin-moments-candidate-body">
                                        <strong>{post.title || excerpt(post.content, 24) || '校园热帖'}</strong>
                                        <small>{excerpt(post.content, 42) || post.category_name || '图文笔记'}</small>
                                        <em>{compactNumber(post.like_count)} 赞 · {compactNumber(post.comment_count)} 评 · {compactNumber(post.collected_count)} 藏</em>
                                    </div>
                                    {selected ? <FiCheckSquare /> : <FiSquare />}
                                </button>
                            );
                        })}
                    </div>
                )}
            </section>

            {message && (
                <div className="admin-alert success">
                    <FiCheckCircle />
                    <span>{message}</span>
                </div>
            )}
            {error && (
                <div className="admin-alert danger">
                    <FiAlertCircle />
                    <span>{error}</span>
                </div>
            )}

            {packageData?.caption && (
                <section className="admin-panel admin-moments-caption">
                    <div>
                        <span>朋友圈文案</span>
                        <strong>{packageData.caption}</strong>
                    </div>
                    <button className="admin-icon-button" type="button" onClick={copyCaption} title="复制文案">
                        <FiCopy />
                    </button>
                </section>
            )}

            <section className="admin-moments-grid">
                {previews.length === 0 && (
                    <div className="admin-empty admin-moments-empty">选择日期后生成今日热帖素材。</div>
                )}
                {previews.map((item) => (
                    <article className="admin-moments-card" key={`${packageData?.package_id}-${item.slot}`}>
                        <div className="admin-moments-preview">
                            <img src={item.previewUrl} alt={`朋友圈素材 ${item.slot}`} />
                        </div>
                        <div className="admin-moments-card-body">
                            <div>
                                <span>#{item.slot}</span>
                                <strong>{item.title || '校园热帖'}</strong>
                                <small>
                                    {compactNumber(item.like_count)} 赞 · {compactNumber(item.comment_count)} 评 · {compactNumber(item.collected_count)} 藏
                                </small>
                            </div>
                            <button className="admin-icon-button" type="button" onClick={() => handleDownloadImage(item)} title="下载单图">
                                <FiDownload />
                            </button>
                        </div>
                    </article>
                ))}
            </section>
        </div>
    );
};

export default AdminMoments;
