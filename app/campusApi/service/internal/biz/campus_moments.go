package biz

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
	"lehu-video/pkg/apperror"
)

const (
	momentsPosterSize        = 1080
	momentsPosterImageHeight = 650
	momentsPosterQRSize      = 230
	momentsPackageMaxPosts   = 9
	momentsCandidateLimit    = 30
	momentsMaxSourceBytes    = 10 << 20
)

type CreateCampusMomentsPackageInput struct {
	UserID    string
	Date      string
	PostIDs   []int64
	RequestID string
}

type ListCampusMomentsCandidatesInput struct {
	UserID string
	Date   string
}

type CampusMomentsPackageOutput struct {
	PackageID    string                      `json:"package_id"`
	Date         string                      `json:"date"`
	Count        int                         `json:"count"`
	ExpiresAt    time.Time                   `json:"expires_at"`
	Caption      string                      `json:"caption"`
	Warnings     []string                    `json:"warnings"`
	Posts        []*CampusMomentsPackagePost `json:"posts"`
	DownloadPath string                      `json:"download_path"`
}

type CampusMomentsPackagePost struct {
	Slot           int       `json:"slot"`
	PostID         string    `json:"post_id"`
	Title          string    `json:"title"`
	Excerpt        string    `json:"excerpt"`
	CoverURL       string    `json:"cover_url"`
	CategoryName   string    `json:"category_name"`
	LikeCount      int64     `json:"like_count"`
	CommentCount   int64     `json:"comment_count"`
	CollectedCount int64     `json:"collected_count"`
	HeatScore      int64     `json:"heat_score"`
	CreatedAt      time.Time `json:"created_at"`
	ImagePath      string    `json:"image_path"`
}

type CampusMomentsPackageFile struct {
	Path     string
	Name     string
	MimeType string
}

func (uc *CampusUsecase) AdminCreateMomentsPackage(ctx context.Context, input *CreateCampusMomentsPackageInput) (*CampusMomentsPackageOutput, error) {
	if input == nil {
		input = &CreateCampusMomentsPackageInput{}
	}
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	day, start, end, err := parseMomentsDate(input.Date)
	if err != nil {
		return nil, err
	}
	tmpRoot := momentsTmpRoot()
	retention := momentsRetention()
	if err := cleanupMomentsPackages(tmpRoot, retention); err != nil {
		uc.log.WithContext(ctx).Warnf("cleanup moments packages failed: request_id=%s err=%v", input.RequestID, err)
	}
	if err := ensureMomentsImageHostConfigured(); err != nil {
		return nil, err
	}
	posts, err := uc.momentsPackagePosts(ctx, input, start, end)
	if err != nil {
		return nil, err
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moments posts failed: request_id=%s err=%v", input.RequestID, err)
	}

	packageID := uc.nextMomentsPackageID()
	packageDir := filepath.Join(tmpRoot, packageID)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return nil, apperror.Internal(err, "创建素材包失败")
	}

	output := &CampusMomentsPackageOutput{
		PackageID:    packageID,
		Date:         day,
		Caption:      momentsCaption(day),
		Warnings:     []string{},
		Posts:        []*CampusMomentsPackagePost{},
		ExpiresAt:    time.Now().Add(retention),
		DownloadPath: fmt.Sprintf("/campus/admin/moments/packages/%s/download.zip", packageID),
	}

	slot := 1
	for _, post := range posts {
		if post == nil || slot > momentsPackageMaxPosts {
			break
		}
		coverURL := momentsPostCover(post)
		if coverURL == "" {
			continue
		}
		cover, err := fetchMomentsSourceImage(ctx, coverURL)
		if err != nil {
			uc.log.WithContext(ctx).Warnf("skip moments post image: request_id=%s package_id=%s post_id=%d object=cover err=%v", input.RequestID, packageID, post.ID, err)
			output.Warnings = append(output.Warnings, fmt.Sprintf("帖子 %d 的封面图下载失败，已跳过", post.ID))
			continue
		}
		qrBytes, err := getMomentsPostQRCode(ctx, post.ID)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("generate moments qrcode failed: request_id=%s package_id=%s post_id=%d slot=%d err=%v", input.RequestID, packageID, post.ID, slot, err)
			return nil, err
		}
		qrImage, _, err := image.Decode(bytes.NewReader(qrBytes))
		if err != nil {
			return nil, apperror.DependencyUnavailable(err, "微信小程序码解析失败")
		}
		poster := renderMomentsPoster(post, cover, qrImage, slot, day)
		fileName := fmt.Sprintf("ezai-moments-%s-%02d.png", strings.ReplaceAll(day, "-", ""), slot)
		filePath := filepath.Join(packageDir, fileName)
		if err := writePNG(filePath, poster); err != nil {
			return nil, apperror.Internal(err, "写入朋友圈图片失败")
		}
		output.Posts = append(output.Posts, momentsPackagePostFromPost(post, slot, coverURL, packageID))
		slot++
	}

	if len(output.Posts) == 0 {
		_ = os.RemoveAll(packageDir)
		return nil, apperror.InvalidArgument("今天还没有可生成朋友圈素材的图片帖")
	}
	if len(output.Posts) < momentsPackageMaxPosts {
		output.Warnings = append(output.Warnings, fmt.Sprintf("今日可用图片帖不足 9 条，本次生成 %d 条", len(output.Posts)))
	}
	output.Count = len(output.Posts)
	if err := writeMomentsManifest(packageDir, output); err != nil {
		return nil, apperror.Internal(err, "写入素材包清单失败")
	}
	if err := writeMomentsZip(packageDir, output); err != nil {
		return nil, apperror.Internal(err, "打包朋友圈素材失败")
	}
	return output, nil
}

func (uc *CampusUsecase) AdminListMomentsCandidates(ctx context.Context, input *ListCampusMomentsCandidatesInput) ([]*CampusForumPost, error) {
	if input == nil {
		input = &ListCampusMomentsCandidatesInput{}
	}
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	_, start, end, err := parseMomentsDate(input.Date)
	if err != nil {
		return nil, err
	}
	posts, err := uc.repo.ListTopImagePostsByDate(ctx, start, end, momentsCandidateLimit)
	if err != nil {
		return nil, apperror.Internal(err, "获取朋友圈候选帖子失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moments candidates failed: err=%v", err)
	}
	return posts, nil
}

func (uc *CampusUsecase) momentsPackagePosts(ctx context.Context, input *CreateCampusMomentsPackageInput, start, end time.Time) ([]*CampusForumPost, error) {
	selectedIDs := normalizeMomentsSelectedPostIDs(input.PostIDs)
	if len(selectedIDs) == 0 {
		posts, err := uc.repo.ListTopImagePostsByDate(ctx, start, end, momentsCandidateLimit)
		if err != nil {
			return nil, apperror.Internal(err, "获取今日热帖失败")
		}
		return posts, nil
	}
	posts, err := uc.repo.ListPostsByIDs(ctx, selectedIDs, []int32{CampusAuditStatusVisible})
	if err != nil {
		return nil, apperror.Internal(err, "获取已选帖子失败")
	}
	out := make([]*CampusForumPost, 0, len(posts))
	for _, post := range posts {
		if post == nil || post.MediaType != CampusPostMediaImage || momentsPostCover(post) == "" {
			continue
		}
		out = append(out, post)
	}
	if len(out) == 0 {
		return nil, apperror.InvalidArgument("已选帖子里没有可生成素材的图片帖")
	}
	if len(out) < len(selectedIDs) {
		missing := len(selectedIDs) - len(out)
		return out, apperror.InvalidArgument(fmt.Sprintf("有 %d 个已选帖子不可用，请只选择正常展示的图片帖", missing))
	}
	return out, nil
}

func (uc *CampusUsecase) AdminGetMomentsImageFile(ctx context.Context, userID, packageID string, slot int) (*CampusMomentsPackageFile, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	if !validMomentsPackageID(packageID) || slot < 1 || slot > momentsPackageMaxPosts {
		return nil, apperror.InvalidArgument("素材包参数无效")
	}
	manifest, packageDir, err := readMomentsManifest(packageID)
	if err != nil {
		return nil, err
	}
	for _, post := range manifest.Posts {
		if post != nil && post.Slot == slot {
			name := fmt.Sprintf("ezai-moments-%s-%02d.png", strings.ReplaceAll(manifest.Date, "-", ""), slot)
			path := filepath.Join(packageDir, name)
			if _, err := os.Stat(path); err != nil {
				return nil, apperror.NotFound("朋友圈图片已过期，请重新生成")
			}
			return &CampusMomentsPackageFile{Path: path, Name: name, MimeType: "image/png"}, nil
		}
	}
	return nil, apperror.NotFound("朋友圈图片不存在")
}

func (uc *CampusUsecase) AdminGetMomentsZipFile(ctx context.Context, userID, packageID string) (*CampusMomentsPackageFile, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	if !validMomentsPackageID(packageID) {
		return nil, apperror.InvalidArgument("素材包参数无效")
	}
	manifest, packageDir, err := readMomentsManifest(packageID)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("ezai-moments-%s-%s.zip", strings.ReplaceAll(manifest.Date, "-", ""), packageID)
	path := filepath.Join(packageDir, name)
	if _, err := os.Stat(path); err != nil {
		return nil, apperror.NotFound("朋友圈素材包已过期，请重新生成")
	}
	return &CampusMomentsPackageFile{Path: path, Name: name, MimeType: "application/zip"}, nil
}

func (uc *CampusUsecase) nextMomentsPackageID() string {
	if uc.idGen != nil {
		return fmt.Sprintf("%d", uc.idGen.NextID())
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func parseMomentsDate(value string) (string, time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = time.Now().In(loc).Format("2006-01-02")
	}
	day, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return "", time.Time{}, time.Time{}, apperror.InvalidArgument("日期格式应为 YYYY-MM-DD")
	}
	return day.Format("2006-01-02"), day, day.AddDate(0, 0, 1), nil
}

func momentsTmpRoot() string {
	if value := strings.TrimSpace(os.Getenv("LEHU_ADMIN_MOMENTS_TMP_DIR")); value != "" {
		return value
	}
	return filepath.Join(os.TempDir(), "lehu-campus-moments")
}

func momentsRetention() time.Duration {
	value := strings.TrimSpace(os.Getenv("LEHU_ADMIN_MOMENTS_RETENTION_HOURS"))
	if value == "" {
		return 24 * time.Hour
	}
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return 24 * time.Hour
	}
	if hours > 168 {
		hours = 168
	}
	return time.Duration(hours) * time.Hour
}

func cleanupMomentsPackages(root string, retention time.Duration) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-retention)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
	return nil
}

func momentsCaption(day string) string {
	return fmt.Sprintf("%s 深汕校园e站今日热帖精选来了。保存这 9 张图发朋友圈，同学扫码就能直达小程序帖子详情。", day)
}

func momentsPostCover(post *CampusForumPost) string {
	if post == nil {
		return ""
	}
	if strings.TrimSpace(post.CoverURL) != "" {
		return strings.TrimSpace(post.CoverURL)
	}
	for _, imageURL := range post.Images {
		if strings.TrimSpace(imageURL) != "" {
			return strings.TrimSpace(imageURL)
		}
	}
	return ""
}

func normalizeMomentsSelectedPostIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if len(out) >= momentsPackageMaxPosts {
			break
		}
	}
	return out
}

func momentsPackagePostFromPost(post *CampusForumPost, slot int, coverURL, packageID string) *CampusMomentsPackagePost {
	return &CampusMomentsPackagePost{
		Slot:           slot,
		PostID:         fmt.Sprintf("%d", post.ID),
		Title:          post.Title,
		Excerpt:        truncateRunes(cleanPosterText(post.Content), 64),
		CoverURL:       coverURL,
		CategoryName:   post.CategoryName,
		LikeCount:      post.LikeCount,
		CommentCount:   post.CommentCount,
		CollectedCount: post.CollectedCount,
		HeatScore:      post.LikeCount*2 + post.CommentCount*4 + post.CollectedCount*5,
		CreatedAt:      post.CreatedAt,
		ImagePath:      fmt.Sprintf("/campus/admin/moments/packages/%s/images/%d.png", packageID, slot),
	}
}

func writeMomentsManifest(packageDir string, output *CampusMomentsPackageOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(packageDir, "manifest.json"), data, 0o644)
}

func readMomentsManifest(packageID string) (*CampusMomentsPackageOutput, string, error) {
	packageDir := filepath.Join(momentsTmpRoot(), packageID)
	data, err := os.ReadFile(filepath.Join(packageDir, "manifest.json"))
	if os.IsNotExist(err) {
		return nil, "", apperror.NotFound("朋友圈素材包已过期，请重新生成")
	}
	if err != nil {
		return nil, "", apperror.Internal(err, "读取朋友圈素材包失败")
	}
	var manifest CampusMomentsPackageOutput
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", apperror.Internal(err, "解析朋友圈素材包失败")
	}
	return &manifest, packageDir, nil
}

func writeMomentsZip(packageDir string, output *CampusMomentsPackageOutput) error {
	zipName := fmt.Sprintf("ezai-moments-%s-%s.zip", strings.ReplaceAll(output.Date, "-", ""), output.PackageID)
	file, err := os.Create(filepath.Join(packageDir, zipName))
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()
	if err := addStringToZip(zw, "朋友圈文案.txt", output.Caption+"\n"); err != nil {
		return err
	}
	for _, post := range output.Posts {
		if post == nil {
			continue
		}
		name := fmt.Sprintf("ezai-moments-%s-%02d.png", strings.ReplaceAll(output.Date, "-", ""), post.Slot)
		if err := addFileToZip(zw, filepath.Join(packageDir, name), name); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, path, name string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate
	target, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(target, source)
	return err
}

func addStringToZip(zw *zip.Writer, name, value string) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	target, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(target, value)
	return err
}

func fetchMomentsSourceImage(ctx context.Context, rawURL string) (image.Image, error) {
	if err := validateMomentsImageURL(rawURL); err != nil {
		return nil, err
	}
	reqURL := rewriteMomentsImageURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, momentsMaxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > momentsMaxSourceBytes {
		return nil, fmt.Errorf("image exceeds %d bytes", momentsMaxSourceBytes)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func ensureMomentsImageHostConfigured() error {
	if len(momentsAllowedHosts()) == 0 {
		return apperror.InvalidArgument("请配置 LEHU_ADMIN_MOMENTS_IMAGE_HOST_ALLOWLIST 或 COS_PUBLIC_CDN_BASE_URL")
	}
	return nil
}

func validateMomentsImageURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil || parsed.Host == "" {
		return fmt.Errorf("invalid image url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported image scheme %q", parsed.Scheme)
	}
	host := strings.ToLower(parsed.Hostname())
	if ip := net.ParseIP(host); ip != nil && !isAllowedLocalMomentsHost(host) {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("private image host not allowed")
		}
	}
	for _, allowed := range momentsAllowedHosts() {
		if hostMatchesAllowedMomentsHost(host, allowed) {
			return nil
		}
	}
	return fmt.Errorf("image host %q is not allowlisted", host)
}

func rewriteMomentsImageURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil || parsed.Host == "" {
		return rawURL
	}
	for _, item := range strings.Split(firstNonEmpty(os.Getenv("LEHU_ADMIN_MOMENTS_IMAGE_HOST_REWRITE"), os.Getenv("MINIO_PUBLIC_HOST_REWRITE")), ",") {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) != 2 {
			continue
		}
		publicHost := strings.TrimSpace(strings.ToLower(parts[0]))
		internalHost := strings.TrimSpace(parts[1])
		if publicHost == "" || internalHost == "" {
			continue
		}
		if strings.EqualFold(parsed.Host, publicHost) || strings.EqualFold(parsed.Hostname(), publicHost) {
			if strings.Contains(internalHost, "://") {
				internalURL, err := url.Parse(internalHost)
				if err != nil || internalURL.Host == "" {
					continue
				}
				parsed.Scheme = internalURL.Scheme
				parsed.Host = internalURL.Host
			} else {
				parsed.Host = internalHost
			}
			return parsed.String()
		}
	}
	return rawURL
}

func momentsAllowedHosts() []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	add := func(value string) {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			return
		}
		if strings.Contains(value, "://") {
			if parsed, err := url.Parse(value); err == nil {
				value = parsed.Hostname()
			}
		} else if strings.Contains(value, "/") {
			value = strings.Split(value, "/")[0]
		}
		if strings.Contains(value, ":") && !strings.HasPrefix(value, "[") {
			if host, _, err := net.SplitHostPort(value); err == nil {
				value = host
			}
		}
		value = strings.Trim(value, "[]")
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	for _, item := range strings.Split(os.Getenv("LEHU_ADMIN_MOMENTS_IMAGE_HOST_ALLOWLIST"), ",") {
		add(item)
	}
	add(os.Getenv("COS_PUBLIC_CDN_BASE_URL"))
	add(os.Getenv("LEHU_PUBLIC_MINIO_ENDPOINT"))
	if strings.ToLower(strings.TrimSpace(os.Getenv("LEHU_STORAGE_PROVIDER"))) != "cos" {
		add("localhost")
		add("127.0.0.1")
		add("minio")
	}
	return out
}

func hostMatchesAllowedMomentsHost(host, allowed string) bool {
	if allowed == "*" {
		return true
	}
	if strings.HasPrefix(allowed, "*.") {
		suffix := strings.TrimPrefix(allowed, "*")
		return strings.HasSuffix(host, suffix)
	}
	return host == allowed
}

func isAllowedLocalMomentsHost(host string) bool {
	for _, allowed := range momentsAllowedHosts() {
		if host == allowed && (host == "127.0.0.1" || host == "localhost" || host == "minio") {
			return true
		}
	}
	return false
}

var momentsWechatToken = struct {
	sync.Mutex
	token     string
	expiresAt time.Time
}{}

var momentsWechatHTTPClient = &http.Client{Timeout: 10 * time.Second}

func getMomentsPostQRCode(ctx context.Context, postID int64) ([]byte, error) {
	if envBoolTrue(os.Getenv("LEHU_ADMIN_MOMENTS_MOCK_QR")) {
		return encodeMockMomentsQRCode(fmt.Sprintf("id=%d", postID))
	}
	token, err := getWechatAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"scene": fmt.Sprintf("id=%d", postID),
		"page":  "pages/post-detail/post-detail",
		"width": 430,
		"env_version": firstNonEmpty(
			strings.TrimSpace(os.Getenv("WECHAT_MINIPROGRAM_QR_ENV_VERSION")),
			"release",
		),
		"check_path": !envBoolFalse(os.Getenv("WECHAT_MINIPROGRAM_QR_CHECK_PATH")),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token="+url.QueryEscape(token), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := momentsWechatHTTPClient.Do(req)
	if err != nil {
		return nil, apperror.DependencyUnavailable(err, "生成微信小程序码失败")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, apperror.DependencyUnavailable(err, "读取微信小程序码失败")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apperror.DependencyUnavailable(fmt.Errorf("wechat qrcode status %d", resp.StatusCode), "生成微信小程序码失败")
	}
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		var out struct {
			ErrCode int64  `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(data, &out); err == nil && out.ErrCode != 0 {
			return nil, apperror.DependencyUnavailable(fmt.Errorf("wechat qrcode error %d %s", out.ErrCode, out.ErrMsg), "生成微信小程序码失败")
		}
	}
	return data, nil
}

func getWechatAccessToken(ctx context.Context) (string, error) {
	momentsWechatToken.Lock()
	defer momentsWechatToken.Unlock()
	if momentsWechatToken.token != "" && time.Now().Before(momentsWechatToken.expiresAt) {
		return momentsWechatToken.token, nil
	}
	appID := strings.TrimSpace(os.Getenv("WECHAT_APP_ID"))
	secret := strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET"))
	if appID == "" || secret == "" {
		return "", apperror.DependencyUnavailable(fmt.Errorf("WECHAT_APP_ID/WECHAT_APP_SECRET missing"), "微信小程序码未配置")
	}
	endpoint := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + url.QueryEscape(appID) + "&secret=" + url.QueryEscape(secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := momentsWechatHTTPClient.Do(req)
	if err != nil {
		return "", apperror.DependencyUnavailable(err, "获取微信 access_token 失败")
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int64  `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := jsonDecode(resp.Body, &out); err != nil {
		return "", apperror.DependencyUnavailable(err, "解析微信 access_token 失败")
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return "", apperror.DependencyUnavailable(fmt.Errorf("wechat token error %d %s", out.ErrCode, out.ErrMsg), "获取微信 access_token 失败")
	}
	ttl := time.Duration(out.ExpiresIn-300) * time.Second
	if ttl < time.Minute {
		ttl = time.Minute
	}
	momentsWechatToken.token = out.AccessToken
	momentsWechatToken.expiresAt = time.Now().Add(ttl)
	return momentsWechatToken.token, nil
}

func renderMomentsPoster(post *CampusForumPost, cover image.Image, qr image.Image, slot int, day string) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, momentsPosterSize, momentsPosterSize))
	fillRect(canvas, image.Rect(0, 0, momentsPosterSize, momentsPosterSize), color.RGBA{248, 250, 252, 255})
	coverImg := resizeCover(cover, momentsPosterSize, momentsPosterImageHeight)
	stddraw.Draw(canvas, image.Rect(0, 0, momentsPosterSize, momentsPosterImageHeight), coverImg, image.Point{}, stddraw.Over)
	fillRect(canvas, image.Rect(0, momentsPosterImageHeight, momentsPosterSize, momentsPosterSize), color.White)

	titleFace := loadMomentsFontFace(48)
	bodyFace := loadMomentsFontFace(30)
	smallFace := loadMomentsFontFace(26)
	defer closeFontFace(titleFace)
	defer closeFontFace(bodyFace)
	defer closeFontFace(smallFace)

	drawLabel(canvas, fmt.Sprintf("NO.%d", slot), 52, 706, titleFace, color.RGBA{14, 165, 233, 255})
	title := firstNonEmpty(cleanPosterText(post.Title), "校园热帖")
	drawWrappedText(canvas, title, 52, 765, 650, 2, titleFace, color.RGBA{15, 23, 42, 255})
	excerpt := truncateRunes(cleanPosterText(post.Content), 56)
	if excerpt != "" {
		drawWrappedText(canvas, excerpt, 52, 882, 650, 2, bodyFace, color.RGBA{71, 85, 105, 255})
	}
	metric := fmt.Sprintf("点赞 %d  评论 %d  收藏 %d", post.LikeCount, post.CommentCount, post.CollectedCount)
	drawLabel(canvas, metric, 52, 1010, smallFace, color.RGBA{100, 116, 139, 255})
	drawLabel(canvas, "深汕校园e站 · e仔今日热帖", 52, 1046, smallFace, color.RGBA{148, 163, 184, 255})

	qrRect := image.Rect(790, 742, 790+momentsPosterQRSize, 742+momentsPosterQRSize)
	fillRect(canvas, qrRect.Inset(-18), color.RGBA{241, 245, 249, 255})
	fillRect(canvas, qrRect.Inset(-10), color.White)
	qrImg := resizeFit(qr, momentsPosterQRSize, momentsPosterQRSize)
	stddraw.Draw(canvas, qrRect, qrImg, image.Point{}, stddraw.Over)
	drawLabel(canvas, "扫码查看帖子", 802, 1018, smallFace, color.RGBA{15, 23, 42, 255})
	drawLabel(canvas, day, 828, 1052, smallFace, color.RGBA{148, 163, 184, 255})
	return canvas
}

func resizeCover(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}
	scaleW := float64(width) / float64(srcW)
	scaleH := float64(height) / float64(srcH)
	scale := scaleW
	if scaleH > scale {
		scale = scaleH
	}
	resizedW := int(float64(srcW)*scale + 0.5)
	resizedH := int(float64(srcH)*scale + 0.5)
	resized := image.NewRGBA(image.Rect(0, 0, resizedW, resizedH))
	draw.CatmullRom.Scale(resized, resized.Bounds(), src, bounds, draw.Over, nil)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	offsetX := (resizedW - width) / 2
	offsetY := (resizedH - height) / 2
	stddraw.Draw(dst, dst.Bounds(), resized, image.Point{X: offsetX, Y: offsetY}, stddraw.Src)
	return dst
}

func resizeFit(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(dst, dst.Bounds(), color.White)
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return dst
	}
	scaleW := float64(width) / float64(bounds.Dx())
	scaleH := float64(height) / float64(bounds.Dy())
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	resizedW := int(float64(bounds.Dx())*scale + 0.5)
	resizedH := int(float64(bounds.Dy())*scale + 0.5)
	rect := image.Rect((width-resizedW)/2, (height-resizedH)/2, (width-resizedW)/2+resizedW, (height-resizedH)/2+resizedH)
	draw.CatmullRom.Scale(dst, rect, src, bounds, draw.Over, nil)
	return dst
}

func fillRect(dst stddraw.Image, rect image.Rectangle, c color.Color) {
	stddraw.Draw(dst, rect, image.NewUniform(c), image.Point{}, stddraw.Src)
}

func loadMomentsFontFace(size float64) font.Face {
	candidates := []string{}
	if value := strings.TrimSpace(os.Getenv("LEHU_ADMIN_MOMENTS_FONT_PATH")); value != "" {
		candidates = append(candidates, value)
	}
	candidates = append(candidates,
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJKsc-Regular.otf",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/Supplemental/NISC18030.ttf",
	)
	for _, path := range candidates {
		face, err := loadFontFaceFromPath(path, size)
		if err == nil {
			return face
		}
	}
	return basicfont.Face7x13
}

func loadFontFaceFromPath(path string, size float64) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		collection, collectionErr := opentype.ParseCollection(data)
		if collectionErr != nil || collection.NumFonts() == 0 {
			return nil, err
		}
		f, err = collection.Font(0)
		if err != nil {
			return nil, err
		}
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func closeFontFace(face font.Face) {
	if face != nil {
		_ = face.Close()
	}
}

func drawLabel(dst stddraw.Image, text string, x, baseline int, face font.Face, c color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(text)
}

func drawWrappedText(dst stddraw.Image, text string, x, y, maxWidth, maxLines int, face font.Face, c color.Color) {
	lines := wrapPosterText(text, maxWidth, maxLines, face)
	lineHeight := face.Metrics().Height.Ceil()
	if lineHeight <= 0 {
		lineHeight = 36
	}
	baseline := y
	for _, line := range lines {
		drawLabel(dst, line, x, baseline, face, c)
		baseline += lineHeight + 8
	}
}

func wrapPosterText(text string, maxWidth, maxLines int, face font.Face) []string {
	text = cleanPosterText(text)
	if text == "" || maxLines <= 0 {
		return nil
	}
	d := &font.Drawer{Face: face}
	lines := make([]string, 0, maxLines)
	current := ""
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		next := current + string(r)
		if current != "" && d.MeasureString(next).Ceil() > maxWidth {
			lines = append(lines, current)
			current = string(r)
			if len(lines) >= maxLines {
				break
			}
			continue
		}
		current = next
	}
	if current != "" && len(lines) < maxLines {
		lines = append(lines, current)
	}
	if len(lines) == maxLines && text != "" {
		last := lines[len(lines)-1]
		for d.MeasureString(last+"...").Ceil() > maxWidth && utf8.RuneCountInString(last) > 1 {
			_, size := utf8.DecodeLastRuneInString(last)
			last = last[:len(last)-size]
		}
		lines[len(lines)-1] = last + "..."
	}
	return lines
}

func cleanPosterText(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.Join(strings.Fields(value), " ")
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "..."
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func encodeMockMomentsQRCode(scene string) ([]byte, error) {
	size := 430
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fillRect(img, img.Bounds(), color.White)
	sum := sha1.Sum([]byte(scene))
	for y := 0; y < 29; y++ {
		for x := 0; x < 29; x++ {
			idx := (x + y*29) % len(sum)
			if (sum[idx]>>uint((x+y)%8))&1 == 1 || isFinderPattern(x, y) {
				cell := size / 29
				rect := image.Rect(x*cell+8, y*cell+8, (x+1)*cell+8, (y+1)*cell+8)
				fillRect(img, rect, color.RGBA{15, 23, 42, 255})
			}
		}
	}
	return encodePNGBytes(img)
}

func isFinderPattern(x, y int) bool {
	return (x < 7 && y < 7) || (x > 21 && y < 7) || (x < 7 && y > 21)
}

func encodePNGBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

func validMomentsPackageID(packageID string) bool {
	packageID = strings.TrimSpace(packageID)
	if packageID == "" || len(packageID) > 64 {
		return false
	}
	for _, r := range packageID {
		if (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func envBoolTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
