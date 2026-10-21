package music

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

// QiShui 汽水音乐（抖音音乐）。
//
// 汽水没有对外公开的免费 API（官方 open.qishui.com 需 OAuth + IP 白名单），
// 这里解析网页端 SSR 数据，无需 Cookie；若页面结构变动，替换下列常量即可。
type QiShui struct{}

const (
	qiShuiSearchPage = "https://music.douyin.com/qishui/search?keyword="
	qiShuiTrackPage  = "https://music.douyin.com/qishui/share/track?track_id="
)

var (
	qiShuiTrackIDRe = regexp.MustCompile(`(?:track_id|trackId|song_id)["']?\s*[:=]\s*["']?(\d{10,25})`)
	qiShuiRenderRe  = regexp.MustCompile(`(?is)id=["']RENDER_DATA["'][^>]*>(.*?)</script>`)
	qiShuiRouterRe  = regexp.MustCompile(`(?is)_ROUTER_DATA\s*=\s*(\{.*?\})\s*</script>`)
	audioURLRe      = regexp.MustCompile(`https?://[^"'\s\\]*douyinvod\.com[^"'\s\\]*`)
	coverURLRe      = regexp.MustCompile(`https?://[^"'\s\\]*byteimg\.com[^"'\s\\]*`)
)

// Source 实现 Searcher。
func (QiShui) Source() string { return "qishui" }

// Name 实现 Searcher。
func (QiShui) Name() string { return "汽水音乐" }

// NeedCookie 实现 Searcher。
func (QiShui) NeedCookie() bool { return false }

// Search 实现 Searcher。
//
// 汽水音乐的搜索页由前端渲染，抓不到结果，因此不支持按歌名搜索，
// 请改用 Detail（歌曲ID 或分享链接）。
func (QiShui) Search(ctx context.Context, _, _, _ string) (*Result, error) {
	return nil, ErrNoSearch
}

// ExtractTrackID 从裸 ID 或分享链接中提取 track_id。
func ExtractTrackID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isDigits(raw) && len(raw) >= 10 {
		return raw
	}
	if m := findTrackID(raw); m != "" {
		return m
	}
	// 形如 .../track/7531954496243927057
	for _, seg := range strings.Split(strings.ReplaceAll(raw, `\/`, "/"), "/") {
		if isDigits(seg) && len(seg) >= 10 {
			return seg
		}
	}
	return ""
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Detail 按 track_id 解析歌曲信息。
func (QiShui) Detail(ctx context.Context, trackID, cookie string) (*Result, error) {
	headers := map[string]string{"Referer": "https://music.douyin.com/"}
	raw, err := doRequest(ctx, qiShuiTrackPage+url.QueryEscape(trackID),
		httpOptions{Cookie: cookie, Headers: headers})
	if err != nil {
		return nil, err
	}
	html := string(raw)
	// JSON 中 / 会被转义为 \/
	text := strings.ReplaceAll(html, `\/`, "/")

	res := &Result{
		Source:   "qishui",
		SongID:   trackID,
		Link:     qiShuiTrackPage + trackID,
		MusicURL: firstMatch(audioURLRe, text),
		Cover:    firstMatch(coverURLRe, text),
	}

	if data := extractSSRData(html); data != nil {
		res.Title = deepFindString(data, []string{"title", "song_name", "track_name", "name"})
		res.Singer = deepFindString(data, []string{"author", "artist_name", "singer", "artist"})

		rawLyric := deepFindString(data, []string{"lyric", "lyrics", "lrc"})
		res.Lyric = cleanLyric(rawLyric)
		res.LyricLRC = rawLyric
	}

	if res.MusicURL == "" {
		return nil, ErrNoPlayURL
	}
	if res.Title == "" {
		res.Title = trackID
	}
	return res, nil
}

// ErrNoPlayURL 未能解析出播放地址。
var ErrNoPlayURL = &playURLError{}

type playURLError struct{}

func (*playURLError) Error() string { return "汽水音乐未解析出播放地址，页面结构可能已变更" }

// findTrackID 从搜索页 HTML 中提取 track_id。
func findTrackID(html string) string {
	text := strings.ReplaceAll(html, `\/`, "/")
	if m := qiShuiTrackIDRe.FindStringSubmatch(text); len(m) > 1 {
		return m[1]
	}
	return ""
}

func firstMatch(re *regexp.Regexp, text string) string {
	if m := re.FindString(text); m != "" {
		return m
	}
	return ""
}

// extractSSRData 提取页面内嵌的 SSR JSON。
func extractSSRData(html string) interface{} {
	candidates := make([]string, 0, 3)

	if m := qiShuiRenderRe.FindStringSubmatch(html); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		if decoded, err := url.QueryUnescape(raw); err == nil {
			candidates = append(candidates, decoded)
		}
		candidates = append(candidates, raw)
	}
	if m := qiShuiRouterRe.FindStringSubmatch(html); len(m) > 1 {
		candidates = append(candidates, strings.TrimSpace(m[1]))
	}

	for _, raw := range candidates {
		var data interface{}
		if err := json.Unmarshal([]byte(raw), &data); err == nil {
			return data
		}
	}
	return nil
}

// deepFindString 递归查找第一个匹配键名的非空字符串。
func deepFindString(v interface{}, keys []string) string {
	switch node := v.(type) {
	case map[string]interface{}:
		for _, key := range keys {
			if raw, ok := node[key]; ok {
				if s := str(raw); s != "" {
					return s
				}
			}
		}
		for _, child := range node {
			if s := deepFindString(child, keys); s != "" {
				return s
			}
		}
	case []interface{}:
		for _, child := range node {
			if s := deepFindString(child, keys); s != "" {
				return s
			}
		}
	}
	return ""
}
