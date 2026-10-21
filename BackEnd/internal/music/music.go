// Package music 实现多音乐源的按歌名搜索与解析。
package music

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound 未搜索到歌曲。
var ErrNotFound = errors.New("未找到歌曲")

// ErrNoCookie 该音乐源没有可用 Cookie。
var ErrNoCookie = errors.New("Cookie 池为空，请先在后台添加 Cookie")

// ErrNoSearch 该音乐源不支持按歌名搜索。
var ErrNoSearch = errors.New("该音乐源不支持按歌名搜索，请改用歌曲ID或分享链接")

// Detailer 支持按歌曲 ID 直接解析的音乐源。
type Detailer interface {
	Detail(ctx context.Context, trackID, cookie string) (*Result, error)
}

// Result 统一的点歌结果。
type Result struct {
	Source   string `json:"source"`
	SongID   string `json:"song_id,omitempty"`
	Title    string `json:"title"`
	Singer   string `json:"singer"`
	Cover    string `json:"cover"`
	Link     string `json:"link"`
	MusicURL string `json:"music_url"`
	Lyric    string `json:"lyric"`
	// DurationMS 歌曲时长（毫秒），0 表示未知。
	DurationMS int64 `json:"-"`
	// LyricLRC 带时间轴的原始歌词，为空表示不可用。
	LyricLRC string `json:"-"`
	// Note 面向调用方的补充提示（例如需要会员 Cookie），不参与 JSON 序列化。
	Note string `json:"-"`
}

// Searcher 音乐源接口。
type Searcher interface {
	// Source 返回内部标识，如 qqmusic。
	Source() string
	// Name 返回展示名称，如 QQ音乐。
	Name() string
	// NeedCookie 是否需要 Cookie。
	NeedCookie() bool
	// Search 按歌名搜索并解析出播放地址与歌词，singer 用于优先命中原唱。
	Search(ctx context.Context, keyword, singer, cookie string) (*Result, error)
}

// Registry 音乐源注册表。
type Registry struct {
	searchers map[string]Searcher
	order     []string
}

// NewRegistry 创建注册表并注册内置音乐源。
func NewRegistry() *Registry {
	r := &Registry{searchers: make(map[string]Searcher, 4)}
	r.Register(QQMusic{})
	r.Register(NetEase{})
	r.Register(KuGou{})
	r.Register(QiShui{})
	return r
}

// Register 注册音乐源。
func (r *Registry) Register(s Searcher) {
	if _, ok := r.searchers[s.Source()]; !ok {
		r.order = append(r.order, s.Source())
	}
	r.searchers[s.Source()] = s
}

// Get 按标识获取音乐源。
func (r *Registry) Get(source string) (Searcher, bool) {
	s, ok := r.searchers[source]
	return s, ok
}

// Sources 返回音乐源列表（保持注册顺序）。
func (r *Registry) Sources() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.order))
	for _, key := range r.order {
		s := r.searchers[key]
		out = append(out, map[string]interface{}{
			"key":         s.Source(),
			"name":        s.Name(),
			"need_cookie": s.NeedCookie(),
		})
	}
	return out
}

// ---------------------------------------------------------------- HTTP 工具

// httpClient 全局复用连接池。
var httpClient = &http.Client{Timeout: 12 * time.Second}

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type httpOptions struct {
	Method  string
	Body    string
	Cookie  string
	Headers map[string]string
}

func doRequest(ctx context.Context, rawURL string, opt httpOptions) ([]byte, error) {
	method := opt.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if opt.Body != "" {
		body = strings.NewReader(opt.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Accept", "*/*")
	if opt.Cookie != "" {
		req.Header.Set("Cookie", opt.Cookie)
	}
	for k, v := range opt.Headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("上游返回 HTTP %d", resp.StatusCode)
	}
	return data, nil
}

func getJSON(ctx context.Context, rawURL, cookie string, headers map[string]string) (map[string]interface{}, error) {
	data, err := doRequest(ctx, rawURL, httpOptions{Cookie: cookie, Headers: headers})
	if err != nil {
		return nil, err
	}
	return parseJSON(data)
}

// parseJSON 解析字节为 map。
func parseJSON(data []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("上游返回非 JSON 数据: %w", err)
	}
	return out, nil
}

// dig 按路径取嵌套字段。
func dig(v interface{}, keys ...string) interface{} {
	for _, k := range keys {
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// str 把任意 JSON 值转成字符串。
func str(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	}
	return ""
}

// arr 把 JSON 值转成切片。
func arr(v interface{}) []interface{} {
	if a, ok := v.([]interface{}); ok {
		return a
	}
	return nil
}

// obj 把 JSON 值转成对象。
func obj(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

var lyricTimeRe = regexp.MustCompile(`\[\d{2}:\d{2}[:.]\d{2,3}\]`)
var lyricTagRe = regexp.MustCompile(`\[(ti|ar|al|by|offset|kana|total|length):[^\]]*\]`)

// cleanLyric 去掉时间轴与标签行，过滤空行。
func cleanLyric(raw string) string {
	raw = lyricTimeRe.ReplaceAllString(raw, "")
	raw = lyricTagRe.ReplaceAllString(raw, "")

	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, strings.TrimRight(line, "\r"))
		}
	}
	return strings.Join(out, "\n")
}

// joinSingers 拼接多个歌手名。
func joinSingers(list []interface{}, key string) string {
	names := make([]string, 0, len(list))
	for _, item := range list {
		if name := str(dig(item, key)); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "/")
}

// queryEscape 便捷方法。
func queryEscape(s string) string { return url.QueryEscape(s) }

// cookieValue 从 Cookie 字符串中提取指定键的值。
func cookieValue(raw, key string) string {
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.IndexByte(part, '=')
		if idx <= 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(part[:idx]), key) {
			return strings.TrimSpace(part[idx+1:])
		}
	}
	return ""
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML 去掉上游返回的高亮标签（如酷狗的 <em>晴天</em>）。
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	return html.UnescapeString(htmlTagRe.ReplaceAllString(s, ""))
}
