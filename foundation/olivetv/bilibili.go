package olivetv

import (
	"fmt"
	"time"

	"github.com/go-olive/olive/foundation/olivetv/model"
	"github.com/go-olive/olive/foundation/olivetv/util"
)

func init() {
	registerSite("bilibili", &bilibili{})
}

type bilibili struct {
	base
}

func (this *bilibili) Name() string {
	return "哔哩哔哩"
}

func (this *bilibili) Snap(tv *Tv) error {
	tv.Info = &Info{
		Timestamp: time.Now().Unix(),
	}

	options := []Option{
		this.setRoomOn(),
		this.setStreamURL(),
	}

	for _, option := range options {
		if err := option(tv); err != nil {
			return err
		}
	}

	return nil
}

func (this *bilibili) setRoomOn() Option {
	type roomInit struct {
		Code int64 `json:"code"`
		Data struct {
			RoomID     int64 `json:"room_id"`
			LiveStatus int64 `json:"live_status"`
			UID        int64 `json:"uid"`
		}
	}

	return func(tv *Tv) error {
		roomInit := new(roomInit)
		req := &util.HttpRequest{
			// https://github.com/SocialSisterYi/bilibili-API-collect/blob/master/live/info.md#获取房间页初始化信息
			URL:    "https://api.live.bilibili.com/room/v1/Room/room_init",
			Method: "POST",
			RequestData: map[string]interface{}{
				"id": tv.RoomID,
			},
			ResponseData: roomInit,
			ContentType:  "application/form-data",
		}
		if err := req.Send(); err != nil {
			return err
		}
		if roomInit.Code != 0 || roomInit.Data.LiveStatus != 1 {
			return nil
		}

		tv.RoomID = fmt.Sprint(roomInit.Data.RoomID)
		tv.roomOn = true

		titleInfo := new(model.BilibiliRoomTitle)
		req = &util.HttpRequest{
			URL:          fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByRoom?room_id=%s", tv.RoomID),
			Method:       "GET",
			ResponseData: titleInfo,
			ContentType:  "application/json",
		}
		if err := req.Send(); err != nil {
			return nil
		}

		tv.roomName = titleInfo.Data.RoomInfo.Title
		return nil
	}
}

func (this *bilibili) setStreamURL() Option {
	return this.getRealURL
}

func (this *bilibili) getAutoGenerated(roomID string, currentQn int) (*model.BilibiliAutoGenerated, error) {
	auto := new(model.BilibiliAutoGenerated)
	req := &util.HttpRequest{
		// https://github.com/SocialSisterYi/bilibili-API-collect/blob/master/live/live_stream.md
		URL:    "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo",
		Method: "GET",
		RequestData: map[string]interface{}{
			"room_id":  roomID,
			"protocol": "0,1",
			"format":   "0,1,2",
			"codec":    "0,1",
			"qn":       currentQn,
			"platform": "h5",
			"ptype":    "8",
		},
		ResponseData: auto,
		ContentType:  "application/form-data",
	}
	err := req.Send()
	return auto, err
}

func (this *bilibili) getRealURL(tv *Tv) error {
	if !tv.roomOn {
		return nil
	}

	// 原画画质
	const highestQn = 10000
	auto, err := this.getAutoGenerated(tv.RoomID, highestQn)
	if err != nil {
		return err
	}
	for _, stream := range auto.Data.PlayurlInfo.Playurl.Stream {
		if stream.ProtocolName != "http_stream" {
			continue
		}
		for _, format := range stream.Format {
			if format.FormatName != "flv" {
				continue
			}
			var qnMax int
			if len(format.Codec) <= 0 {
				continue
			}
			for _, qn := range format.Codec[0].AcceptQn {
				if qn > qnMax {
					qnMax = qn
				}
			}
			if format.Codec[0].CurrentQn != qnMax && qnMax > 0 {
				auto, err = this.getAutoGenerated(tv.RoomID, qnMax)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, stream := range auto.Data.PlayurlInfo.Playurl.Stream {
		if stream.ProtocolName != "http_stream" {
			continue
		}
		for _, format := range stream.Format {
			if format.FormatName != "flv" {
				continue
			}
			if len(format.Codec) <= 0 {
				continue
			}
			baseURL := format.Codec[0].BaseURL
			urlInfo := format.Codec[0].URLInfo[0]
			streamURL := urlInfo.Host + baseURL + urlInfo.Extra
			tv.streamUrl = streamURL
			return nil
		}
	}
	if tv.streamUrl == "" {
		tv.roomOn = false
	}
	return fmt.Errorf("fail to getRealURL")
}
