package wechatmp

import (
	"encoding/json"

	"github.com/herb-go/fetch"
)

const ScopeSnsapiBase = "snsapi_base"
const ScopeSnsapiUserinfo = "snsapi_userinfo"

var Server = fetch.Server{
	Host: "https://api.weixin.qq.com",
}

var apiGetUserInfo = Server.EndPoint("GET", "/sns/userinfo")
var apiToken = Server.EndPoint("GET", "/cgi-bin/token")
var apiOauth2AccessToken = Server.EndPoint("GET", "/sns/oauth2/access_token")

const ApiErrAccessTokenWrong = 40014
const ApiErrAccessTokenOutOfDate = 42001
const ApiErrSuccess = 0
const ApiErrUserUnaccessible = 50002
const ApiErrOauthCodeWrong = 40029

type resultAPIError struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type resultAccessToken struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type resultOauthToken struct {
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	scope        string `json:"scope"`
	Unionid      string `json:"unionid"`
}
type resultUserDetail struct {
	Errcode    int             `json:"errcode"`
	Errmsg     string          `json:"errmsg"`
	OpenID     string          `json:"openid"`
	Nickname   string          `json:"nickname"`
	Sex        string          `json:"sex"`
	Province   string          `json:"province"`
	City       string          `json:"city"`
	Country    string          `json:"country"`
	HeadimgURL string          `json:"headimgurl"`
	Privilege  json.RawMessage `json:"privilege"`
	Unionid    string          `json:"unionid"`
}