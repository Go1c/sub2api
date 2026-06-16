// Package geoip 提供基于内置 MaxMind GeoLite2-Country 数据库的 IP 国家归属查询。
//
// 数据库通过 go:embed 打进二进制，自包含、不依赖宿主 volume；
// 用于合规场景下按地区（如中国大陆 CN）拦截网页访问。
package geoip

import (
	_ "embed"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// embeddedDB 是内置的 GeoLite2-Country 数据库。
// 数据来源：MaxMind GeoLite2（更新方式见 doc/compliance-geo-block.md）。
//
//go:embed data/GeoLite2-Country.mmdb
var embeddedDB []byte

var (
	reader     *geoip2.Reader
	readerErr  error
	readerOnce sync.Once
)

// getReader 懒加载并复用 geoip2.Reader（内部线程安全，可并发查询）。
func getReader() (*geoip2.Reader, error) {
	readerOnce.Do(func() {
		reader, readerErr = geoip2.FromBytes(embeddedDB)
	})
	return reader, readerErr
}

// LookupCountryISO 返回给定 IP 的 ISO 3166-1 alpha-2 国家码（如 "CN"、"US"）。
//
// 第二个返回值表示是否成功解析出国家：当 IP 非法、为私有/保留地址、
// 数据库中无记录或数据库加载失败时返回 ("", false)，调用方应据此 fail-open（放行）。
func LookupCountryISO(ipStr string) (string, bool) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", false
	}

	r, err := getReader()
	if err != nil || r == nil {
		return "", false
	}

	record, err := r.Country(ip)
	if err != nil || record == nil {
		return "", false
	}

	code := record.Country.IsoCode
	if code == "" {
		return "", false
	}
	return code, true
}
