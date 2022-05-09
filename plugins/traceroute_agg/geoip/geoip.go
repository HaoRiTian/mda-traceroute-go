package geoip

import (
	"bytes"
	"encoding/json"
	"fmt"
	geoip2 "github.com/oschwald/geoip2-golang"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"net"
	"net/http"
)

var GlobalGeoIP *GeoIPDB

func InitGeoipDB(cityDBPrefix string, asnDBPrefix string) {
	geoip, err := NewGeoIP(cityDBPrefix, asnDBPrefix)
	if err != nil {
		logrus.Errorf("%v", err)
	}

	GlobalGeoIP = geoip
	logrus.Infof("GlobalGeoIP init complete.")
}

// NewGeoIP New is used to create database
func NewGeoIP(cityDBPrefix string, asnDBPrefix string) (*GeoIPDB, error) {
	var r = GeoIPDB{}
	var err error
	r.CityDB, err = geoip2.Open(cityDBPrefix)
	r.ASNDB, err = geoip2.Open(asnDBPrefix)
	if err != nil {
		logrus.Fatal(err)
	}
	return &r, err
}

func ComputeDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R float64 = 6371
	Deg2Rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * Deg2Rad
	dLon := (lon2 - lon1) * Deg2Rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*Deg2Rad)*math.Cos(lat2*Deg2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * R * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

//GeoIPDB is the main struct of ip lookup engine
type GeoIPDB struct {
	CityDB *geoip2.Reader
	ASNDB  *geoip2.Reader
}

//GeoLocation is the response type for location lookup
type GeoLocation struct {
	City      string
	Region    string
	Country   string
	ASN       uint
	SPName    string
	Latitude  float64
	Longitude float64
}

func (g GeoLocation) String() string {
	return fmt.Sprintf("City:  %-15s|Region:  %-15s|Country: %-15s |ASN: %8d::%-40s| Lat: %4.3f,%4.3f", g.City, g.Region, g.Country, g.ASN, g.SPName, g.Latitude, g.Longitude)
}

//Lookup is used to find IP location in GeoIPDB
func (g *GeoIPDB) Lookup(ipAddr string) (GeoLocation, error) {
	var r GeoLocation
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return r, fmt.Errorf("ip is nil. ipAddr:[%s]", ipAddr)
	}
	c, err := g.CityDB.City(ip)
	if err != nil {
		return r, err
	}
	if asn, err := g.ASNDB.ASN(ip); err == nil {
		r.ASN = asn.AutonomousSystemNumber
		r.SPName = asn.AutonomousSystemOrganization
	}
	if c.City.GeoNameID != 0 {
		r.City = c.City.Names["en"]
	}
	if len(c.Subdivisions) > 0 {
		if c.Subdivisions[0].GeoNameID != 0 {
			r.Region = c.Subdivisions[0].Names["en"]
		}
	}
	if c.Country.GeoNameID != 0 {
		r.Country = c.Country.Names["en"]
	}

	// 赞！！！
	if r.Country == "Hong Kong" {
		r.Country = "China"
		r.Region = "Hong Kong"
		r.City = "Hong Kong"
	}
	if r.Country == "Macau" || r.Country == "Macao" {
		r.Country = "China"
		r.Region = "Macau"
		r.City = "Macau"
	}

	if r.Country == "Taiwan" {
		r.Country = "China"
		r.Region = "Taiwan"
	}

	r.Latitude = c.Location.Latitude
	r.Longitude = c.Location.Longitude

	return r, nil
}

// https://api.vore.top/api/IPdata?ip=43.130.227.102 东京地址查成了香港的。缺少一个可靠的接口

// LookupWithNet 使用网络资源查找IP地理位置
func (g *GeoIPDB) LookupWithNet(ipAddr string) GeoLocation {
	var r GeoLocation
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return r
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.vore.top/api/IPdata?ip="+ipAddr, bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
	}

	res, err := client.Do(req)
	if err != nil {
		logrus.Errorf("%v", err)
	}

	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	var iface interface{}
	err = json.Unmarshal(content, &iface)
	if err != nil {
		logrus.Errorf("%v", err)
		return r
	}

	//str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	//fmt.Println(*str)

	result, ok := iface.(map[string]interface{})
	if !ok {
		logrus.Errorf("%v", err)
		return r
	}

	ipDataIface, ok := result["ipdata"]
	if ok {
		ipData, ok := ipDataIface.(map[string]interface{})
		if ok {
			//r.Country = ipData[""]
			r.Region = ipData["info1"].(string)
			r.City = ipData["info2"].(string)
			r.SPName = ipData["isp"].(string)
		}
	}
	return r
}
