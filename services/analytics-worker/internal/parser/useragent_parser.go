package parser

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/mssola/useragent"
)

type DeviceInfo struct {
	DeviceType string
	Browser    string
	OS         string
}

func ParseUserAgent(uaString string) DeviceInfo {
	ua := useragent.New(uaString)

	deviceType := "desktop"
	if ua.Mobile() {
		deviceType = "mobile"
	} else if ua.Bot() {
		deviceType = "bot"
	}

	browserName, _ := ua.Browser()

	return DeviceInfo{
		DeviceType: deviceType,
		Browser:    browserName,
		OS:         ua.OS(),
	}
}

func HashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])[:16]
}
