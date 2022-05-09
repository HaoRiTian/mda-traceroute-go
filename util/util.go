package util

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/BurntSushi/toml"
)

func If(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

// FormatNow 格式化（2006-01-02 15:04:05）输出当前时间
func FormatNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func SortInsertUint8(s *[]uint8, e uint8) {
	ss := *s
	i := sort.Search(len(ss), func(i int) bool { return ss[i] >= e })
	ss = append(ss, uint8(0))
	copy(ss[i+1:], ss[i:])
	ss[i] = e
	*s = ss
}

func SortInsertUint32(s *[]uint32, e uint32) {
	//	x := 23
	//	i := sort.Search(len(data), func(i int) bool { return data[i] >= x })
	//	if i < len(data) && data[i] == x {
	//		// x is present at data[i]
	//	} else {
	//		// x is not present in data,
	//		// but i is the index where it would be inserted.
	//	}
	ss := *s
	i := sort.Search(len(ss), func(i int) bool { return ss[i] >= e })
	ss = append(ss, uint32(0))
	copy(ss[i+1:], ss[i:])
	ss[i] = e
	*s = ss
}

func FileOrPathIsExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func GetToday() string {
	date := time.Now().Format("20060102")
	return fmt.Sprintf("%s", date)
}

func ParseConfigToml(conf string, c interface{}) error {
	if conf == "" {
		return fmt.Errorf("need config file")
	}

	if !FileOrPathIsExists(conf) {
		return fmt.Errorf("config " + conf + " is not exist")
	}

	_, err := toml.DecodeFile(conf, c)
	if err != nil {
		return err
	}

	return nil
}

// MatchDst 目的地址为域名则返回0，为IP则返回1，都不是则返回-1
func MatchDst(dst string) int8 {
	domain := regexp.MustCompile(`^((http://)|(https://))?[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+.?`)
	ip := regexp.MustCompile(`((?:(?:25[0-5]|2[0-4]d|[01]?d?d).){3}(?:25[0-5]|2[0-4]d|[01]?d?d))`)
	if domain.MatchString(dst) {
		return 0
	} else if ip.MatchString(dst) {
		return 1
	}
	return -1
}
