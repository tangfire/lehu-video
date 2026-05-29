import { useEffect, useMemo, useRef, useState } from 'react';
import { FiCheck, FiChevronDown, FiImage, FiPlus, FiSend, FiUploadCloud, FiX } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { excerpt, postTypeText, toArrayFromLines } from './adminUtils';
import './Admin.css';

const initialForm = {
    category_code: 'guide',
    post_type: 'guide',
    media_type: 'text',
    title: '',
    content: '',
    images: '',
    cover_url: '',
    is_official: true,
    is_featured: true,
    is_pinned: true,
    sort_weight: 100,
};

const contentTemplates = [
    {
        name: '报到总攻略',
        badge: '置顶',
        category_code: 'guide',
        post_type: 'guide',
        title: '深汕校区新生报到前，先看这一篇',
        content: `先说结论：报到当天别靠临场发挥，提前把“证件、路线、宿舍、班级通知”四件事确认好。

报到前建议准备：
1. 身份证、录取通知书，放在随身证件袋里，不要托运或塞到行李箱深处。
2. 证件照、银行卡/缴费凭证、团员材料、个人档案等，按学校当年入学须知和辅导员通知准备。
3. 提前确认自己是深汕校区报到，报到日期、报到入口、学院/班级群通知都以学校最新通知为准。
4. 第一天行李尽量轻一点，优先带“马上会用”的东西，大件等宿舍情况确认后再买。

深汕e仔会把学校公开信息和大家问得最多的问题持续整理在首页。评论区可以直接问，后面会做成 FAQ。

信息来源参考：深圳职业技术大学招生信息网新生入学须知、学校官网学生入口。2026 级以学校最新通知为准。`,
        sort_weight: 300,
    },
    {
        name: '证件材料清单',
        badge: '收藏',
        category_code: 'guide',
        post_type: 'guide',
        title: '报到证件袋怎么装？新生别漏这几样',
        content: `建议你单独准备一个透明文件袋，报到当天随身带。

优先放：
1. 身份证原件。
2. 录取通知书。
3. 证件照若干张，电子版也备一份。
4. 学校要求提交的档案、团员/党员组织关系材料，按当年通知执行。
5. 缴费、银行卡、助学贷款或绿色通道相关材料，如自己涉及就提前整理。

不要把身份证、通知书、档案放进大行李箱托运；报到点、宿舍办理、身份核验都可能临时用到。

提醒：不同年份、不同培养层次的材料要求可能不完全一样，最终以录取通知书随附材料和学校招生信息网最新入学须知为准。`,
        sort_weight: 285,
    },
    {
        name: '宿舍 FAQ',
        badge: '高收藏',
        category_code: 'life',
        post_type: 'guide',
        title: '宿舍东西别乱买：这些先带，这些到校再说',
        content: `最容易踩坑的不是少买，而是买太早。

可以先带：
1. 换洗衣物、洗漱用品、拖鞋、毛巾。
2. 充电器、充电宝、排插是否允许以宿舍规定为准。
3. 常用药、创可贴、口罩、纸巾。
4. 少量衣架、收纳袋、雨伞。

建议到校确认后再买：
1. 床帘、蚊帐、床垫、床上桌。
2. 大件收纳架、置物架。
3. 台灯、吹风机等可能受宿舍用电规则影响的物品。

原因很简单：新校区宿舍楼栋、床位尺寸、用电规则、快递点安排，开学前后可能以学校最新通知为准。先别把快递寄爆，省钱也省心。`,
        sort_weight: 275,
    },
    {
        name: '交通路线',
        badge: '转发友好',
        category_code: 'life',
        post_type: 'guide',
        title: '第一次去深汕校区，路线先这样规划',
        content: `深汕校区在深汕特别合作区。第一次到校，建议不要只看地图终点，先确认“到站、转乘、报到入口”三件事。

出发前做这几步：
1. 先看学校招生信息网、学生入口、辅导员通知，有没有接站或集中报到安排。
2. 自行到校的同学，提前查高铁/大巴到站，再看最后一段怎么到校区。
3. 尽量白天到，行李多的话预留转乘和排队时间。
4. 到校当天把录取通知书、身份证放在随身位置，别压进行李箱。

深汕e仔后面会把“常见出发地到校路线”整理成评论区合集。你从哪里出发，可以在评论区留一下城市/车站。`,
        sort_weight: 255,
    },
    {
        name: '课表导入说明',
        badge: '功能引导',
        category_code: 'study',
        post_type: 'guide',
        title: '课表怎么查？开学后先认准学校教务系统',
        content: `大学不是固定班级课表，同一个专业的同学也可能因为选课不同，课表不完全一样。

正式课表请以学校教务系统为准。学校官网学生入口里有本科教务系统、专科教务系统、全校课表等入口，开学后建议先完成学校账号相关设置，再确认自己的课表。

深汕校园 e站的“课表”入口目前是内测演示链路，后续如果学校确认开放教务导入，我们会接入真实导入能力。导入时密码只用于本次请求，不会保存。

建议收藏这篇：开学后课程、教室、周次、单双周都要看清楚，别只看课程名。`,
        sort_weight: 230,
    },
    {
        name: '深汕校区认路',
        badge: '新校区',
        category_code: 'guide',
        post_type: 'guide',
        title: '深汕校区到底是什么情况？新生先了解这几点',
        content: `很多同学最关心：深汕校区是不是新校区？生活会不会不方便？哪些信息能提前确认？

目前可以确定的是：深汕校区是深圳职业技术大学推进的重要校区建设项目，面向深汕特别合作区发展布局。新校区开学初期，部分生活服务、快递点、路线、社团活动安排，可能会随着学校正式通知逐步完善。

新生最该关注三类信息：
1. 学校招生信息网：录取、新生入学、报到安排。
2. 学校官网学生入口：教务系统、课表、校园网、一网通办等。
3. 辅导员/学院通知：班级、宿舍、现场报到和临时安排。

深汕e仔会优先整理“确认过的信息”，不确定的会标注待确认。`,
        sort_weight: 225,
    },
    {
        name: '问答引导',
        badge: '拉互动',
        category_code: 'qa',
        post_type: 'question',
        title: '新生集中提问帖：你最想提前知道什么？',
        content: `这条专门给新生提问。

可以问：
1. 报到要带什么。
2. 宿舍东西怎么买。
3. 从哪个站到深汕校区更方便。
4. 课表、教务系统、校园网怎么用。
5. 社团、军训、快递、生活服务什么时候确认。

提问建议带上关键词，比如“宿舍”“交通”“校园网”“军训”，这样后面同学更容易搜到。

深汕e仔会把高频问题整理成系列攻略。能确认的直接答，暂时不能确认的会标注“待学校通知”。`,
        sort_weight: 210,
    },
    {
        name: '校园网提醒',
        badge: '高频问题',
        category_code: 'guide',
        post_type: 'guide',
        title: '校园网怎么用？先收藏，开学后照着排查',
        content: `学校官网已有校园网使用相关说明。一般来说，新生到校后要先完成学校账号/统一身份认证相关设置，再按学校信息化服务指引连接校园网。

开学后如果连不上，先检查：
1. 学校账号是否已激活。
2. 密码是否设置/修改成功。
3. 连接的是不是学校公布的校园网名称。
4. 认证页是否能打开。
5. 是否处在宿舍/教学区覆盖范围内。

不同校区、不同区域的网络安排可能有差异，深汕校区具体以学校信息中心、辅导员和现场服务通知为准。

你遇到问题时，可以在评论区写清设备、位置、报错页面，方便大家一起排查。`,
        sort_weight: 190,
    },
    {
        name: '快递说明',
        badge: '待确认',
        category_code: 'life',
        post_type: 'guide',
        title: '开学快递先别乱寄：尤其是大件',
        content: `新校区开学前后，快递点、收件地址、取件时间、宿舍楼信息都可能要等学校最终通知。

建议：
1. 被褥、收纳架、床垫这类大件不要太早寄。
2. 收件地址没确认前，不要直接写模糊地址。
3. 到校第一周先买必需品，尺寸相关物品等床位确认后再买。
4. 如果学校公布统一收件格式，严格按格式填写。

深汕e仔会在确认后更新“快递收件格式/取件点/常见快递问题”。这条先收藏，别让快递比你先到校还找不到地方。`,
        sort_weight: 180,
    },
    {
        name: '军训提醒',
        badge: '待确认',
        category_code: 'guide',
        post_type: 'guide',
        title: '军训先别焦虑：现在能准备的只有这些',
        content: `军训时间、地点、服装发放、请假流程等，必须以学校正式通知为准。现在不建议到处听“往年版本”。

可以提前准备：
1. 防晒用品。
2. 水杯。
3. 舒适鞋垫。
4. 常用药和创可贴。
5. 透气吸汗的内搭。

先别急着买太多“军训神器”，很多东西不一定允许带，也不一定用得上。

等学校正式通知出来后，深汕e仔会整理一版“确认版军训清单”。`,
        sort_weight: 170,
    },
    {
        name: '社团招新入口',
        badge: '社团',
        category_code: 'club',
        post_type: 'club',
        title: '深汕校区社团/组织招新，先来这里集合',
        content: `这条给社团、学生组织、兴趣小组和新生互相找到彼此。

如果你是负责人，可以按这个格式评论：
组织/社团名：
主要做什么：
适合什么同学：
预计招新时间：
后续联系渠道：

如果你是新生，可以直接说想找什么方向：运动、摄影、音乐、编程、志愿服务、辩论、英语、创业、学生组织都可以。

深汕e仔会把靠谱信息整理成“社团招新合集”，方便大家一次看完。`,
        sort_weight: 150,
    },
    {
        name: '失物招领模板',
        badge: '互助',
        category_code: 'lost',
        post_type: 'lost',
        title: '失物招领怎么发？照这个格式更容易找回',
        content: `发失物招领时，信息越清楚，找回概率越高。

推荐格式：
丢失/捡到：
物品名称：
时间：
地点：
明显特征：
联系方法：

注意：
1. 不要公开身份证号、完整手机号、宿舍号等隐私信息。
2. 贵重物品建议让对方补充特征确认。
3. 捡到校园卡、证件类物品，优先交给老师、宿管或学校相关服务点。

深汕e仔也会帮忙整理高优先级失物信息。`,
        sort_weight: 140,
    },
];

const AdminCompose = () => {
    const [form, setForm] = useState(initialForm);
    const [categories, setCategories] = useState([]);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [showAdvanced, setShowAdvanced] = useState(false);
    const fileInputRef = useRef(null);

    useEffect(() => {
        campusAdminApi.listCategories()
            .then((data) => setCategories(data.categories || []))
            .catch(() => setCategories([]));
    }, []);

    const images = useMemo(() => toArrayFromLines(form.images), [form.images]);
    const previewCover = form.cover_url || images[0] || '';

    const update = (key, value) => {
        setForm((prev) => ({ ...prev, [key]: value }));
    };

    const updateImages = (nextImages, nextCover = form.cover_url) => {
        setForm((prev) => ({
            ...prev,
            media_type: nextImages.length > 0 ? 'image' : (prev.media_type === 'image' ? 'text' : prev.media_type),
            images: nextImages.join('\n'),
            cover_url: nextCover || nextImages[0] || '',
        }));
    };

    const applyTemplate = (template) => {
        setForm((prev) => ({
            ...prev,
            ...template,
            media_type: 'text',
            images: '',
            cover_url: '',
            is_official: true,
            is_featured: true,
            is_pinned: true,
        }));
        setMessage(`已填充模板：${template.name}`);
        setError('');
    };

    const uploadImages = async (event) => {
        const files = Array.from(event.target.files || []);
        event.target.value = '';
        if (files.length === 0) return;
        if (images.length + files.length > 9) {
            setError('图文笔记最多 9 张图片');
            return;
        }
        setUploading(true);
        setError('');
        setMessage('');
        try {
            const urls = [];
            for (const file of files) {
                const data = await campusAdminApi.uploadImage(file);
                if (data.url) urls.push(data.url);
            }
            updateImages([...images, ...urls]);
            setMessage(`已上传 ${urls.length} 张图片`);
        } catch (err) {
            setError(err.message || '图片上传失败');
        } finally {
            setUploading(false);
        }
    };

    const removeImage = (url) => {
        const nextImages = images.filter((item) => item !== url);
        updateImages(nextImages, form.cover_url === url ? nextImages[0] || '' : form.cover_url);
    };

    const submit = async (event) => {
        if (event) event.preventDefault();
        setLoading(true);
        setError('');
        setMessage('');
        try {
            const nextImages = toArrayFromLines(form.images);
            const mediaType = form.media_type === 'text' && nextImages.length > 0 ? 'image' : form.media_type;
            await campusAdminApi.createPost({
                category_code: form.category_code,
                post_type: form.post_type,
                media_type: mediaType,
                title: form.title,
                content: form.content,
                images: nextImages,
                cover_url: form.cover_url || nextImages[0] || '',
                is_official: form.is_official,
                is_featured: form.is_featured,
                is_pinned: form.is_pinned,
                sort_weight: Number(form.sort_weight || 0),
            });
            setMessage('已发布到小程序首页');
            setForm(initialForm);
        } catch (err) {
            setError(err.message || '发布失败');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="admin-compose-page">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}
            <div className="admin-composer-grid simple">
                <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>发一篇深汕e仔官方帖</h2>
                            <p>模板、正文、配图。</p>
                        </div>
                        <button className="admin-button primary" type="button" disabled={loading || uploading} onClick={submit}>
                            <FiSend />
                            {loading ? '发布中...' : '发布'}
                        </button>
                    </div>

                    <div className="admin-step-title">1. 选一个模板</div>
                    <div className="admin-template-grid">
                        {contentTemplates.map((template) => (
                            <button className="admin-template-card" type="button" key={template.name} onClick={() => applyTemplate(template)}>
                                <span>{template.badge}</span>
                                <strong>{template.name}</strong>
                                <p>{excerpt(template.title, 28)}</p>
                            </button>
                        ))}
                    </div>

                    <div className="admin-step-title">2. 改成今天要发的内容</div>
                    <div className="admin-form simple-compose">
                        <div className="admin-field wide">
                            <label>标题</label>
                            <input className="admin-input" value={form.title} onChange={(e) => update('title', e.target.value)} maxLength={60} placeholder="建议 18-32 字，像小红书笔记标题一样直接有用" />
                        </div>
                        <div className="admin-field wide">
                            <label>正文</label>
                            <textarea className="admin-textarea tall" value={form.content} onChange={(e) => update('content', e.target.value)} maxLength={2000} placeholder="把最重要的信息放在前两行，方便首页卡片抓住新生注意力。" />
                        </div>
                    </div>

                    <div className="admin-upload-panel">
                        <div className="admin-upload-head">
                            <div>
                                <h3>3. 配图</h3>
                                <p>首图默认封面。</p>
                            </div>
                            <button className="admin-button" type="button" disabled={uploading || images.length >= 9} onClick={() => fileInputRef.current?.click()}>
                                <FiUploadCloud />
                                {uploading ? '上传中...' : '上传图片'}
                            </button>
                            <input ref={fileInputRef} type="file" accept="image/*" multiple hidden onChange={uploadImages} />
                        </div>
                        <div className="admin-upload-grid">
                            {images.map((url) => (
                                <div className="admin-upload-item" key={url}>
                                    <img src={url} alt="" />
                                    {form.cover_url === url && <span><FiCheck /> 封面</span>}
                                    <button type="button" onClick={() => removeImage(url)}><FiX /></button>
                                </div>
                            ))}
                            {images.length < 9 && (
                                <button className="admin-upload-add" type="button" disabled={uploading} onClick={() => fileInputRef.current?.click()}>
                                    <FiPlus />
                                    添加图片
                                </button>
                            )}
                        </div>
                    </div>

                    <button className="admin-advanced-toggle" type="button" onClick={() => setShowAdvanced((value) => !value)}>
                        高级发布设置
                        <FiChevronDown className={showAdvanced ? 'rotate' : ''} />
                    </button>
                    {showAdvanced && (
                        <div className="admin-advanced-box">
                            <div className="admin-form two">
                                <div className="admin-field">
                                    <label>版块</label>
                                    <select className="admin-select" value={form.category_code} onChange={(e) => update('category_code', e.target.value)}>
                                        {categories.length === 0 && <option value="guide">校园攻略</option>}
                                        {categories.map((item) => (
                                            <option key={item.code} value={item.code}>{item.name}</option>
                                        ))}
                                    </select>
                                </div>
                                <div className="admin-field">
                                    <label>笔记类型</label>
                                    <select className="admin-select" value={form.post_type} onChange={(e) => update('post_type', e.target.value)}>
                                        <option value="guide">攻略</option>
                                        <option value="question">问答</option>
                                        <option value="lost">失物招领</option>
                                        <option value="club">社团</option>
                                        <option value="note">普通笔记</option>
                                    </select>
                                </div>
                                <div className="admin-field">
                                    <label>媒体类型</label>
                                    <select className="admin-select" value={form.media_type} onChange={(e) => update('media_type', e.target.value)}>
                                        <option value="text">纯文字</option>
                                        <option value="image">图文</option>
                                    </select>
                                </div>
                                <div className="admin-field">
                                    <label>排序权重</label>
                                    <input className="admin-input" type="number" value={form.sort_weight} onChange={(e) => update('sort_weight', e.target.value)} />
                                </div>
                                <div className="admin-field">
                                    <label>封面 URL</label>
                                    <input className="admin-input" value={form.cover_url} onChange={(e) => update('cover_url', e.target.value)} placeholder="为空时自动取第一张图片" />
                                </div>
                            </div>
                        </div>
                    )}
                </section>

                <aside className="admin-compose-aside">
                    <section className="admin-panel">
                        <h2>默认发布方式</h2>
                        <div className="admin-simple-switch-row">
                            <button className={form.is_official ? 'admin-pill active' : 'admin-pill'} type="button" onClick={() => update('is_official', !form.is_official)}>官方</button>
                            <button className={form.is_featured ? 'admin-pill active' : 'admin-pill'} type="button" onClick={() => update('is_featured', !form.is_featured)}>精选</button>
                            <button className={form.is_pinned ? 'admin-pill active' : 'admin-pill'} type="button" onClick={() => update('is_pinned', !form.is_pinned)}>置顶</button>
                        </div>
                        <p className="admin-muted">攻略可置顶，公告可关闭。</p>
                    </section>

                    <section className="admin-preview-panel">
                        <div className="admin-phone-preview">
                            <div className="admin-preview-cover">
                                {previewCover ? (
                                    <img src={previewCover} alt="" />
                                ) : (
                                    <div><FiImage /> 文字笔记</div>
                                )}
                            </div>
                            <div className="admin-preview-body">
                                <h3>{form.title || '给新生看的实用标题'}</h3>
                                <p>{excerpt(form.content, 54) || '正文前两行会影响首页点击，建议直接写清楚能解决什么问题。'}</p>
                                <div className="admin-preview-author">
                                    <span>深汕e仔</span>
                                    {form.is_official && <em>官方</em>}
                                    {form.is_pinned && <em>置顶</em>}
                                    {form.is_featured && <em>精选</em>}
                                </div>
                                <div className="admin-muted">{postTypeText(form.post_type)} · 权重 {form.sort_weight || 0}</div>
                            </div>
                        </div>
                    </section>
                </aside>
            </div>
        </div>
    );
};

export default AdminCompose;
