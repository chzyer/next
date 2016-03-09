// Packet Layout
//   --------------------------------------------------------------
//   iv +
//   crc32(type+payload) + aes(crc32(type+payload), token, iv) +
//   length + aes(payload + type, token, iv)
//   --------------------------------------------------------------
//   iv => userId(uint16) + portNum(uint16) + reqId(int32) + rand(8) = 16byte
//   length => int16
//   aes => aes-256-cfb
//   type => int8
//   payload => []byte
//   token => auth request
package packet
