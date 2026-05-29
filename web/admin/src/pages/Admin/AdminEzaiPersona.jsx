import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiCheckCircle, FiCopy, FiPlay, FiRefreshCw, FiRotateCcw, FiSave, FiSliders, FiZap } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const emptyPersona = {
    name: '',
    role: '',
    personality: '',
    tone: '',
    style_rules: '',
    safety_rules: '',
    no_knowledge_reply: '',
    fallback_reply: '',
    max_reply_chars: 140,
    prompt_version: 'ezai-persona-v1',
};

const reasonText = {
    no_high_confidence_knowledge: '知识库无高置信度命中，已使用无资料默认回复',
    model_disabled: 'AI 回复未配置，当前只展示默认回复',
    model_not_run: '未运行模型，当前只展示默认回复',
    empty_model_answer: '模型返回为空，已使用失败默认回复',
};

const fieldDefs = [
    { key: 'name', label: '名称', maxLength: 24, type: 'input' },
    { key: 'role', label: '身份', maxLength: 120, type: 'input' },
    { key: 'personality', label: '性格', maxLength: 120, type: 'textarea' },
    { key: 'tone', label: '语气', maxLength: 120, type: 'textarea' },
    { key: 'style_rules', label: '回答规则', maxLength: 360, type: 'textarea', wide: true },
    { key: 'safety_rules', label: '安全边界', maxLength: 360, type: 'textarea', wide: true },
    { key: 'no_knowledge_reply', label: '无资料默认回复', maxLength: 160, type: 'textarea', wide: true },
    { key: 'fallback_reply', label: '失败默认回复', maxLength: 160, type: 'textarea', wide: true },
];

const AdminEzaiPersona = () => {
    const [persona, setPersona] = useState(emptyPersona);
    const [savedPersona, setSavedPersona] = useState(emptyPersona);
    const [defaultPersona, setDefaultPersona] = useState(emptyPersona);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [preview, setPreview] = useState(null);
    const [testForm, setTestForm] = useState({
        question: '校园网怎么连？',
        post_title: '新生报到问题集中问',
        post_content: '大家可以在评论区问报到、宿舍、校园网这些问题。',
        use_knowledge: true,
        run_model: true,
    });

    const load = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.getEzaiPersona();
            const nextPersona = { ...emptyPersona, ...(data.persona || {}) };
            setPersona(nextPersona);
            setSavedPersona(nextPersona);
            setDefaultPersona({ ...emptyPersona, ...(data.default_persona || {}) });
        } catch (err) {
            setError(err.message || '获取 e仔人设失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load();
    }, []);

    const changed = useMemo(() => JSON.stringify({ ...persona, updated_at: undefined, updated_by: undefined }) !== JSON.stringify({ ...savedPersona, updated_at: undefined, updated_by: undefined }), [persona, savedPersona]);

    const updatePersona = (key, value) => {
        setPersona((prev) => ({ ...prev, [key]: value }));
    };

    const updateTest = (key, value) => {
        setTestForm((prev) => ({ ...prev, [key]: value }));
    };

    const save = async () => {
        setSaving(true);
        setError('');
        setMessage('');
        try {
            const data = await campusAdminApi.updateEzaiPersona(persona);
            const nextPersona = { ...emptyPersona, ...(data.persona || {}) };
            setPersona(nextPersona);
            setSavedPersona(nextPersona);
            setMessage('e仔人设已保存');
            window.setTimeout(() => setMessage(''), 2400);
        } catch (err) {
            setError(err.message || '保存 e仔人设失败');
        } finally {
            setSaving(false);
        }
    };

    const resetDefault = () => {
        setPersona(defaultPersona);
        setMessage('已恢复默认值，保存后生效');
        window.setTimeout(() => setMessage(''), 2400);
    };

    const runPreview = async () => {
        setTesting(true);
        setError('');
        setPreview(null);
        try {
            const data = await campusAdminApi.previewEzaiPersona(testForm);
            setPreview(data.preview || null);
        } catch (err) {
            setError(err.message || '测试 e仔回复失败');
        } finally {
            setTesting(false);
        }
    };

    const copyPrompt = async () => {
        if (!preview) return;
        const content = `SYSTEM\n${preview.system_prompt || ''}\n\nUSER\n${preview.user_prompt || ''}`;
        await navigator.clipboard.writeText(content);
        setMessage('提示词已复制');
        window.setTimeout(() => setMessage(''), 1800);
    };

    const fallbackReason = preview?.fallback_reason || '';
    const fallbackLabel = reasonText[fallbackReason] || fallbackReason;

    return (
        <div className="admin-ezai-persona-page">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}

            <section className="admin-ezai-status-strip">
                <FiSliders />
                <div>
                    <strong>{persona.name || '深汕e仔'}</strong>
                    <span>{persona.role || '校园 e站内容小伙伴'}</span>
                </div>
                <div className="admin-ezai-status-meta">
                    <span>{persona.updated_at ? `上次更新 ${persona.updated_at}` : '使用默认配置'}</span>
                    <span>{persona.max_reply_chars || 140} 字以内</span>
                </div>
                <button className="admin-icon-button" type="button" disabled={loading} onClick={load} title="刷新">
                    <FiRefreshCw className={loading ? 'spin' : ''} />
                </button>
            </section>

            <section className="admin-ezai-layout">
                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <span className="admin-kicker">PERSONA</span>
                            <h2>人设设定</h2>
                            <p>评论区 @e仔 生效。</p>
                        </div>
                    </div>

                    <div className="admin-form two">
                        {fieldDefs.map((field) => (
                            <label className={`admin-field ${field.wide ? 'wide' : ''}`} key={field.key}>
                                <span>{field.label}</span>
                                {field.type === 'input' ? (
                                    <input
                                        className="admin-input"
                                        value={persona[field.key] || ''}
                                        maxLength={field.maxLength}
                                        onChange={(event) => updatePersona(field.key, event.target.value)}
                                    />
                                ) : (
                                    <textarea
                                        className="admin-textarea compact"
                                        value={persona[field.key] || ''}
                                        maxLength={field.maxLength}
                                        onChange={(event) => updatePersona(field.key, event.target.value)}
                                    />
                                )}
                                <small>{(persona[field.key] || '').length}/{field.maxLength}</small>
                            </label>
                        ))}
                        <label className="admin-field">
                            <span>最大回复字数</span>
                            <input
                                className="admin-input"
                                type="number"
                                min="60"
                                max="220"
                                value={persona.max_reply_chars || 140}
                                onChange={(event) => updatePersona('max_reply_chars', Number(event.target.value))}
                            />
                        </label>
                        <label className="admin-field">
                            <span>提示词版本</span>
                            <input
                                className="admin-input"
                                value={persona.prompt_version || ''}
                                maxLength={40}
                                onChange={(event) => updatePersona('prompt_version', event.target.value)}
                            />
                        </label>
                    </div>

                    <div className="admin-ezai-actions">
                        <button className="admin-button" type="button" disabled={saving} onClick={resetDefault}>
                            <FiRotateCcw />
                            恢复默认
                        </button>
                        <button className="admin-button primary" type="button" disabled={saving} onClick={save}>
                            <FiSave />
                            {saving ? '保存中...' : '保存设定'}
                        </button>
                    </div>
                    {changed && <div className="admin-ezai-note"><FiAlertTriangle /> 当前有未保存修改。</div>}
                </div>

                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <span className="admin-kicker">REPLY PREVIEW</span>
                            <h2>回复预览</h2>
                            <p>按当前配置生成一条预览回复。</p>
                        </div>
                    </div>

                    <div className="admin-form">
                        <label className="admin-field">
                            <span>用户问题</span>
                            <input className="admin-input" value={testForm.question} onChange={(event) => updateTest('question', event.target.value)} />
                        </label>
                        <label className="admin-field">
                            <span>帖子标题</span>
                            <input className="admin-input" value={testForm.post_title} onChange={(event) => updateTest('post_title', event.target.value)} />
                        </label>
                        <label className="admin-field">
                            <span>帖子正文</span>
                            <textarea className="admin-textarea compact" value={testForm.post_content} onChange={(event) => updateTest('post_content', event.target.value)} />
                        </label>
                        <div className="admin-ezai-switches">
                            <label>
                                <input type="checkbox" checked={testForm.use_knowledge} onChange={(event) => updateTest('use_knowledge', event.target.checked)} />
                                检索知识库
                            </label>
                            <label>
                                <input type="checkbox" checked={testForm.run_model} onChange={(event) => updateTest('run_model', event.target.checked)} />
                                运行模型
                            </label>
                        </div>
                        <button className="admin-button primary" type="button" disabled={testing} onClick={runPreview}>
                            <FiPlay />
                            {testing ? '测试中...' : '生成预览'}
                        </button>
                    </div>

                    {preview && (
                        <div className="admin-ezai-preview">
                            <div className="admin-ezai-preview-head">
                                <span className={`admin-ai-pill ${preview.used_model ? 'ready' : 'missing'}`}>
                                    {preview.used_model ? <FiZap /> : <FiAlertTriangle />}
                                    {preview.used_model ? '模型回复' : '默认回复'}
                                </span>
                                {fallbackLabel && <span className="admin-ezai-reason">{fallbackLabel}</span>}
                            </div>
                            <div className="admin-ezai-reply">
                                <FiCheckCircle />
                                <p>{preview.reply || '暂无回复'}</p>
                            </div>
                            <div className="admin-ezai-mini-grid">
                                <span>知识库：{preview.knowledge?.need_knowledge ? '需要' : '未命中需求'}</span>
                                <span>置信度：{Number(preview.knowledge?.confidence || 0).toFixed(2)}</span>
                            </div>
                            <details className="admin-ezai-prompt">
                                <summary>查看提示词</summary>
                                <button className="admin-button subtle" type="button" onClick={copyPrompt}>
                                    <FiCopy />
                                    复制提示词
                                </button>
                                <pre>{preview.system_prompt}</pre>
                                <pre>{preview.user_prompt}</pre>
                            </details>
                        </div>
                    )}
                </div>
            </section>
        </div>
    );
};

export default AdminEzaiPersona;
