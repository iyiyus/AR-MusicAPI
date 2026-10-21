package music

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// KuGou 酷狗音乐。
//
// 酷狗对匿名请求普遍限权（privilege=10），免费歌曲也需要登录 Cookie。
// 因此这里同时请求「移动端信息接口」与「PC 播放接口」，任一返回直链即可用。
type KuGou struct{}

// Source 实现 Searcher。
func (KuGou) Source() string { return "kugou" }

// Name 实现 Searcher。
func (KuGou) Name() string { return "酷狗音乐" }

// NeedCookie 实现 Searcher。匿名可用但大概率拿不到直链。
func (KuGou) NeedCookie() bool { return false }

// Search 实现 Searcher。
func (s KuGou) Search(ctx context.Context, keyword, singer, cookie string) (*Result, error) {
	cookie = withKgMid(cookie, randomHex())

	searchURL := "https://songsearch.kugou.com/song_search_v2?keyword=" + queryEscape(keyword) +
		"&page=1&pagesize=20&userid=-1&clientver=&platform=WebFilter&tag=em" +
		"&filter=2&iscorrection=1&privilege_filter=0"

	data, err := getJSON(ctx, searchURL, cookie, map[string]string{"Referer": "https://www.kugou.com/"})
	if err != nil {
		return nil, err
	}

	lists := arr(dig(data, "data", "lists"))
	if len(lists) == 0 {
		return nil, ErrNotFound
	}
	song := pickBestSong(lists, keyword, singer, func(m map[string]interface{}) string {
		return stripHTML(str(m["SongName"]))
	}, func(m map[string]interface{}) string {
		return str(m["SingerName"])
	})

	hash := str(song["FileHash"])
	albumID := str(song["AlbumID"])

	info := s.songInfo(ctx, hash, cookie)
	play := s.playData(ctx, hash, albumID, cookie)

	title := stripHTML(str(info["songName"]))
	if title == "" {
		title = stripHTML(str(song["SongName"]))
	}
	singers := str(info["author_name"])
	if singers == "" {
		singers = str(song["SingerName"])
	}

	musicURL := str(info["url"])
	if musicURL == "" {
		musicURL = str(play["play_url"])
	}

	cover := strings.ReplaceAll(str(info["album_img"]), "{size}", "500")
	if cover == "" {
		cover = str(play["img"])
	}
	if cover == "" && albumID != "" {
		cover = "https://imge.kugou.com/stdmusic/500/" + albumID + ".jpg"
	}

	var durationMS int64
	if v, ok := info["timeLength"].(float64); ok && v > 0 {
		durationMS = int64(v)
	} else if v, ok := song["Duration"].(float64); ok {
		durationMS = int64(v) * 1000
	}

	rawLyric := str(play["lyrics"])
	if cleanLyric(rawLyric) == "" {
		rawLyric = s.lyric(ctx, hash, title, durationMS, cookie)
	}

	note := ""
	if musicURL == "" {
		note = "酷狗提示：该歌曲需要会员 Cookie，请先在 Cookie 池添加酷狗登录 Cookie"
	}

	return &Result{
		Source:     s.Source(),
		SongID:     hash,
		Title:      title,
		Singer:     singers,
		Cover:      cover,
		Link:       "https://www.kugou.com/song/#hash=" + hash + "&album_id=" + albumID,
		MusicURL:   musicURL,
		Lyric:      cleanLyric(rawLyric),
		LyricLRC:   rawLyric,
		DurationMS: durationMS,
		Note:       note,
	}, nil
}

// songInfo 移动端歌曲信息接口（含直链、封面、时长、限权状态）。
func (KuGou) songInfo(ctx context.Context, hash, cookie string) map[string]interface{} {
	reqURL := "https://m.kugou.com/app/i/getSongInfo.php?hash=" + queryEscape(hash) + "&cmd=playInfo"
	data, err := getJSON(ctx, reqURL, cookie, map[string]string{
		"Referer":    "https://m.kugou.com/",
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15",
	})
	if err != nil {
		return map[string]interface{}{}
	}
	return data
}

// playData PC 端播放接口（含歌词）。
func (KuGou) playData(ctx context.Context, hash, albumID, cookie string) map[string]interface{} {
	reqURL := fmt.Sprintf(
		"https://wwwapi.kugou.com/yy/index.php?r=play/getdata&hash=%s&album_id=%s&platid=4&dfid=-&appid=1014&_=%d",
		queryEscape(hash), queryEscape(albumID), time.Now().UnixMilli())

	data, err := getJSON(ctx, reqURL, cookie, map[string]string{"Referer": "https://www.kugou.com/"})
	if err != nil {
		return map[string]interface{}{}
	}
	return obj(data["data"])
}

// lyric 从酷狗歌词库获取歌词（返回 base64 编码的 LRC）。
func (KuGou) lyric(ctx context.Context, hash, title string, durationMS int64, cookie string) string {
	searchURL := fmt.Sprintf(
		"https://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=%s&duration=%d&hash=%s",
		queryEscape(title), durationMS, queryEscape(hash))

	data, err := getJSON(ctx, searchURL, cookie, nil)
	if err != nil {
		return ""
	}
	candidates := arr(data["candidates"])
	if len(candidates) == 0 {
		return ""
	}

	item := obj(candidates[0])
	downURL := fmt.Sprintf(
		"https://lyrics.kugou.com/download?ver=1&client=pc&id=%s&accesskey=%s&fmt=lrc&charset=utf8",
		str(item["id"]), queryEscape(str(item["accesskey"])))

	downData, err := getJSON(ctx, downURL, cookie, nil)
	if err != nil {
		return ""
	}
	content := str(downData["content"])
	if content == "" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return ""
	}
	return string(decoded)
}

// withKgMid 追加 kg_mid，规避酷狗错误码 20010。
func withKgMid(cookie, mid string) string {
	cookie = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(cookie), ";"))
	if cookie == "" {
		return "kg_mid=" + mid
	}
	if strings.Contains(cookie, "kg_mid=") {
		return cookie
	}
	return cookie + "; kg_mid=" + mid
}

// randomHex 生成 32 位随机串作设备标识。
func randomHex() string {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixNano(), strconv.IntSize)
	sum := md5.Sum([]byte(seed))
	return fmt.Sprintf("%x", sum)
}
