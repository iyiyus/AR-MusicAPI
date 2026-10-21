package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/music"
	"ar-musicapi/internal/store"
)

// musicAPIResponse 对外点歌接口保持与旧版 PHP 一致的扁平结构，便于机器人直接对接。
type musicAPIResponse struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg,omitempty"`
	Source     string `json:"source,omitempty"`
	SongID     string `json:"song_id,omitempty"`
	Title      string `json:"title,omitempty"`
	Singer     string `json:"singer,omitempty"`
	Cover      string `json:"cover,omitempty"`
	Link       string `json:"link,omitempty"`
	MusicURL   string `json:"music_url,omitempty"`
	Lyric      string `json:"lyric,omitempty"`
	// LyricLRC 带时间轴的原始歌词，可用于前端逐行滚动。
	LyricLRC string `json:"lyric_lrc,omitempty"`
	// DurationMS 本次请求耗时（毫秒）。
	DurationMS int64 `json:"duration_ms,omitempty"`
	// SongDurationMS 歌曲时长（毫秒）。
	SongDurationMS int64 `json:"song_duration_ms,omitempty"`
}

// handleMusicSources 返回音乐源列表（公开）。
func (s *Server) handleMusicSources(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, s.registry.Sources())
}

// handleMusicSearch 统一的点歌接口。
//
//	GET /api/music/search?name=歌曲名&source=qqmusic
//
// 返回扁平 JSON，兼容旧版 PHP 接口。
func (s *Server) handleMusicSearch(w http.ResponseWriter, r *http.Request) {
	name := httpx.QueryStr(r, "name")
	if name == "" {
		name = httpx.QueryStr(r, "keyword")
	}
	if name == "" {
		name = httpx.QueryStr(r, "s")
	}

	// 汽水音乐等源不支持按歌名搜索，可传歌曲 ID 或分享链接。
	rawID := httpx.QueryStr(r, "id")
	if rawID == "" {
		rawID = httpx.QueryStr(r, "url")
	}
	trackID := music.ExtractTrackID(rawID)

	if name == "" && trackID == "" {
		writeMusicJSON(w, musicAPIResponse{Code: httpx.CodeBadRequest, Msg: "请输入歌曲名"})
		return
	}

	// 指定歌手时优先命中原唱，避免搜到翻唱。
	singer := httpx.QueryStr(r, "singer")
	if singer == "" {
		singer = httpx.QueryStr(r, "artist")
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	if !s.publicAPIAvailable(r) {
		writeMusicJSON(w, musicAPIResponse{Code: httpx.CodeForbidden, Msg: "该接口未开放"})
		return
	}

	source := httpx.QueryStr(r, "source")
	if source == "" {
		source, _ = s.store.GetSetting(ctx, "default_source")
	}
	if source == "" {
		source = "qqmusic"
	}

	searcher, ok := s.registry.Get(source)
	if !ok {
		writeMusicJSON(w, musicAPIResponse{Code: httpx.CodeBadRequest, Msg: "不支持的音乐源: " + source})
		return
	}

	started := time.Now()

	var (
		result *music.Result
		err    error
	)
	if trackID != "" {
		detailer, ok := searcher.(music.Detailer)
		if !ok {
			writeMusicJSON(w, musicAPIResponse{
				Code:   httpx.CodeBadRequest,
				Msg:    "该音乐源不支持按 ID 解析",
				Source: source,
			})
			return
		}
		result, err = s.runDetail(ctx, source, detailer, trackID)
	} else {
		result, err = s.runSearch(ctx, searcher, name, singer)
	}

	duration := time.Since(started).Milliseconds()
	logName := name
	if logName == "" {
		logName = trackID
	}
	s.writeMusicLog(r, source, logName, result, err, duration)

	if err != nil {
		code := httpx.CodeServerError
		switch {
		case errors.Is(err, music.ErrNotFound):
			code = httpx.CodeNotFound
		case errors.Is(err, music.ErrNoSearch):
			code = httpx.CodeBadRequest
		}
		writeMusicJSON(w, musicAPIResponse{Code: code, Msg: err.Error(), Source: source})
		return
	}

	msg := "success"
	switch {
	case result.Note != "":
		msg = result.Note
	case result.MusicURL == "":
		msg = "未获取到播放直链，请在 Cookie 池添加该音乐源的登录 Cookie"
	}

	writeMusicJSON(w, musicAPIResponse{
		Code:       httpx.CodeSuccess,
		Msg:        msg,
		Source:     result.Source,
		SongID:     result.SongID,
		Title:      result.Title,
		Singer:     result.Singer,
		Cover:      result.Cover,
		Link:       result.Link,
		MusicURL:       result.MusicURL,
		Lyric:          result.Lyric,
		LyricLRC:       result.LyricLRC,
		DurationMS:     duration,
		SongDurationMS: result.DurationMS,
	})
}

// runSearch 自动挑选 Cookie 并执行搜索。
func (s *Server) runSearch(ctx context.Context, searcher music.Searcher, name, singer string) (*music.Result, error) {
	threshold := 3
	if raw, err := s.store.GetSetting(ctx, "cookie_fail_threshold"); err == nil && raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			threshold = n
		}
	}

	cookie, cookieErr := s.store.PickCookie(ctx, searcher.Source(), threshold)
	if cookieErr != nil {
		cookie = ""
		if searcher.NeedCookie() {
			return nil, music.ErrNoCookie
		}
	}

	result, err := searcher.Search(ctx, name, singer, cookie)
	if cookie != "" {
		_ = s.store.ReportCookie(ctx, searcher.Source(), cookie, err == nil)
	}
	return result, err
}

// runDetail 按歌曲 ID 解析（汽水音乐等无搜索接口的源）。
func (s *Server) runDetail(ctx context.Context, source string, detailer music.Detailer, trackID string) (*music.Result, error) {
	threshold := 3
	if raw, err := s.store.GetSetting(ctx, "cookie_fail_threshold"); err == nil && raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			threshold = n
		}
	}

	cookie, _ := s.store.PickCookie(ctx, source, threshold)

	result, err := detailer.Detail(ctx, trackID, cookie)
	if cookie != "" {
		_ = s.store.ReportCookie(ctx, source, cookie, err == nil)
	}
	return result, err
}

// publicAPIAvailable 校验是否允许匿名调用点歌接口。
func (s *Server) publicAPIAvailable(r *http.Request) bool {
	// 已登录用户始终可用
	if token := r.Header.Get("Authorization"); token != "" {
		if _, err := s.jwt.Parse(token); err == nil {
			return true
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	raw, err := s.store.GetSetting(ctx, "allow_public_api")
	if err != nil || raw == "" {
		return true
	}
	return raw != "0"
}

// writeMusicLog 记录一次调用（失败不影响主流程）。
func (s *Server) writeMusicLog(r *http.Request, source, name string, result *music.Result, searchErr error, durationMS int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry := &store.Log{
		Source:     source,
		SongName:   name,
		DurationMS: int(durationMS),
		IP:         httpx.ClientIP(r),
		UserAgent:  r.UserAgent(),
	}

	params, _ := json.Marshal(map[string]string{"name": name, "source": source})
	entry.RequestParams = string(params)

	if searchErr != nil {
		entry.Success = false
		entry.ResponseCode = httpx.CodeServerError
		if errors.Is(searchErr, music.ErrNotFound) {
			entry.ResponseCode = httpx.CodeNotFound
		}
		entry.ErrorMsg = searchErr.Error()
	} else {
		entry.Success = true
		entry.ResponseCode = httpx.CodeSuccess
		entry.AlbumImg = result.Cover
		entry.MusicURL = result.MusicURL
		entry.LyricLRC = result.LyricLRC
		entry.Singer = result.Singer
		entry.SongID = result.SongID
		if result.Title != "" {
			entry.SongName = result.Title
		}
	}

	_ = s.store.AddLog(ctx, entry)
}

func writeMusicJSON(w http.ResponseWriter, resp musicAPIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
