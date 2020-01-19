package utils

import (
	"github.com/savaki/geoip2"
)

type IpDetail struct {
	Ip         string  `bson:"ip_addr" json:"ip_addr"`
	LocationCN string  `bson:"location_cn" json:"location_cn"`
	LocationEN string  `bson:"location_en" json:"location_en"`
	Longitude  float64 `bson:"longitude" json:"longitude"`
	Latitude   float64 `bson:"latitude" json:"latitude"`
}

func GetIpDetails(geoIpUserId, geoIpKey, ip string) (res *IpDetail, err error) {
	if len(ip) < 1 && IsLanIp(ip) {
		return nil, nil
	}
	api := geoip2.New(geoIpUserId, geoIpKey)
	resp, err := api.City(nil, ip)
	if err != nil {
		return
	}
	detail := new(IpDetail)
	continentCN := ""
	countryCN := ""
	provinceCN := ""
	cityCN := ""
	continentEN := ""
	countryEN := ""
	provinceEN := ""
	cityEN := ""

	if resp.Continent.Names != nil {
		if c, ok := resp.Continent.Names["zh-CN"]; ok {
			continentCN = c
		}
		if cen, ok := resp.Continent.Names["en"]; ok {
			continentEN = cen
		}

	}
	if resp.Country.Names != nil {
		if c, ok := resp.Country.Names["zh-CN"]; ok {
			countryCN = c
		}
		if cen, ok := resp.Country.Names["en"]; ok {
			countryEN = cen
		}

	}
	if resp.Subdivisions != nil && len(resp.Subdivisions) > 0 {
		if p, ok := resp.Subdivisions[0].Names["zh-CN"]; ok {
			provinceCN = p
		}
		if pen, ok := resp.Subdivisions[0].Names["en"]; ok {
			provinceEN = pen
		}

	}
	if resp.City.Names != nil {
		if c, ok := resp.City.Names["zh-CN"]; ok {
			cityCN = c
		}
		if cen, ok := resp.City.Names["en"]; ok {
			cityEN = cen
		}

	}
	detail.LocationCN = continentCN + " " + countryCN + " " + provinceCN + " " + cityCN
	detail.LocationEN = continentEN + " " + countryEN + " " + provinceEN + " " + cityEN
	detail.Ip = ip
	detail.Latitude = resp.Location.Latitude
	detail.Longitude = resp.Location.Longitude

	return detail, nil
}
func IsLanIp(ip string) bool {
	if ip[0:3] == "10." || ip[0:4] == "172." || ip[0:4] == "192." {
		return true
	}
	return false
}
