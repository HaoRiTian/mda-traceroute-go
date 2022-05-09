package dao

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"

	// gorm 要求导入的驱动
	_ "github.com/go-sql-driver/mysql"
)

/*
	CREATE TABLE `topo` (
		`id` int(11) NOT NULL AUTO_INCREMENT,
		`domain` varchar(32) DEFAULT NULL,
		`ttl` int(11) DEFAULT NULL,
		`dst_ip` varchar(128),
		`res_addr` varchar(128),
		`name` varchar(128),
		`session` varchar(128),
		`mean_latency` double,
		`recv_cnt` int(11),
        `tracert_time` datetime,
		`insert_time` datetime DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (`id`)
	) ENGINE=InnoDB
*/

type Topo struct {
	Id          int       `json:"id" gorm:"column:id"`
	Domain      string    `json:"domain" gorm:"column:domain"`
	TTL         uint8     `json:"ttl" gorm:"column:ttl"`
	DstIP       string    `json:"dst_ip" gorm:"column:dst_ip"`
	ResAddr     string    `json:"res_addr" gorm:"column:res_addr"`
	Name        string    `json:"name" gorm:"column:name"`
	Session     string    `json:"session" gorm:"column:session"`
	MeanLatency float64   `json:"mean_latency" gorm:"column:mean_latency"`
	RecvCnt     uint64    `json:"recv_cnt" gorm:"column:recv_cnt"`
	Country     string    `json:"country" gorm:"column:country"`
	Region      string    `json:"region" gorm:"column:region"`
	City        string    `json:"city" gorm:"column:city"`
	ISP         string    `json:"isp" gorm:"column:isp"`
	TracertTime time.Time `json:"tracert_time" gorm:"column:tracert_time;type:datetime"`
	InsertTime  time.Time `json:"insert_time" gorm:"autoCreateTime;column:insert_time;type:datetime"`
}

/*
CREATE TABLE `tracert_record` (
  `id` int NOT NULL AUTO_INCREMENT,
  `dst` varchar(255) DEFAULT NULL,
  `group` varchar(255) DEFAULT NULL,
  `node_num` int DEFAULT NULL,
  `tracert_time` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_dst` (`dst`),
  KEY `idx_rt_time` (`tracert_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
*/

type TracertRecord struct {
	Id          int       `json:"id" gorm:"column:id"`
	Dst         string    `json:"dst" gorm:"column:dst"`
	Group       string    `json:"group" gorm:"column:group""`
	NodeNum     int32     `json:"node-num" gorm:"column:node_num"`
	TracertTime time.Time `json:"tracert-time" gorm:"column:tracert_time;type:datetime"`
}

var TopoDB = Topo{}

var TracertRecordDB = TracertRecord{}

// GetConn 获取数据库连接
func GetConn() (*gorm.DB, error) {
	var p *sql.DB
	portal, err := gorm.Open("mysql", "root:rootymzh2022@tcp(127.0.0.1:3306)/tracert?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		fmt.Errorf("connect to tracert db: %s", err.Error())
	}
	portal.Dialect().SetDB(p)
	portal.SingularTable(true)
	return portal, err
}

func (t *Topo) TableName() string {
	return "topo"
}

func (t *TracertRecord) TableName() string {
	return "tracert_record"
}
