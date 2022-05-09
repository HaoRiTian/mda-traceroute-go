package utils

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
)

func GetHash(src net.IP, dst net.IP, srcPort uint16, dstPort uint16, proto uint16) string {
	h := sha1.New()
	h.Write(src)
	h.Write(dst)
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, srcPort)
	h.Write(p)
	binary.BigEndian.PutUint16(p, dstPort)
	h.Write(p)
	binary.BigEndian.PutUint16(p, proto)
	h.Write(p)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CheckSum 计算校验和
func CheckSum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}
