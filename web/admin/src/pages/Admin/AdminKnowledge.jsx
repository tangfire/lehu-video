import { useCallback, useEffect, useMemo, useState } from 'react';
import { FiBookOpen, FiCheckCircle, FiFileText, FiRefreshCw, FiSearch, FiUploadCloud, FiZap } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber, excerpt } from './adminUtils';
import './Admin.css';

const pageSize = 12;

const categories = [
    ['general', '通用'],
    ['registration', '报到'],
    ['dorm', '宿舍'],
    ['traffic', '交通'],
    ['timetable', '课表'],
    ['network', '校园网'],
    ['express', '快递'],
    ['military', '军训'],
    ['club', '社团'],
    ['platform', '平台使用'],
];

const statusText = {
    draft: '草稿',
    indexing: '索引中',
    active: '已启用',
    disabled: '已下架',
    failed: '失败',
};

const initialManual = {
    title: '',
    source: '运营录入',
    category: 'general',
    content_type: 'text',
    raw_content: '',
    status: 'active',
    effective_at: '',
    expired_at: '',
};

const formatDateTimeLocal = (value) => {
    if (!value) return '';
    return value.replace(' ', 'T').slice(0, 16);
};

const toISOStringOrEmpty = (value) => {
    if (!value) return '';
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? '' : date.toISOString();
};

const AdminKnowledge = ({ mode = 'full' }) => {
    const [documents, setDocuments] = useState([]);
    const [chunks, setChunks] = useState([]);
    const [logs, setLogs] = useState([]);
    const [selectedDoc, setSelectedDoc] = useState(null);
    const [query, setQuery] = useState('');
    const [status, setStatus] = useState('');
    const [category, setCategory] = useState('');
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [manual, setManual] = useState(initialManual);
    const [uploadMeta, setUploadMeta] = useState({ title: '', source: '学校官方资料', category: 'general', effective_at: '', expired_at: '' });
    const [testQuery, setTestQuery] = useState('');
    const [testResult, setTestResult] = useState(null);
    const [ragHealth, setRagHealth] = useState(null);
    const [loading, setLoading] = useState(false);
    const [working, setWorking] = useState('');
    const [error, setError] = useState('');
    const [toast, setToast] = useState('');

    const loadDocuments = useCallback(async (nextPage = page) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listKnowledgeDocuments({
                keyword: query,
                status,
                category,
                page: nextPage,
                size: pageSize,
            });
            setDocuments(data.documents || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取知识库文档失败');
        } finally {
            setLoading(false);
        }
    }, [category, page, query, status]);

    const loadLogs = useCallback(async () => {
        try {
            const data = await campusAdminApi.listKnowledgeQueryLogs({ page: 1, size: 5 });
            setLogs(data.logs || []);
        } catch {
            setLogs([]);
        }
    }, []);

    const loadRagHealth = useCallback(async () => {
        try {
            const data = await campusAdminApi.aiReplySummary();
            setRagHealth(data.rag_health || null);
        } catch {
            setRagHealth({ status: 'unavailable', qdrant: 'unknown', last_error: '无法获取 RAG 状态' });
        }
    }, []);

    useEffect(() => {
        loadDocuments(1);
        loadLogs();
        loadRagHealth();
    }, []); // eslint-disable-line react-hooks/exhaustive-deps

    const activeCount = useMemo(() => documents.filter((item) => item.status === 'active').length, [documents]);
    const failedCount = useMemo(() => documents.filter((item) => item.status === 'failed').length, [documents]);

    const selectDoc = async (doc) => {
        setSelectedDoc(doc);
        setChunks([]);
        try {
            const data = await campusAdminApi.listKnowledgeChunks(doc.id, { page: 1, size: 20 });
            setChunks(data.chunks || []);
        } catch (err) {
            setError(err.message || '获取片段失败');
        }
    };

    const createManual = async () => {
        if (working) return;
        if (!manual.title.trim() || !manual.raw_content.trim()) {
            setError('请填写标题和正文');
            return;
        }
        setWorking('manual');
        setError('');
        setToast('');
        try {
            const data = await campusAdminApi.createKnowledgeDocument({
                ...manual,
                effective_at: toISOStringOrEmpty(manual.effective_at),
                expired_at: toISOStringOrEmpty(manual.expired_at),
            });
            setToast('已录入并开始索引');
            setManual(initialManual);
            await loadDocuments(1);
            if (data.document) await selectDoc(data.document);
        } catch (err) {
            setError(err.message || '录入知识失败');
        } finally {
            setWorking('');
        }
    };

    const uploadFile = async (event) => {
        const file = event.target.files?.[0];
        event.target.value = '';
        if (!file || working) return;
        setWorking('upload');
        setError('');
        setToast('');
        try {
            const uploaded = await campusAdminApi.uploadKnowledgeFile(file);
            const data = await campusAdminApi.createKnowledgeDocument({
                title: uploadMeta.title.trim() || file.name.replace(/\.[^.]+$/, ''),
                source: uploadMeta.source || '学校官方资料',
                category: uploadMeta.category || 'general',
                content_type: 'file',
                file_url: uploaded.url,
                file_id: uploaded.file_id,
                file_type: uploaded.file_type,
                status: 'active',
                effective_at: toISOStringOrEmpty(uploadMeta.effective_at),
                expired_at: toISOStringOrEmpty(uploadMeta.expired_at),
            });
            setToast('文档已上传并开始索引');
            setUploadMeta({ title: '', source: '学校官方资料', category: uploadMeta.category || 'general', effective_at: '', expired_at: '' });
            await loadDocuments(1);
            if (data.document) await selectDoc(data.document);
        } catch (err) {
            setError(err.message || '上传文档失败');
        } finally {
            setWorking('');
        }
    };

    const updateDocStatus = async (doc, nextStatus) => {
        if (working) return;
        setWorking(doc.id);
        setError('');
        setToast('');
        try {
            const data = await campusAdminApi.updateKnowledgeDocument(doc.id, {
                title: doc.title,
                source: doc.source,
                category: doc.category,
                status: nextStatus,
                effective_at: doc.effective_at,
                expired_at: doc.expired_at,
            });
            setToast(nextStatus === 'disabled' ? '已下架知识文档' : '已启用知识文档');
            await loadDocuments(page);
            if (data.document) await selectDoc(data.document);
        } catch (err) {
            setError(err.message || '更新文档状态失败');
        } finally {
            setWorking('');
        }
    };

    const refreshSelectedDoc = async () => {
        if (!selectedDoc || working) return;
        await selectDoc(selectedDoc);
        await loadDocuments(page);
    };

    const reindex = async (doc) => {
        if (working) return;
        setWorking(doc.id);
        setError('');
        setToast('');
        try {
            const data = await campusAdminApi.reindexKnowledgeDocument(doc.id);
            setToast('已开始重新索引');
            await loadDocuments(page);
            if (data.document) await selectDoc(data.document);
        } catch (err) {
            setError(err.message || '重新索引失败');
        } finally {
            setWorking('');
        }
    };

    const runTestQuery = async () => {
        if (!testQuery.trim()) {
            setError('请输入测试问题');
            return;
        }
        setWorking('test');
        setError('');
        try {
            const data = await campusAdminApi.testKnowledgeQuery({ query: testQuery, top_k: 5 });
            setTestResult(data.result || {});
            await loadLogs();
        } catch (err) {
            setError(err.message || '测试提问失败');
        } finally {
            setWorking('');
        }
    };

    return (
        <div className="admin-knowledge-page">
            {toast && <div className="admin-toast success">{toast}</div>}
            {error && <div className="admin-error">{error}</div>}

            {mode === 'full' && (
                <section className="admin-simple-head knowledge">
                    <div>
                        <span className="admin-kicker">RAG Knowledge Base</span>
                        <h2>让 e仔有资料可查</h2>
                        <p>上传学校官方资料或录入已确认信息，e仔被 @ 时会先判断是否需要查库，再基于命中的资料自然回复。</p>
                    </div>
                    <button className="admin-button" type="button" onClick={() => { loadDocuments(page); loadLogs(); loadRagHealth(); }} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                        刷新
                    </button>
                </section>
            )}

            {mode !== 'test' && <section className="admin-key-grid ai">
                <div className={`admin-key-stat knowledge-health ${ragHealth?.status === 'ok' ? 'ok' : 'warn'}`}>
                    <span>RAG 服务</span>
                    <strong>{ragHealth?.status || '未知'}</strong>
                    <em>Qdrant {ragHealth?.qdrant || '-'}</em>
                </div>
                <div className="admin-key-stat knowledge-health">
                    <span>活跃片段</span>
                    <strong>{compactNumber(ragHealth?.chunk_count || 0)}</strong>
                    <em>真实参与 e仔检索</em>
                </div>
                <div className="admin-key-stat">
                    <span>当前文档</span>
                    <strong>{compactNumber(total)}</strong>
                    <em>包含草稿、启用、失败</em>
                </div>
                <div className="admin-key-stat">
                    <span>本页启用</span>
                    <strong>{compactNumber(activeCount)}</strong>
                    <em>只有启用片段参与回复</em>
                </div>
                <div className="admin-key-stat">
                    <span>本页失败</span>
                    <strong>{compactNumber(failedCount)}</strong>
                    <em>需要重新索引或检查文件</em>
                </div>
                <div className="admin-key-stat">
                    <span>最近查询</span>
                    <strong>{compactNumber(logs.length)}</strong>
                    <em>用于追溯 e仔回答依据</em>
                </div>
            </section>}
            {ragHealth?.last_error && <div className="admin-rag-health-error">RAG 最近错误：{ragHealth.last_error}</div>}

            {mode !== 'test' && <div className="admin-knowledge-grid">
                <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>知识文档</h2>
                            <p>建议只放已确认资料，不把用户帖子自动入库。</p>
                        </div>
                    </div>
                    <div className="admin-toolbar knowledge">
                        <input className="admin-input" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="搜索标题、来源或正文" />
                        <select className="admin-select" value={category} onChange={(e) => setCategory(e.target.value)}>
                            <option value="">全部分类</option>
                            {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                        </select>
                        <select className="admin-select" value={status} onChange={(e) => setStatus(e.target.value)}>
                            <option value="">全部状态</option>
                            {Object.entries(statusText).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                        </select>
                        <button className="admin-button primary" type="button" onClick={() => loadDocuments(1)}>
                            <FiSearch />
                            筛选
                        </button>
                    </div>

                    <div className="admin-knowledge-list">
                        {loading && !documents.length && <div className="admin-loading">知识库加载中...</div>}
                        {!loading && !documents.length && <div className="admin-empty compact">暂无知识文档</div>}
                        {documents.map((doc) => (
                            <article className={`admin-knowledge-doc ${selectedDoc?.id === doc.id ? 'active' : ''}`} key={doc.id} onClick={() => selectDoc(doc)}>
                                <div className="admin-knowledge-doc-main">
                                    <div className="admin-ai-task-head">
                                        <span className={`admin-status knowledge-status-${doc.status}`}>{statusText[doc.status] || doc.status}</span>
                                        <span>{categoryLabel(doc.category)}</span>
                                        <span>{doc.content_type === 'file' ? doc.file_type?.toUpperCase() : '手动录入'}</span>
                                    </div>
                                    <strong>{doc.title}</strong>
                                    <p>{doc.source || '未填写来源'} · {doc.chunk_count || 0} 个片段 · {doc.updated_at}</p>
                                    {(doc.effective_at || doc.expired_at) && (
                                        <p className="admin-knowledge-validity">
                                            生效 {doc.effective_at || '立即'} · 失效 {doc.expired_at || '长期'}
                                        </p>
                                    )}
                                    {doc.error_message && <div className="admin-ai-error">{doc.error_message}</div>}
                                </div>
                                <div className="admin-knowledge-doc-actions" onClick={(e) => e.stopPropagation()}>
                                    <button className="admin-button subtle" type="button" disabled={working === doc.id} onClick={() => reindex(doc)}>
                                        <FiRefreshCw className={working === doc.id ? 'spin' : ''} />
                                        重建
                                    </button>
                                    {doc.status === 'active' ? (
                                        <button className="admin-button danger" type="button" disabled={working === doc.id} onClick={() => updateDocStatus(doc, 'disabled')}>下架</button>
                                    ) : (
                                        <button className="admin-button" type="button" disabled={working === doc.id} onClick={() => updateDocStatus(doc, 'active')}>启用</button>
                                    )}
                                </div>
                            </article>
                        ))}
                    </div>
                    <div className="admin-pagination">
                        <span>共 {compactNumber(total)} 条</span>
                        <button className="admin-button" disabled={loading || page <= 1} onClick={() => loadDocuments(page - 1)}>上一页</button>
                        <button className="admin-button" disabled={loading || page * pageSize >= total} onClick={() => loadDocuments(page + 1)}>下一页</button>
                    </div>
                </section>

                <aside className="admin-knowledge-side">
                    <section className="admin-panel">
                        <div className="admin-panel-head">
                            <div>
                                <h2>上传文档</h2>
                                <p>支持 PDF / DOCX / TXT / MD，v1 不做 OCR。</p>
                            </div>
                        </div>
                        <div className="admin-form simple-compose">
                            <input className="admin-input" value={uploadMeta.title} onChange={(e) => setUploadMeta((prev) => ({ ...prev, title: e.target.value }))} placeholder="标题，不填则使用文件名" />
                            <input className="admin-input" value={uploadMeta.source} onChange={(e) => setUploadMeta((prev) => ({ ...prev, source: e.target.value }))} placeholder="来源，例如学校官方通知" />
                            <select className="admin-select" value={uploadMeta.category} onChange={(e) => setUploadMeta((prev) => ({ ...prev, category: e.target.value }))}>
                                {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                            </select>
                            <div className="admin-form two">
                                <input className="admin-input" type="datetime-local" value={formatDateTimeLocal(uploadMeta.effective_at)} onChange={(e) => setUploadMeta((prev) => ({ ...prev, effective_at: e.target.value }))} />
                                <input className="admin-input" type="datetime-local" value={formatDateTimeLocal(uploadMeta.expired_at)} onChange={(e) => setUploadMeta((prev) => ({ ...prev, expired_at: e.target.value }))} />
                            </div>
                            <label className={`admin-upload-button ${working === 'upload' ? 'disabled' : ''}`}>
                                <FiUploadCloud />
                                {working === 'upload' ? '上传并索引中...' : '选择文件上传'}
                                <input type="file" accept=".pdf,.docx,.txt,.md,.markdown" onChange={uploadFile} disabled={working === 'upload'} />
                            </label>
                        </div>
                    </section>

                    <section className="admin-panel">
                        <div className="admin-panel-head">
                            <div>
                                <h2>手动录入</h2>
                                <p>适合报到时间、快递点、平台规则这类短资料。</p>
                            </div>
                        </div>
                        <div className="admin-form simple-compose">
                            <input className="admin-input" value={manual.title} onChange={(e) => setManual((prev) => ({ ...prev, title: e.target.value }))} placeholder="标题" />
                            <div className="admin-form two">
                                <input className="admin-input" value={manual.source} onChange={(e) => setManual((prev) => ({ ...prev, source: e.target.value }))} placeholder="来源" />
                                <select className="admin-select" value={manual.category} onChange={(e) => setManual((prev) => ({ ...prev, category: e.target.value }))}>
                                    {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                                </select>
                            </div>
                            <div className="admin-form two">
                                <input className="admin-input" type="datetime-local" value={formatDateTimeLocal(manual.effective_at)} onChange={(e) => setManual((prev) => ({ ...prev, effective_at: e.target.value }))} />
                                <input className="admin-input" type="datetime-local" value={formatDateTimeLocal(manual.expired_at)} onChange={(e) => setManual((prev) => ({ ...prev, expired_at: e.target.value }))} />
                            </div>
                            <textarea className="admin-textarea" value={manual.raw_content} onChange={(e) => setManual((prev) => ({ ...prev, raw_content: e.target.value }))} placeholder="录入已确认的校园信息..." />
                            <button className="admin-button primary" type="button" onClick={createManual} disabled={working === 'manual'}>
                                <FiCheckCircle />
                                {working === 'manual' ? '索引中...' : '保存并启用'}
                            </button>
                        </div>
                    </section>
                </aside>
            </div>}

            {mode !== 'documents' && <div className={`admin-knowledge-grid bottom ${mode === 'test' ? 'single' : ''}`}>
                {mode === 'full' && <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>片段预览</h2>
                            <p>{selectedDoc ? selectedDoc.title : '点击左侧文档查看切片内容。'}</p>
                        </div>
                        {selectedDoc && (
                            <button className="admin-button subtle" type="button" onClick={refreshSelectedDoc} disabled={!!working}>
                                <FiRefreshCw />
                                刷新
                            </button>
                        )}
                    </div>
                    <div className="admin-knowledge-chunks">
                        {!selectedDoc && <div className="admin-empty compact">未选择文档</div>}
                        {selectedDoc && !chunks.length && <div className="admin-empty compact">暂无片段</div>}
                        {chunks.map((chunk) => (
                            <article className="admin-knowledge-chunk" key={chunk.id}>
                                <span>#{Number(chunk.chunk_index) + 1} · {chunk.source}</span>
                                <p>{chunk.content}</p>
                            </article>
                        ))}
                    </div>
                </section>}

                <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>测试提问</h2>
                            <p>运营可以先问一遍，确认 e仔能不能命中正确资料。</p>
                        </div>
                        <button className="admin-button primary" type="button" onClick={runTestQuery} disabled={working === 'test'}>
                            <FiZap />
                            测试
                        </button>
                    </div>
                    <div className="admin-test-query-row">
                        <input className="admin-input" value={testQuery} onChange={(e) => setTestQuery(e.target.value)} placeholder="例如：新生报到要带什么？" />
                    </div>
                    {testResult && (
                        <div className="admin-rag-result">
                            <div className="admin-ai-task-head">
                                <span className={`admin-status ${testResult.need_knowledge ? 'status-1' : ''}`}>{testResult.need_knowledge ? '需要查库' : '无需查库'}</span>
                                <span>置信度 {Number(testResult.confidence || 0).toFixed(2)}</span>
                            </div>
                            {(testResult.chunks || []).map((chunk) => (
                                <article key={chunk.chunk_id}>
                                    <strong>{chunk.title || '未命名资料'}</strong>
                                    <span>{chunk.source} · {chunk.score}</span>
                                    <p>{excerpt(chunk.content, 180)}</p>
                                </article>
                            ))}
                            {!(testResult.chunks || []).length && <div className="admin-empty compact">没有命中资料</div>}
                        </div>
                    )}
                </section>
            </div>}

            {mode === 'full' && <section className="admin-panel">
                <div className="admin-panel-head">
                    <div>
                        <h2>最近 RAG 查询</h2>
                        <p>用于追溯 e仔回答有没有查库、命中了哪些资料。</p>
                    </div>
                </div>
                <div className="admin-table-wrap">
                    <table className="admin-table">
                        <thead>
                            <tr>
                                <th>时间</th>
                                <th>问题</th>
                                <th>命中</th>
                                <th>回答</th>
                                <th>耗时</th>
                            </tr>
                        </thead>
                        <tbody>
                            {!logs.length && (
                                <tr><td colSpan="5"><div className="admin-empty compact">暂无查询日志</div></td></tr>
                            )}
                            {logs.map((item) => (
                                <tr key={item.id}>
                                    <td>{item.created_at}</td>
                                    <td className="admin-title-cell">{excerpt(item.query, 80)}</td>
                                    <td>{item.need_knowledge ? `${Number(item.confidence || 0).toFixed(2)} / ${(item.hit_chunks || []).length}片段` : '未查库'}</td>
                                    <td>{excerpt(item.answer || item.error_message, 80)}</td>
                                    <td>{item.duration_ms || 0}ms</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </section>}
        </div>
    );
};

function categoryLabel(value) {
    return categories.find((item) => item[0] === value)?.[1] || value || '通用';
}

export default AdminKnowledge;
