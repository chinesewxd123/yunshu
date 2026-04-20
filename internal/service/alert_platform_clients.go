package service

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"yunshu/internal/pkg/parseutil"
)

func (s *AlertService) getDingTalkAccessToken(ctx context.Context, appKey, appSecret string) (string, error) {
	u := "https://oapi.dingtalk.com/gettoken?appkey=" + neturl.QueryEscape(appKey) + "&appsecret=" + neturl.QueryEscape(appSecret)
	var body struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	cacheKey := "alert:ding:token:" + appKey
	return s.cachedToken(ctx, cacheKey, func(ctx context.Context) (string, int, error) {
		_, err := s.platformHTTPClient().DoJSON(ctx, "GET", u, nil, nil, &body)
		if err != nil {
			return "", 0, err
		}
		if body.ErrCode != 0 || strings.TrimSpace(body.AccessToken) == "" {
			return "", 0, fmt.Errorf("dingtalk gettoken failed: %s", body.ErrMsg)
		}
		return strings.TrimSpace(body.AccessToken), body.ExpiresIn, nil
	})
}

func (s *AlertService) resolveDingTalkUserIDsByMobiles(ctx context.Context, accessToken string, mobiles []string) ([]string, error) {
	out := make([]string, 0, len(mobiles))
	for _, m := range mobiles {
		uid, err := s.getDingTalkUserIDByMobile(ctx, accessToken, m)
		if err == nil && strings.TrimSpace(uid) != "" {
			out = append(out, strings.TrimSpace(uid))
		}
	}
	return parseutil.UniqueStrings(out), nil
}

func (s *AlertService) getDingTalkUserIDByMobile(ctx context.Context, accessToken, mobile string) (string, error) {
	u := "https://oapi.dingtalk.com/topapi/v2/user/getbymobile?access_token=" + neturl.QueryEscape(accessToken)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			UserID string `json:"userid"`
		} `json:"result"`
	}
	_, err := s.platformHTTPClient().DoJSON(ctx, "POST", u, nil, map[string]string{"mobile": strings.TrimSpace(mobile)}, &result)
	if err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("dingtalk get user by mobile failed: %s", result.ErrMsg)
	}
	return strings.TrimSpace(result.Result.UserID), nil
}

func (s *AlertService) resolveWeComUserIDsByMobiles(ctx context.Context, settings map[string]interface{}, mobiles []string) ([]string, error) {
	corpID := strings.TrimSpace(fmt.Sprintf("%v", settings["corpID"]))
	corpSecret := strings.TrimSpace(fmt.Sprintf("%v", settings["corpSecret"]))
	if corpID == "" || corpSecret == "" || len(mobiles) == 0 {
		return nil, nil
	}
	token, err := s.getWeComAccessToken(ctx, corpID, corpSecret)
	if err != nil || token == "" {
		return nil, err
	}
	out := make([]string, 0, len(mobiles))
	for _, m := range mobiles {
		uid, e := s.getWeComUserIDByMobile(ctx, token, m)
		if e == nil && strings.TrimSpace(uid) != "" {
			out = append(out, strings.TrimSpace(uid))
		}
	}
	return parseutil.UniqueStrings(out), nil
}

func (s *AlertService) getWeComAccessToken(ctx context.Context, corpID, corpSecret string) (string, error) {
	u := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + neturl.QueryEscape(corpID) + "&corpsecret=" + neturl.QueryEscape(corpSecret)
	var body struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	cacheKey := "alert:wcom:token:" + corpID
	return s.cachedToken(ctx, cacheKey, func(ctx context.Context) (string, int, error) {
		_, err := s.platformHTTPClient().DoJSON(ctx, "GET", u, nil, nil, &body)
		if err != nil {
			return "", 0, err
		}
		if body.ErrCode != 0 || strings.TrimSpace(body.AccessToken) == "" {
			return "", 0, fmt.Errorf("wechat gettoken failed: %s", body.ErrMsg)
		}
		return strings.TrimSpace(body.AccessToken), body.ExpiresIn, nil
	})
}

func (s *AlertService) getWeComUserIDByMobile(ctx context.Context, accessToken, mobile string) (string, error) {
	u := "https://qyapi.weixin.qq.com/cgi-bin/user/getuserid?access_token=" + neturl.QueryEscape(accessToken)
	var body struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		UserID  string `json:"userid"`
	}
	_, err := s.platformHTTPClient().DoJSON(ctx, "POST", u, nil, map[string]string{"mobile": strings.TrimSpace(mobile)}, &body)
	if err != nil {
		return "", err
	}
	if body.ErrCode != 0 {
		return "", fmt.Errorf("wechat getuserid failed: %s", body.ErrMsg)
	}
	return strings.TrimSpace(body.UserID), nil
}
