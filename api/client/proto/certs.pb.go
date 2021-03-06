// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: certs.proto

package proto

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Set of certificates corresponding to a single public key.
type Certs struct {
	// SSH X509 cert (PEM-encoded).
	SSH []byte `protobuf:"bytes,1,opt,name=SSH,proto3" json:"ssh,omitempty"`
	// TLS X509 cert (PEM-encoded).
	TLS []byte `protobuf:"bytes,2,opt,name=TLS,proto3" json:"tls,omitempty"`
	// TLSCACerts is a list of TLS certificate authorities.
	TLSCACerts [][]byte `protobuf:"bytes,3,rep,name=TLSCACerts,proto3" json:"tls_ca_certs,omitempty"`
	// SSHCACerts is a list of SSH certificate authorities.
	SSHCACerts           [][]byte `protobuf:"bytes,4,rep,name=SSHCACerts,proto3" json:"ssh_ca_certs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Certs) Reset()         { *m = Certs{} }
func (m *Certs) String() string { return proto.CompactTextString(m) }
func (*Certs) ProtoMessage()    {}
func (*Certs) Descriptor() ([]byte, []int) {
	return fileDescriptor_78c43cca93027bbd, []int{0}
}
func (m *Certs) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Certs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Certs.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Certs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Certs.Merge(m, src)
}
func (m *Certs) XXX_Size() int {
	return m.Size()
}
func (m *Certs) XXX_DiscardUnknown() {
	xxx_messageInfo_Certs.DiscardUnknown(m)
}

var xxx_messageInfo_Certs proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Certs)(nil), "proto.Certs")
}

func init() { proto.RegisterFile("certs.proto", fileDescriptor_78c43cca93027bbd) }

var fileDescriptor_78c43cca93027bbd = []byte{
	// 192 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4e, 0x4e, 0x2d, 0x2a,
	0x29, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x52, 0x22, 0xe9, 0xf9, 0xe9,
	0xf9, 0x60, 0xa6, 0x3e, 0x88, 0x05, 0x91, 0x54, 0x3a, 0xc9, 0xc8, 0xc5, 0xea, 0x0c, 0x52, 0x2c,
	0xa4, 0xcc, 0xc5, 0x1c, 0x1c, 0xec, 0x21, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1, 0xe3, 0x24, 0xf8, 0xea,
	0x9e, 0x3c, 0x6f, 0x71, 0x71, 0x86, 0x4e, 0x7e, 0x6e, 0x66, 0x49, 0x6a, 0x6e, 0x41, 0x49, 0x65,
	0x10, 0x48, 0x16, 0xa4, 0x28, 0xc4, 0x27, 0x58, 0x82, 0x09, 0xa1, 0xa8, 0x24, 0xa7, 0x18, 0x59,
	0x51, 0x88, 0x4f, 0xb0, 0x90, 0x15, 0x17, 0x57, 0x88, 0x4f, 0xb0, 0xb3, 0x23, 0xd8, 0x5c, 0x09,
	0x66, 0x05, 0x66, 0x0d, 0x1e, 0x27, 0xa9, 0x57, 0xf7, 0xe4, 0xc5, 0x4a, 0x72, 0x8a, 0xe3, 0x93,
	0x13, 0xe3, 0xc1, 0x8e, 0x43, 0xd2, 0x84, 0xa4, 0x1a, 0xa4, 0x37, 0x38, 0xd8, 0x03, 0xa6, 0x97,
	0x05, 0xa1, 0xb7, 0xb8, 0x38, 0x03, 0xab, 0x5e, 0x84, 0x6a, 0x27, 0x81, 0x13, 0x0f, 0xe5, 0x18,
	0x4e, 0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x24, 0x36, 0xb0,
	0x27, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0xd5, 0x12, 0x91, 0xaf, 0x10, 0x01, 0x00, 0x00,
}

func (m *Certs) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Certs) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Certs) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.SSHCACerts) > 0 {
		for iNdEx := len(m.SSHCACerts) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.SSHCACerts[iNdEx])
			copy(dAtA[i:], m.SSHCACerts[iNdEx])
			i = encodeVarintCerts(dAtA, i, uint64(len(m.SSHCACerts[iNdEx])))
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.TLSCACerts) > 0 {
		for iNdEx := len(m.TLSCACerts) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.TLSCACerts[iNdEx])
			copy(dAtA[i:], m.TLSCACerts[iNdEx])
			i = encodeVarintCerts(dAtA, i, uint64(len(m.TLSCACerts[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.TLS) > 0 {
		i -= len(m.TLS)
		copy(dAtA[i:], m.TLS)
		i = encodeVarintCerts(dAtA, i, uint64(len(m.TLS)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.SSH) > 0 {
		i -= len(m.SSH)
		copy(dAtA[i:], m.SSH)
		i = encodeVarintCerts(dAtA, i, uint64(len(m.SSH)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintCerts(dAtA []byte, offset int, v uint64) int {
	offset -= sovCerts(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Certs) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SSH)
	if l > 0 {
		n += 1 + l + sovCerts(uint64(l))
	}
	l = len(m.TLS)
	if l > 0 {
		n += 1 + l + sovCerts(uint64(l))
	}
	if len(m.TLSCACerts) > 0 {
		for _, b := range m.TLSCACerts {
			l = len(b)
			n += 1 + l + sovCerts(uint64(l))
		}
	}
	if len(m.SSHCACerts) > 0 {
		for _, b := range m.SSHCACerts {
			l = len(b)
			n += 1 + l + sovCerts(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovCerts(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozCerts(x uint64) (n int) {
	return sovCerts(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Certs) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCerts
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Certs: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Certs: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SSH", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCerts
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCerts
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SSH = append(m.SSH[:0], dAtA[iNdEx:postIndex]...)
			if m.SSH == nil {
				m.SSH = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TLS", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCerts
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCerts
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TLS = append(m.TLS[:0], dAtA[iNdEx:postIndex]...)
			if m.TLS == nil {
				m.TLS = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TLSCACerts", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCerts
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCerts
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TLSCACerts = append(m.TLSCACerts, make([]byte, postIndex-iNdEx))
			copy(m.TLSCACerts[len(m.TLSCACerts)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SSHCACerts", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCerts
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCerts
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SSHCACerts = append(m.SSHCACerts, make([]byte, postIndex-iNdEx))
			copy(m.SSHCACerts[len(m.SSHCACerts)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCerts(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCerts
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipCerts(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCerts
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCerts
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthCerts
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupCerts
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthCerts
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthCerts        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCerts          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupCerts = fmt.Errorf("proto: unexpected end of group")
)
