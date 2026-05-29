import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCalendar, FiCheckCircle, FiCopy, FiDownload, FiGrid, FiRefreshCw } from 'react-icons/fi';
import { campusAdminApi, downloadBlob } from '../../api/admin';
import { compactNumber } from './adminUtils';
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
    const [previews, setPreviews] = useState([]);
    const [loading, setLoading] = useState(false);
    const [downloadLoading, setDownloadLoading] = useState('');
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    useEffect(() => () => revokePreviews(previews), [previews]);

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
            const data = await campusAdminApi.createMomentsPackage({ date });
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
                        <span>{statsText}</span>
                    </div>
                </div>
                <div className="admin-moments-actions">
                    <label className="admin-date-control">
                        <FiCalendar />
                        <input type="date" value={date} onChange={(event) => setDate(event.target.value)} />
                    </label>
                    <button className="admin-button primary" type="button" onClick={handleGenerate} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                        {loading ? '生成中' : '生成素材'}
                    </button>
                    <button className="admin-button" type="button" onClick={handleDownloadZip} disabled={!packageData || downloadLoading === 'zip'}>
                        <FiDownload />
                        下载 ZIP
                    </button>
                </div>
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
