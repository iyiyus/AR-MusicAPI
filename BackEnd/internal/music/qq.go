package music

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
)

// QQMusic QQ 音乐。
type QQMusic struct{}

// Source 实现 Searcher。
func (QQMusic) Source() string { return "qqmusic" }

// Name 实现 Searcher。
func (QQMusic) Name() string { return "QQ音乐" }

// NeedCookie 实现 Searcher。
func (QQMusic) NeedCookie() bool { return false }

// Search 实现 Searcher。
func (s QQMusic) Search(ctx context.Context, keyword, singer, cookie string) (*Result, error) {
	searchURL := "https://c.y.qq.com/soso/fcgi-bin/client_search_cp?w=" + queryEscape(keyword) +
		"&p=1&n=20&format=json"
	headers := map[string]string{"Referer": "https://y.qq.com/"}

	data, err := getJSON(ctx, searchURL, cookie, headers)
	if err != nil {
		return nil, err
	}

	list := arr(dig(data, "data", "song", "list"))
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	song := pickBestSong(list, keyword, singer, func(m map[string]interface{}) string {
		return str(m["songname"])
	}, func(m map[string]interface{}) string {
		return joinSingers(arr(m["singer"]), "name")
	})

	songmid := str(song["songmid"])
	albummid := str(song["albummid"])
	title := str(song["songname"])
	singers := joinSingers(arr(song["singer"]), "name")

	musicURL := s.vkey(ctx, songmid, cookie)

	var durationMS int64
	if v, ok := song["interval"].(float64); ok && v > 0 {
		durationMS = int64(v) * 1000
	}

	rawLyric := s.lyric(ctx, songmid, cookie)

	note := ""
	if musicURL == "" {
		note = "QQ音乐未返回播放直链：Cookie 登录态有效，但 vkey 返回 invalidq，" +
			"通常是该账号未开通绿钻或 Cookie 缺少 skey。封面/歌词/详情页仍可正常使用。"
	}

	// 没有专辑 mid 时不拼地址，避免返回 404 的封面
	cover := ""
	if albummid != "" {
		cover = "https://y.gtimg.cn/music/photo_new/T002R300x300M000" + albummid + ".jpg"
	}

	return &Result{
		Source:     s.Source(),
		SongID:     str(song["songid"]),
		Title:      title,
		Singer:     singers,
		Cover:      cover,
		Link:       "https://y.qq.com/n/ryqq/songDetail/" + songmid,
		MusicURL:   musicURL,
		Lyric:      cleanLyric(rawLyric),
		LyricLRC:   rawLyric,
		DurationMS: durationMS,
		Note:       note,
	}, nil
}

// vkey 获取播放地址。
//
// uin 必须与 Cookie 中的 uin 一致，否则接口只会返回空 purl。
func (QQMusic) vkey(ctx context.Context, songmid, cookie string) string {
	guid := strconv.FormatInt(int64(rand.Int63n(8999999999)+1000000000), 10)

	uin := cookieValue(cookie, "uin")
	if uin == "" {
		uin = "0"
	}

	payload := map[string]interface{}{
		"comm": map[string]interface{}{
			"uin":    uin,
			"format": "json",
			"ct":     24,
			"cv":     0,
		},
		"req_0": map[string]interface{}{
			"module": "vkey.GetVkeyServer",
			"method": "CgiGetVkey",
			"param": map[string]interface{}{
				"guid":      guid,
				"songmid":   []string{songmid},
				"songtype":  []int{0},
				"uin":       uin,
				"loginflag": 1,
				"platform":  "20",
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	reqURL := "https://u.y.qq.com/cgi-bin/musicu.fcg?data=" + queryEscape(string(raw))
	data, err := getJSON(ctx, reqURL, cookie, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return ""
	}

	infoList := arr(dig(data, "req_0", "data", "midurlinfo"))
	if len(infoList) == 0 {
		return ""
	}
	purl := str(dig(infoList[0], "purl"))
	if purl == "" {
		return ""
	}

	sipList := arr(dig(data, "req_0", "data", "sip"))
	sip := "https://ws.stream.qqmusic.qq.com/"
	if len(sipList) > 0 {
		if v := str(sipList[0]); v != "" {
			sip = v
		}
	}
	return sip + purl
}

// lyric 获取歌词。
func (QQMusic) lyric(ctx context.Context, songmid, cookie string) string {
	reqURL := fmt.Sprintf(
		"https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?songmid=%s&format=json&nobase64=1",
		queryEscape(songmid))
	data, err := getJSON(ctx, reqURL, cookie, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return ""
	}
	return str(data["lyric"])
}
