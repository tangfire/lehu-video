from __future__ import annotations

from datetime import date
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import mm
from reportlab.pdfbase.cidfonts import UnicodeCIDFont
from reportlab.pdfbase.pdfmetrics import registerFont
from reportlab.platypus import (
    Flowable,
    KeepTogether,
    PageBreak,
    Paragraph,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)


OUT = Path(__file__).with_name("campus_e站项目进度汇报_2026-05-30.pdf")


registerFont(UnicodeCIDFont("STSong-Light"))


PAGE_WIDTH, PAGE_HEIGHT = A4
MARGIN_X = 18 * mm
MARGIN_Y = 16 * mm
CONTENT_WIDTH = PAGE_WIDTH - MARGIN_X * 2

INK = colors.HexColor("#17202A")
MUTED = colors.HexColor("#5D6773")
LINE = colors.HexColor("#D7DCE2")
SOFT = colors.HexColor("#F5F7FA")
ACCENT = colors.HexColor("#C84B6A")
ACCENT_DARK = colors.HexColor("#8D2F46")
GREEN = colors.HexColor("#1F7A4D")
AMBER = colors.HexColor("#9A6A00")
BLUE = colors.HexColor("#2E5E8E")


styles = getSampleStyleSheet()
styles.add(
    ParagraphStyle(
        "CNTitle",
        fontName="STSong-Light",
        fontSize=24,
        leading=31,
        textColor=INK,
        alignment=TA_LEFT,
        spaceAfter=8,
    )
)
styles.add(
    ParagraphStyle(
        "CNSubtitle",
        fontName="STSong-Light",
        fontSize=10.5,
        leading=15,
        textColor=MUTED,
        alignment=TA_LEFT,
        spaceAfter=16,
    )
)
styles.add(
    ParagraphStyle(
        "CNH1",
        fontName="STSong-Light",
        fontSize=15,
        leading=20,
        textColor=ACCENT_DARK,
        spaceBefore=10,
        spaceAfter=8,
    )
)
styles.add(
    ParagraphStyle(
        "CNH2",
        fontName="STSong-Light",
        fontSize=11.5,
        leading=16,
        textColor=INK,
        spaceBefore=8,
        spaceAfter=5,
    )
)
styles.add(
    ParagraphStyle(
        "CNBody",
        fontName="STSong-Light",
        fontSize=9.6,
        leading=14.2,
        textColor=INK,
        spaceAfter=5,
    )
)
styles.add(
    ParagraphStyle(
        "CNMuted",
        fontName="STSong-Light",
        fontSize=8.8,
        leading=12.5,
        textColor=MUTED,
        spaceAfter=4,
    )
)
styles.add(
    ParagraphStyle(
        "CNCenter",
        fontName="STSong-Light",
        fontSize=9.2,
        leading=12,
        textColor=INK,
        alignment=TA_CENTER,
    )
)
styles.add(
    ParagraphStyle(
        "CNCell",
        fontName="STSong-Light",
        fontSize=8.2,
        leading=11.5,
        textColor=INK,
    )
)
styles.add(
    ParagraphStyle(
        "CNCellMuted",
        fontName="STSong-Light",
        fontSize=8,
        leading=11,
        textColor=MUTED,
    )
)


def p(text: str, style: str = "CNBody") -> Paragraph:
    return Paragraph(text, styles[style])


def bullet(text: str) -> Paragraph:
    return Paragraph(f"• {text}", styles["CNBody"])


def cell(text: str, muted: bool = False) -> Paragraph:
    return Paragraph(text, styles["CNCellMuted" if muted else "CNCell"])


class BadgeRow(Flowable):
    def __init__(self, items: list[tuple[str, str, colors.Color]], width: float):
        super().__init__()
        self.items = items
        self.width = width
        self.height = 20 * mm

    def draw(self):
        gap = 5 * mm
        box_w = (self.width - gap * (len(self.items) - 1)) / len(self.items)
        for i, (label, value, color) in enumerate(self.items):
            x = i * (box_w + gap)
            self.canv.setFillColor(colors.white)
            self.canv.setStrokeColor(LINE)
            self.canv.roundRect(x, 0, box_w, self.height, 4, stroke=1, fill=1)
            self.canv.setFillColor(color)
            self.canv.setFont("STSong-Light", 15)
            self.canv.drawString(x + 5 * mm, 10.5 * mm, value)
            self.canv.setFillColor(MUTED)
            self.canv.setFont("STSong-Light", 7.8)
            self.canv.drawString(x + 5 * mm, 4.5 * mm, label)


def table(data, col_widths, header=True):
    rows = []
    for r, row in enumerate(data):
        rows.append([cell(str(x)) for x in row])
    t = Table(rows, colWidths=col_widths, hAlign="LEFT", repeatRows=1 if header else 0)
    commands = [
        ("GRID", (0, 0), (-1, -1), 0.35, LINE),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("LEFTPADDING", (0, 0), (-1, -1), 5),
        ("RIGHTPADDING", (0, 0), (-1, -1), 5),
        ("TOPPADDING", (0, 0), (-1, -1), 5),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
    ]
    if header:
        commands.extend(
            [
                ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#EEF1F5")),
                ("TEXTCOLOR", (0, 0), (-1, 0), INK),
            ]
        )
    t.setStyle(TableStyle(commands))
    return t


def status_table():
    data = [
        ["模块", "当前状态", "说明"],
        ["小程序主链路", "基本完成", "微信登录、社区发帖、图片上传、评论、点赞收藏、通知、e仔入口已收口为校园 e站。"],
        ["运营后台", "基本完成", "内容审核、知识库、e仔助手、权限、安全、朋友圈九图素材、审核设置等已整合进主项目。"],
        ["后端服务", "基本完成", "保留 api/base/campus-user/campus-rag，旧短视频、chat、Kafka/WebSocket 已从首发链路移除。"],
        ["媒体存储", "方案完成", "生产公开图片改为 COS + CDN；MinIO 仅用于本地开发和低频内部文件过渡。"],
        ["监控告警", "方案完成", "Grafana + Loki + Alloy + Prometheus + health-exporter + 飞书告警已形成浏览器内排障链路。"],
        ["上线准备", "待实机验收", "还需要在真实服务器跑生产 compose、配置微信/COS/CDN/域名，并做灰度。"],
    ]
    return table(data, [28 * mm, 25 * mm, CONTENT_WIDTH - 53 * mm])


def architecture_table():
    data = [
        ["层级", "组件", "职责"],
        ["用户入口", "微信小程序", "学生侧社区、发布、互动、通知、e仔问答。"],
        ["运营入口", "admin-web", "内容管理、审核、安全、知识库、e仔配置、素材生成。"],
        ["业务入口", "campus-api", "统一 HTTP、权限、校园业务、审核、通知、e仔任务编排。"],
        ["基础服务", "base", "账号、文件预签名上传、对象存储确认。"],
        ["用户资料", "campus-user", "用户资料、搜索、统计、最后在线时间。"],
        ["AI/RAG", "campus-rag + Qdrant", "资料解析、切片、embedding、向量检索和知识库测试。"],
        ["数据依赖", "MySQL + Redis", "MySQL 为最终数据源；Redis 用于真实 IP 限流和短 TTL 热点读缓存。"],
        ["观测告警", "Grafana/Loki/Alloy/Prometheus/飞书", "查 request_id、看 up/down、接收关键故障通知。"],
    ]
    return table(data, [24 * mm, 38 * mm, CONTENT_WIDTH - 62 * mm])


def cost_table():
    data = [
        ["项目", "首发建议", "控制逻辑"],
        ["轻量服务器", "2核4G / 100GB / 7Mbps / 1000GB/月", "Go 服务、MySQL、Redis、监控、RAG 小规模同机部署；不建议 2G 内存。"],
        ["公开媒体", "腾讯云 COS + CDN", "图片上传/下载不占服务器出网，避免 3-7Mbps 带宽堵住 API。"],
        ["视频", "首发关闭", "降低带宽、审核、恶意刷流量和存储成本风险。"],
        ["数据库", "自建 MySQL", "省成本，但需要关注备份、磁盘、慢查询、权限和迁移。首发数据价值不高时可先轻备份。"],
        ["日志", "Loki 留存 + Docker 日志限额", "避免日志无限吃磁盘；按 request_id 保留近期排障能力。"],
    ]
    return table(data, [27 * mm, 50 * mm, CONTENT_WIDTH - 77 * mm])


def risk_table():
    data = [
        ["风险", "当前控制", "上线前动作"],
        ["微信提审材料不完整", "已有协议/规范入口和提审文档", "隐私保护指引填真实内容，覆盖头像、相册/相机、网络请求、反馈、内容发布。"],
        ["图片流量打满服务器", "生产改 COS + CDN，上传直传，关闭中转 fallback", "检查微信合法域名、COS CORS、CDN 回源和防盗链。"],
        ["线上问题难定位", "request_id + Loki 日志搜索 + 健康面板 + 飞书告警", "做一次停服务告警演练和 request_id 排障演练。"],
        ["MySQL 压力与磁盘增长", "Redis 热点缓存、access_log 保留期、Docker 日志限额", "观察慢查询和磁盘，灰度期间记录真实容量。"],
        ["AI 回复不稳定", "e仔人设、默认回答策略、RAG 查询日志、质量标注和撤回", "上线初期人工抽查 e仔回复，逐步补知识库。"],
        ["内容安全", "审核模式、举报反馈、后台处理、视频关闭", "首发建议人工审核或 AI 初审 + 人工复核，不直接全放开。"],
    ]
    return table(data, [35 * mm, 55 * mm, CONTENT_WIDTH - 90 * mm])


def roadmap_table():
    data = [
        ["阶段", "目标", "验收标准"],
        ["T-3 至 T-1 天", "生产环境部署和全链路 smoke", "API、后台、小程序、COS/CDN、Grafana、飞书告警全部跑通。"],
        ["灰度 1", "10-30 人试用 2-3 天", "发帖、图片、评论、审核、e仔、后台处理无阻塞性问题。"],
        ["灰度 2", "扩到 100 人左右", "观察 MySQL、Redis、服务器内存、CDN 流量、告警噪音。"],
        ["首发", "面向 400 首批用户开放", "保持视频关闭，运营每日巡检后台、日志、告警和用户反馈。"],
        ["首发后", "按真实需求补功能", "优先考虑校园日历/失物招领/地点百科等低风险高频功能。"],
    ]
    return table(data, [28 * mm, 55 * mm, CONTENT_WIDTH - 83 * mm])


def draw_header_footer(canvas, doc):
    canvas.saveState()
    canvas.setFont("STSong-Light", 8)
    canvas.setFillColor(MUTED)
    canvas.drawString(MARGIN_X, PAGE_HEIGHT - 9 * mm, "校园 e站项目进度汇报")
    canvas.drawRightString(PAGE_WIDTH - MARGIN_X, 9 * mm, f"{doc.page}")
    canvas.setStrokeColor(LINE)
    canvas.line(MARGIN_X, PAGE_HEIGHT - 11 * mm, PAGE_WIDTH - MARGIN_X, PAGE_HEIGHT - 11 * mm)
    canvas.restoreState()


def section(title: str, elements: list):
    return [p(title, "CNH1"), *elements]


def build():
    doc = SimpleDocTemplate(
        str(OUT),
        pagesize=A4,
        rightMargin=MARGIN_X,
        leftMargin=MARGIN_X,
        topMargin=18 * mm,
        bottomMargin=15 * mm,
        title="校园 e站项目进度汇报",
        author="Codex",
    )
    story = []

    story += [
        Spacer(1, 18 * mm),
        p("校园 e站项目进度汇报", "CNTitle"),
        p("面向合伙人同步：当前完成度、架构设计、成本控制、上线准备与剩余风险", "CNSubtitle"),
        BadgeRow(
            [
                ("项目定位", "校园社区 + 运营后台 + e仔", ACCENT_DARK),
                ("首发范围", "文字/图片，不开放视频", BLUE),
                ("上线状态", "接近首发，待生产验收", GREEN),
            ],
            CONTENT_WIDTH,
        ),
        Spacer(1, 9 * mm),
        p(
            "截至 2026-05-30，项目已经从旧短视频/chat 栈收口为校园 e站：核心链路、后台运营、媒体存储、监控告警、AI/RAG 知识库和上线文档均已形成首发版本。当前最重要的工作不是继续堆功能，而是在真实服务器完成生产配置、微信提审材料和灰度验收。",
            "CNBody",
        ),
        Spacer(1, 5 * mm),
        table(
            [
                ["一句话结论", "项目主体已经具备初步上线基础，但需要完成生产环境全链路验收后再公开推广。"],
                ["首发策略", "低成本、小范围、可排障：先做文字/图片校园社区，视频和高风险能力暂缓。"],
                ["推荐节奏", "生产部署 → 10-30 人灰度 → 100 人灰度 → 400 首批用户。"],
            ],
            [30 * mm, CONTENT_WIDTH - 30 * mm],
            header=False,
        ),
        PageBreak(),
    ]

    story += section(
        "1. 当前完成度",
        [
            p("整体看，校园 e站已经从“能跑功能”推进到“可以准备上线验收”的阶段。现在的重点是稳定、成本和运营闭环。"),
            status_table(),
        ],
    )

    story += section(
        "2. 产品边界与核心功能",
        [
            bullet("学生端：微信小程序承载校园社区、发帖、图片、评论、点赞收藏、通知、e仔问答。"),
            bullet("运营端：后台集中处理内容审核、举报反馈、权限、安全面板、e仔助手、知识库和朋友圈九图素材。"),
            bullet("AI 能力：e仔有可配置人设、默认回答策略、知识库测试、回复状态、质量标注和撤回能力。"),
            bullet("内容策略：首发只支持文字/图片，后端固定拒绝视频，减少带宽、审核和恶意刷流量风险。"),
            bullet("增长功能储备：后续可优先做校园日历/考试比赛倒计时、失物招领、校园地点百科等低成本高频功能。"),
        ],
    )

    story += section(
        "3. 架构设计",
        [
            p("当前架构按“小团队可运维、低成本首发、浏览器内排障”设计。旧短视频、IM chat、Kafka、WebSocket 不再作为首发运行栈。"),
            architecture_table(),
        ],
    )

    story += section(
        "4. 媒体存储与成本控制",
        [
            p("最关键的成本调整是：生产公开图片不再走服务器本地 MinIO 或 API 中转，而是走 COS + CDN。服务器只负责签名、鉴权和写数据库，图片上传/下载流量交给云存储和 CDN。"),
            cost_table(),
        ],
    )

    story += [
        PageBreak(),
        p("5. 监控、日志与告警", "CNH1"),
        p("排障目标是尽量不进服务器、不逐个容器翻日志。用户报错时拿 request_id 到 Grafana 查 Loki；服务异常时看 Prometheus 健康面板；关键故障通过飞书群机器人通知。"),
        table(
            [
                ["组件", "用途"],
                ["Alloy", "采集 Docker 容器日志并发送给 Loki。"],
                ["Loki", "存日志，支持按 request_id、接口路径、错误关键词搜索。"],
                ["health-exporter", "探测 API、MySQL、Redis、RAG、Qdrant 等目标是否可用。"],
                ["Prometheus", "抓取健康指标，回答哪个组件 up/down、持续多久。"],
                ["Grafana", "统一看日志、健康面板和告警规则。"],
                ["alert-webhook + 飞书", "把 Grafana 告警转换成飞书群消息。"],
            ],
            [35 * mm, CONTENT_WIDTH - 35 * mm],
        ),
        p("当前告警范围保持保守：API、readyz、MySQL、Redis、health-exporter 属于 critical；base、campus-user、campus-rag、Qdrant、Consul、MinIO 属于 warning。暂不加业务指标告警，避免首发初期噪音过大。"),
    ]

    story += section(
        "6. AI 与 RAG 知识库",
        [
            p("e仔不是直接让大模型凭空回答，而是由 API 编排任务、人设和安全策略；需要校园事实时先查 RAG 知识库，再生成回复。"),
            table(
                [
                    ["模块", "设计"],
                    ["人设配置", "后台可配置名字、角色、性格、语气、安全边界、默认回复和字数。"],
                    ["知识库", "MySQL 存文档和切片预览，Qdrant 存向量索引，campus-rag 负责解析/切片/embedding/检索。"],
                    ["后台测试", "人设预览测试完整回复链路；知识库测试只看检索命中、置信度和片段。"],
                    ["质量闭环", "e仔回复任务和 RAG 查询日志可在后台查看，可标注 good/needs_fix/wrong/unsafe，并可撤回回复。"],
                    ["降级策略", "模型或 RAG 不可用时给默认回复，不阻塞发帖评论主链路。"],
                ],
                [35 * mm, CONTENT_WIDTH - 35 * mm],
            ),
        ],
    )

    story += section(
        "7. 安全与合规准备",
        [
            risk_table(),
            p("这里最容易被低估的是微信提审材料。小程序里虽然已有协议/规范入口，但提审前要把隐私保护指引填成真实内容，并确保权限说明和实际功能一致。"),
        ],
    )

    story += [
        PageBreak(),
        p("8. 上线前剩余事项", "CNH1"),
        KeepTogether(
            [
                p("必须完成", "CNH2"),
                bullet("用 docker-compose.yml + docker-compose.prod.yml 在真实服务器起完整生产栈。"),
                bullet("配置 HTTPS 反向代理、API 域名、后台域名、Grafana 域名、COS 上传域名、CDN 下载域名。"),
                bullet("关闭 mock 登录和上传中转 fallback，配置真实管理员 user_id。"),
                bullet("跑通登录、发帖、图片上传、评论、点赞收藏、审核、e仔、知识库、朋友圈素材。"),
                bullet("测试 Grafana 日志搜索、健康面板、飞书 firing/resolved 告警。"),
                bullet("确认微信隐私保护指引、用户协议、社区规范、举报入口和合法域名。"),
            ]
        ),
        KeepTogether(
            [
                p("建议完成", "CNH2"),
                bullet("首发开启人工审核或 AI 初审 + 人工复核，先不要完全放开。"),
                bullet("灰度期间每天看后台安全统计、access_log、Loki 错误日志和服务器磁盘。"),
                bullet("整理一批高质量校园资料进入 e仔知识库，例如报到、宿舍、校园网、快递、教务常见问题。"),
            ]
        ),
        p("9. 推荐上线节奏", "CNH1"),
        roadmap_table(),
    ]

    story += [
        p("10. 合伙人决策点", "CNH1"),
        table(
            [
                ["决策项", "建议"],
                ["是否继续加功能", "不建议上线前继续大改功能。当前应收口验收，新增功能放到首发后按真实反馈排序。"],
                ["是否先买高配服务器", "不用。首发 2核4G + COS/CDN 更符合成本控制；出现真实瓶颈再升级。"],
                ["是否开放视频", "不建议。视频会显著增加审核、带宽、防刷和预算风险。"],
                ["是否全校推广", "不建议一步到位。先 10-30 人灰度，再扩到 100，稳定后再面向 400 首批用户。"],
                ["下一步最关键", "完成生产部署、微信提审准备和全链路 smoke。"],
            ],
            [40 * mm, CONTENT_WIDTH - 40 * mm],
        ),
        Spacer(1, 6 * mm),
        p("结论：校园 e站已经从旧项目重构为可首发的校园社区产品。当前最大价值不在继续堆功能，而在把生产环境、审核合规、监控告警和灰度运营跑顺。", "CNH2"),
    ]

    doc.build(story, onFirstPage=draw_header_footer, onLaterPages=draw_header_footer)
    print(OUT)


if __name__ == "__main__":
    build()
