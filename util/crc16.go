package util

// crc16tab will be filled automatically at startup.
var crc16tab []uint16

// Initialize the CRC16 lookup table (Modbus polynomial 0x1021)
func init() {
	crc16tab = make([]uint16, 256)
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
		crc16tab[i] = crc
	}
}

// CRC16 computes a CRC16 checksum using the Modbus polynomial (0x1021).
// This is identical to the algorithm used by Redis Cluster.
func CRC16(data []byte) uint16 {
	var crc uint16 = 0
	for _, b := range data {
		idx := byte((crc>>8)^uint16(b)) & 0xFF
		crc = (crc << 8) ^ crc16tab[idx]
	}
	return crc
}
