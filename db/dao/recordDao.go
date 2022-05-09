package dao

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
)

type TracertRecordData struct {
	Server *gorm.DB

	sync.RWMutex
}

var (
	GlobalTracertRecordData = TracertRecordData{}
)

func (trd *TracertRecordData) GetTracertRecordByDomain(dst string) ([]TracertRecord, error) {
	var ret []TracertRecord
	conn, _ := GetConn()
	if conn == nil {
		return ret, fmt.Errorf("can not connect tracert")
	}
	dt := conn.Table(TracertRecordDB.TableName()).
		Where("tracert_record.dst in (?)", dst).
		Scan(&ret)

	if dt.Error != nil {
		logrus.Errorf("Error! GetTracertRecordByDomain failed. [%v]", dt.Error)
	}
	return ret, dt.Error
}

func (trd *TracertRecordData) InsertTracertRecord(tr *TracertRecord) (int, error) {
	conn, _ := GetConn()
	if conn == nil {
		return -1, fmt.Errorf("can not connect tracert")
	}

	// 开启事务
	tx := conn.Begin()
	dt := conn.Create(tr)
	if dt.Error != nil {
		logrus.Errorf("Error! Insert into TracertRecord failed. [%v]", dt.Error)
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
