package music

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// NetEase 网易云音乐。
type NetEase struct{}

// Source 实现 Searcher。
func (NetEase) Source() string { return "wyy" }

// Name 实现 Searcher。
func (NetEase) Name() string { return "网易云音乐" }

// NeedCookie 实现 Searcher。
func (NetEase) NeedCookie() bool { return false }

var neteaseHeaders = map[string]string{"Referer": "https://music.163.com/"}

// Search 实现 Searcher。
func (s NetEase) Search(ctx context.Context, keyword, singer, cookie string) (*Result, error) {
	searchURL := "https://music.163.com/api/search/get/web?s=" + queryEscape(keyword) +
		"&type=1&offset=0&limit=20"

	raw, err := doRequest(ctx, searchURL, httpOptions{
		Method:  http.MethodPost,
		Body:    "",
		Cookie:  cookie,
		Headers: neteaseHeaders,
	})
	if err != nil {
		return nil, err
	}
	data, err := parseJSON(raw)
	if err != nil {
		return nil, err
	}

	songs := arr(dig(data, "result", "songs"))
	if len(songs) == 0 {
		return nil, ErrNotFound
	}
	song := pickBestSong(songs, keyword, singer, func(m map[string]interface{}) string {
		return str(m["name"])
	}, func(m map[string]interface{}) string {
		return joinSingers(arr(m["artists"]), "name")
	})

	songID := str(song["id"])
	title := str(song["name"])
	singers := joinSingers(arr(song["artists"]), "name")
	cover, durationMS := s.detail(ctx, songID, cookie)
	rawLyric := s.lyric(ctx, songID, cookie)

	return &Result{
		Source:     s.Source(),
		SongID:     songID,
		Title:      title,
		Singer:     singers,
		Cover:      cover,
		Link:       "https://music.163.com/#/song?id=" + songID,
		MusicURL:   s.musicURL(ctx, songID, cookie),
		Lyric:      cleanLyric(rawLyric),
		LyricLRC:   rawLyric,
		DurationMS: durationMS,
	}, nil
}

// detail 通过歌曲详情接口获取封面与时长（旧专辑接口已不可用）。
func (NetEase) detail(ctx context.Context, songID, cookie string) (string, int64) {
	if songID == "" {
		return "", 0
	}

	data, err := getJSON(ctx, "https://music.163.com/api/song/detail?ids=["+songID+"]", cookie, neteaseHeaders)
	if err != nil {
		return "", 0
	}

	songs := arr(data["songs"])
	if len(songs) == 0 {
		return "", 0
	}
	song := obj(songs[0])
	if song == nil {
		return "", 0
	}

	cover := strings.Replace(str(dig(song, "album", "picUrl")), "http://", "https://", 1)

	var duration int64
	if v, ok := song["duration"].(float64); ok {
		duration = int64(v)
	}
	return cover, duration
}

// musicURL 获取播放地址，失败时回退到外链。
func (NetEase) musicURL(ctx context.Context, songID, cookie string) string {
	reqURL := fmt.Sprintf("https://music.163.com/api/song/enhance/player/url?ids=[%s]&br=320000", songID)
	if data, err := getJSON(ctx, reqURL, cookie, neteaseHeaders); err == nil {
		items := arr(data["data"])
		if len(items) > 0 {
			if u := str(dig(items[0], "url")); u != "" {
				return u
			}
		}
	}
	return "https://music.163.com/song/media/outer/url?id=" + songID + ".mp3"
}

// lyric 获取歌词。
func (NetEase) lyric(ctx context.Context, songID, cookie string) string {
	reqURL := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%s&lv=1&tv=1", songID)
	data, err := getJSON(ctx, reqURL, cookie, neteaseHeaders)
	if err != nil {
		return ""
	}
	return str(dig(data, "lrc", "lyric"))
}

var bracketRe = regexp.MustCompile(`[（(\[【][^）)\]】]*[）)\]】]`)

// normalizeTitle 去掉括号修饰、空格并转小写，用于标题比对。
func normalizeTitle(s string) string {
	s = bracketRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u3000", "")
	return strings.ToLower(strings.TrimSpace(s))
}

// singerMatch 判断候选歌手（可能是多歌手拼接）中是否包含目标歌手。
func singerMatch(singers, want string) bool {
	want = normalizeTitle(want)
	if want == "" {
		return true
	}

	parts := strings.FieldsFunc(singers, func(r rune) bool {
		return r == '/' || r == '&' || r == ',' || r == '、' || r == '，'
	})
	for _, part := range parts {
		if strings.Contains(normalizeTitle(part), want) {
			return true
		}
	}
	return false
}

// scoreSong 给候选歌曲打分：精确匹配优先，指定歌手优先，免费的优先，标题越干净越可能是原版。
func scoreSong(title, singers, keyword, wantSinger string, fee int) int {
	score := 0
	t, k := normalizeTitle(title), normalizeTitle(keyword)
	switch {
	case t == k:
		score += 100
	case strings.HasPrefix(t, k):
		score += 60
	case strings.Contains(t, k):
		score += 30
	case strings.Contains(k, t):
		score += 20
	}
	if wantSinger != "" {
		if strings.Contains(normalizeTitle(singers), normalizeTitle(wantSinger)) {
			score += 200 // 指定歌手，强烈优先
		} else {
			score -= 120
		}
	}
	switch fee {
	case 0:
		score += 50 // 免费，必定可播
	case 1:
		score += 20 // 会员可听
	}
	score -= len([]rune(title)) / 4
	return score
}

// pickBestSong 在候选列表中挑选最匹配的一条。
//
// titleOf / singerOf 用于适配各音乐源不同的字段名。
func pickBestSong(
	candidates []interface{},
	keyword, singer string,
	titleOf, singerOf func(map[string]interface{}) string,
) map[string]interface{} {
	// 指定歌手时先按歌手硬过滤，避免返回毫不相关的歌手
	if singer != "" {
		matched := make([]interface{}, 0, len(candidates))
		for _, item := range candidates {
			if song := obj(item); song != nil && singerMatch(singerOf(song), singer) {
				matched = append(matched, item)
			}
		}
		if len(matched) > 0 {
			candidates = matched
		}
	}

	var (
		best      map[string]interface{}
		bestScore = -1 << 30
	)
	for _, item := range candidates {
		song := obj(item)
		if song == nil {
			continue
		}
		fee := 0
		if v, ok := song["fee"].(float64); ok {
			fee = int(v)
		}
		if score := scoreSong(titleOf(song), singerOf(song), keyword, singer, fee); score > bestScore {
			best, bestScore = song, score
		}
	}
	if best == nil {
		return obj(candidates[0])
	}
	return best
}
