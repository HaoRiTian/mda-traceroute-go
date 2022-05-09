package dao

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
)

type TopoData struct {
	Server *gorm.DB

	sync.RWMutex
}

var (
	GlobalTopoData = TopoData{}
)

func (td *TopoData) GetTopoByDomain(domain string) ([]Topo, error) {
	var ret []Topo
	conn, _ := GetConn()
	if conn == nil {
		return ret, fmt.Errorf("can not connect tracert")
	}
	dt := conn.Table(TopoDB.TableName()).
		Select("topo.id, topo.domain, topo.ttl, topo.dst_ip, topo.res_addr, "+
			"topo.name, topo.session, topo.mean_latency, topo.recv_cnt, "+
			"topo.tracert_time, topo.insert_time").
		Where("topo.domain in (?)", domain).
		Scan(&ret)

	if dt.Error != nil {
		logrus.Errorf("Error! GetTopoByDomain failed. [%v]", dt.Error)
	}
	return ret, dt.Error
}

func (td *TopoData) GetTopoByDomainAndTTL(domain string, ttl int) ([]Topo, error) {
	var ret []Topo
	conn, _ := GetConn()
	if conn == nil {
		return ret, fmt.Errorf("can not connect tracert")
	}
	dt := conn.Table(TopoDB.TableName()).
		Where("topo.domain in (?) and topo.ttl in (?)", domain, ttl).
		Scan(&ret)

	if dt.Error != nil {
		logrus.Errorf("Error! GetTopoByDomainAndTTL failed. [%v]", dt.Error)
	}
	return ret, dt.Error
}

func (td *TopoData) InsertTopo(topo *Topo) (int, error) {
	conn, _ := GetConn()
	if conn == nil {
		return -1, fmt.Errorf("can not connect tracert")
	}

	// 开启事务
	tx := conn.Begin()
	dt := conn.Create(topo)
	if dt.Error != nil {
		logrus.Errorf("Error! Insert into Topo failed. [%v]", dt.Error)
	}
	// 获取刚插入记录的id
	var id []int
	conn.Raw("select LAST_INSERT_ID() as id").Pluck("id", &id)

	if dt.Error != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return id[0], dt.Error
}
