package ntlm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf16"

	// MD4 is required by NTLMv2, we're stuck with it here.
	"golang.org/x/crypto/md4" //nolint
)

// DialContext is the DialContext function that should be wrapped with a
// NTLM Authentication.
//
// Example for DialContext:
//
// dialContext := (&net.Dialer{KeepAlive: 30*time.Second, Timeout: 30*time.Second}).DialContext
type DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

const (
	expMsgBodyLen          = 40
	ntmlChallengeLen       = 2
	ntmlAuthHeaderLen      = 3
	ntmlMakeLen            = 8
	avIDMsvAvEOL      avID = iota
	avIDMsvAvTimestamp
)

// Version is a struct representing https://msdn.microsoft.com/en-us/library/cc236654.aspx
type Version struct {
	ProductMajorVersion uint8
	ProductMinorVersion uint8
	ProductBuild        uint16
	_                   [3]byte
	NTLMRevisionCurrent uint8
}

var signature = [8]byte{'N', 'T', 'L', 'M', 'S', 'S', 'P', 0}

type messageHeader struct {
	Signature   [8]byte
	MessageType uint32
}

type authenicateMessage struct {
	LmChallengeResponse       []byte
	NtChallengeResponse       []byte
	TargetName                string
	UserName                  string
	EncryptedRandomSessionKey []byte
	NegotiateFlags            negotiateFlags
	MIC                       []byte
}

type varField struct {
	Len          uint16
	MaxLen       uint16
	BufferOffset uint32
}

type authenticateMessageFields struct {
	messageHeader
	LmChallengeResponse varField
	NtChallengeResponse varField
	TargetName          varField
	UserName            varField
	Workstation         varField
	_                   [8]byte
	NegotiateFlags      negotiateFlags
}

type challengeMessageFields struct {
	messageHeader
	TargetName      varField
	NegotiateFlags  negotiateFlags
	ServerChallenge [8]byte
	_               [8]byte
	TargetInfo      varField
}

type negotiateMessageFields struct {
	messageHeader
	NegotiateFlags negotiateFlags

	Domain      varField
	Workstation varField

	Version
}

var defaultFlags = negotiateFlagNTLMSSPNEGOTIATETARGETINFO |
	negotiateFlagNTLMSSPNEGOTIATE56 |
	negotiateFlagNTLMSSPNEGOTIATE128 |
	negotiateFlagNTLMSSPNEGOTIATEUNICODE |
	negotiateFlagNTLMSSPNEGOTIATEEXTENDEDSESSIONSECURITY |
	negotiateFlagNTLMSSPNEGOTIATENTLM |
	negotiateFlagNTLMSSPNEGOTIATEALWAYSSIGN

type negotiateFlags uint32

const (
	/*A*/ negotiateFlagNTLMSSPNEGOTIATEUNICODE negotiateFlags = 1 << 0
	/*G*/ negotiateFlagNTLMSSPNEGOTIATELMKEY = 1 << 7
	/*H*/ negotiateFlagNTLMSSPNEGOTIATENTLM = 1 << 9
	/*K*/ negotiateFlagNTLMSSPNEGOTIATEOEMDOMAINSUPPLIED = 1 << 12
	/*L*/ negotiateFlagNTLMSSPNEGOTIATEOEMWORKSTATIONSUPPLIED = 1 << 13
	/*M*/ negotiateFlagNTLMSSPNEGOTIATEALWAYSSIGN = 1 << 15
	/*P*/ negotiateFlagNTLMSSPNEGOTIATEEXTENDEDSESSIONSECURITY = 1 << 19
	/*S*/ negotiateFlagNTLMSSPNEGOTIATETARGETINFO = 1 << 23
	/*T*/ negotiateFlagNTLMSSPNEGOTIATEVERSION = 1 << 25
	/*U*/ negotiateFlagNTLMSSPNEGOTIATE128 = 1 << 29
	/*V*/ negotiateFlagNTLMSSPNEGOTIATEKEYEXCH = 1 << 30
	/*W*/ negotiateFlagNTLMSSPNEGOTIATE56 = 1 << 31
)

type avID uint16

// NewNTLMProxyDialContext provides a DialContext function that includes transparent NTLM proxy authentication.
// Unlike WrapDialContext, it describes the proxy location with a full URL, whose scheme can be HTTP or HTTPS.
func NewNTLMProxyDialContext(dialer *net.Dialer, proxyURL *url.URL,
	proxyUsername, proxyPassword, proxyDomain string, tlsConfig *tls.Config) DialContext {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialProxy := func() (net.Conn, error) {
			if proxyURL.Scheme == "https" {
				return tls.DialWithDialer(dialer, "tcp", proxyURL.Host, tlsConfig)
			}
			return dialer.DialContext(ctx, network, proxyURL.Host)
		}
		return dialAndNegotiate(addr, proxyUsername, proxyPassword, proxyDomain, dialProxy)
	}
}

func dialAndNegotiate(addr, proxyUsername, proxyPassword, proxyDomain string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	conn, err := baseDial()
	if err != nil {
		log.Printf("Could not call dial context with proxy: %s", err)
		return conn, err
	}
	// NTLM Step 1: Send Negotiate Message
	negotiateMessage, err := newNegotiateMessage(proxyDomain, "")
	if err != nil {
		log.Printf("Could not negotiate domain '%s': %s", proxyDomain, err)
		return conn, err
	}
	header := make(http.Header)
	header.Set("Proxy-Authorization", fmt.Sprintf("NTLM %s", base64.StdEncoding.EncodeToString(negotiateMessage)))
	header.Set("Proxy-Connection", "Keep-Alive")
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}
	err = connect.Write(conn)
	if err != nil {
		log.Printf("Could not write negotiate message to proxy: %s", err)
		return conn, err
	}
	// NTLM Step 2: Receive Challenge Message
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		log.Printf("Could not read response from proxy: %s", err)
		return conn, err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read response body from proxy: %s", err)
		return conn, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusProxyAuthRequired {
		log.Printf("Expected %d as return status, got: %d", http.StatusProxyAuthRequired, resp.StatusCode)
		return conn, errors.New(http.StatusText(resp.StatusCode))
	}
	challenge := strings.Split(resp.Header.Get("Proxy-Authenticate"), " ")
	if len(challenge) < ntmlChallengeLen {
		log.Printf("The proxy did not return an NTLM challenge, got: '%s'", resp.Header.Get("Proxy-Authenticate"))
		return conn, errors.New("no NTLM challenge received")
	}
	challengeMessage, err := base64.StdEncoding.DecodeString(challenge[1])
	if err != nil {
		log.Printf("Could not base64 decode the NTLM challenge: %s", err)
		return conn, err
	}
	// NTLM Step 3: Send Authorization Message
	authenticateMessage, err := processChallenge(challengeMessage, proxyUsername, proxyPassword)
	if err != nil {
		log.Printf("Could not process the NTLM challenge: %s", err)
		return conn, err
	}
	header.Set("Proxy-Authorization", fmt.Sprintf("NTLM %s", base64.StdEncoding.EncodeToString(authenticateMessage)))
	connect = &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}
	if err = connect.Write(conn); err != nil {
		log.Printf("Could not write authorization to proxy: %s", err)
		return conn, err
	}
	resp, err = http.ReadResponse(br, connect)
	if err != nil {
		log.Printf("Could not read response from proxy: %s", err)
		_ = resp.Body.Close()
		return conn, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Expected %d as return status, got: %d", http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
		return conn, errors.New(http.StatusText(resp.StatusCode))
	}
	// Succussfully authorized with NTLM
	_ = resp.Body.Close()
	return conn, nil
}

// NewNegotiateMessage creates a new NEGOTIATE message with the
// flags that this package supports.
func newNegotiateMessage(domainName, workstationName string) ([]byte, error) {
	payloadOffset := expMsgBodyLen
	flags := defaultFlags

	if domainName != "" {
		flags |= negotiateFlagNTLMSSPNEGOTIATEOEMDOMAINSUPPLIED
	}

	if workstationName != "" {
		flags |= negotiateFlagNTLMSSPNEGOTIATEOEMWORKSTATIONSUPPLIED
	}

	msg := negotiateMessageFields{
		messageHeader:  newMessageHeader(1),
		NegotiateFlags: flags,
		Domain:         newVarField(&payloadOffset, len(domainName)),
		Workstation:    newVarField(&payloadOffset, len(workstationName)),
		Version:        DefaultVersion(),
	}

	b := bytes.Buffer{}
	err := binary.Write(&b, binary.LittleEndian, &msg)
	if err != nil {
		return nil, err
	}
	if b.Len() != expMsgBodyLen {
		return nil, errors.New("incorrect body length")
	}

	payload := strings.ToUpper(domainName + workstationName)
	if _, err := b.WriteString(payload); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (field negotiateFlags) Has(flags negotiateFlags) bool {
	return field&flags == flags
}

func (field *negotiateFlags) Unset(flags negotiateFlags) {
	*field ^= *field & flags
}

func (h messageHeader) IsValid() bool {
	return bytes.Equal(h.Signature[:], signature[:]) &&
		h.MessageType > 0 && h.MessageType < 4
}

func newMessageHeader(messageType uint32) messageHeader {
	return messageHeader{signature, messageType}
}

// DefaultVersion returns a Version with "sensible" defaults (Windows 7)
func DefaultVersion() Version {
	return Version{
		ProductMajorVersion: 6,
		ProductMinorVersion: 1,
		ProductBuild:        7601,
		NTLMRevisionCurrent: 15,
	}
}

func (f varField) ReadFrom(buffer []byte) ([]byte, error) {
	if len(buffer) < int(f.BufferOffset+uint32(f.Len)) {
		return nil, errors.New("error reading data, varField extends beyond buffer")
	}
	return buffer[f.BufferOffset : f.BufferOffset+uint32(f.Len)], nil
}

func (f varField) ReadStringFrom(buffer []byte, unicode bool) (string, error) {
	d, err := f.ReadFrom(buffer)
	if err != nil {
		return "", err
	}
	if unicode { // UTF-16LE encoding scheme
		return fromUnicode(d)
	}
	// OEM encoding, close enough to ASCII, since no code page is specified
	return string(d), err
}

func newVarField(ptr *int, fieldsize int) varField {
	f := varField{
		Len:          uint16(fieldsize),
		MaxLen:       uint16(fieldsize),
		BufferOffset: uint32(*ptr),
	}
	*ptr += fieldsize
	return f
}

func fromUnicode(d []byte) (string, error) {
	if len(d)%2 > 0 {
		return "", errors.New("unicode (UTF 16 LE) specified, but uneven data length")
	}
	s := make([]uint16, len(d)/ntmlChallengeLen)
	err := binary.Read(bytes.NewReader(d), binary.LittleEndian, &s)
	if err != nil {
		return "", err
	}
	return string(utf16.Decode(s)), nil
}

func toUnicode(s string) []byte {
	uints := utf16.Encode([]rune(s))
	b := bytes.Buffer{}
	_ = binary.Write(&b, binary.LittleEndian, &uints)
	return b.Bytes()
}

func (m *authenicateMessage) MarshalBinary() ([]byte, error) {
	if !m.NegotiateFlags.Has(negotiateFlagNTLMSSPNEGOTIATEUNICODE) {
		return nil, errors.New("only unicode is supported")
	}

	target, user := toUnicode(m.TargetName), toUnicode(m.UserName)
	workstation := toUnicode("go-ntlmssp")

	ptr := binary.Size(&authenticateMessageFields{})
	f := authenticateMessageFields{
		messageHeader:       newMessageHeader(ntmlAuthHeaderLen),
		NegotiateFlags:      m.NegotiateFlags,
		LmChallengeResponse: newVarField(&ptr, len(m.LmChallengeResponse)),
		NtChallengeResponse: newVarField(&ptr, len(m.NtChallengeResponse)),
		TargetName:          newVarField(&ptr, len(target)),
		UserName:            newVarField(&ptr, len(user)),
		Workstation:         newVarField(&ptr, len(workstation)),
	}

	f.NegotiateFlags.Unset(negotiateFlagNTLMSSPNEGOTIATEVERSION)

	b := bytes.Buffer{}
	err := binary.Write(&b, binary.LittleEndian, &f)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.LittleEndian, &m.LmChallengeResponse)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.LittleEndian, &m.NtChallengeResponse)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.LittleEndian, &target)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.LittleEndian, &user)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.LittleEndian, &workstation)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ProcessChallenge crafts an AUTHENTICATE message in response to the CHALLENGE message
// that was received from the server
func processChallenge(challengeMessageData []byte, user, password string) ([]byte, error) {
	if user == "" && password == "" {
		return nil, errors.New("anonymous authentication not supported")
	}

	var cm challengeMessage
	if err := cm.UnmarshalBinary(challengeMessageData); err != nil {
		return nil, err
	}

	if cm.NegotiateFlags.Has(negotiateFlagNTLMSSPNEGOTIATELMKEY) {
		return nil, errors.New("only NTLM v2 is supported, but server requested v1 (NTLMSSP_NEGOTIATE_LM_KEY)")
	}
	if cm.NegotiateFlags.Has(negotiateFlagNTLMSSPNEGOTIATEKEYEXCH) {
		return nil, errors.New("key exchange requested but not supported (NTLMSSP_NEGOTIATE_KEY_EXCH)")
	}

	am := authenicateMessage{
		UserName:       user,
		TargetName:     cm.TargetName,
		NegotiateFlags: cm.NegotiateFlags,
	}

	timestamp := cm.TargetInfo[avIDMsvAvTimestamp]
	if timestamp == nil { // no time sent, take current time
		ft := uint64(time.Now().UnixNano()) / 100
		ft += 116444736000000000 // add time between unix & windows offset
		timestamp = make([]byte, ntmlMakeLen)
		binary.LittleEndian.PutUint64(timestamp, ft)
	}
	clientChallenge := make([]byte, ntmlMakeLen)
	_, _ = rand.Reader.Read(clientChallenge)
	ntlmV2Hash := getNtlmV2Hash(password, user, cm.TargetName)
	am.NtChallengeResponse = computeNtlmV2Response(ntlmV2Hash,
		cm.ServerChallenge[:], clientChallenge, timestamp, cm.TargetInfoRaw)
	if cm.TargetInfoRaw == nil {
		am.LmChallengeResponse = computeLmV2Response(ntlmV2Hash,
			cm.ServerChallenge[:], clientChallenge)
	}
	return am.MarshalBinary()
}

func (m challengeMessageFields) IsValid() bool {
	return m.messageHeader.IsValid() && m.MessageType == 2
}

type challengeMessage struct {
	challengeMessageFields
	TargetName    string
	TargetInfo    map[avID][]byte
	TargetInfoRaw []byte
}

func (m *challengeMessage) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.LittleEndian, &m.challengeMessageFields)
	if err != nil {
		return err
	}
	if !m.challengeMessageFields.IsValid() {
		return fmt.Errorf("message is not a valid challenge message: %+v", m.challengeMessageFields.messageHeader)
	}

	if m.challengeMessageFields.TargetName.Len > 0 {
		m.TargetName, err = m.challengeMessageFields.TargetName.ReadStringFrom(data, m.NegotiateFlags.Has(negotiateFlagNTLMSSPNEGOTIATEUNICODE))
		if err != nil {
			return err
		}
	}

	if m.challengeMessageFields.TargetInfo.Len > 0 {
		d, err := m.challengeMessageFields.TargetInfo.ReadFrom(data)
		m.TargetInfoRaw = d
		if err != nil {
			return err
		}
		m.TargetInfo = make(map[avID][]byte)
		r := bytes.NewReader(d)
		for {
			var id avID
			var l uint16
			err = binary.Read(r, binary.LittleEndian, &id)
			if err != nil {
				return err
			}
			if id == avIDMsvAvEOL {
				break
			}

			err = binary.Read(r, binary.LittleEndian, &l)
			if err != nil {
				return err
			}
			value := make([]byte, l)
			n, err := r.Read(value)
			if err != nil {
				return err
			}
			if n != int(l) {
				return fmt.Errorf("expected to read %d bytes, got only %d", l, n)
			}
			m.TargetInfo[id] = value
		}
	}

	return nil
}

func getNtlmV2Hash(password, username, target string) []byte {
	return hmacMd5(getNtlmHash(password), toUnicode(strings.ToUpper(username)+target))
}

func getNtlmHash(password string) []byte {
	hash := md4.New()
	_, err := hash.Write(toUnicode(password))
	if err != nil {
		log.Println(err)
	}
	return hash.Sum(nil)
}

func computeNtlmV2Response(ntlmV2Hash, serverChallenge, clientChallenge, timestamp, targetInfo []byte) []byte {
	temp := []byte{1, 1, 0, 0, 0, 0, 0, 0}
	temp = append(temp, timestamp...)
	temp = append(temp, clientChallenge...)
	temp = append(temp, 0, 0, 0, 0)
	temp = append(temp, targetInfo...)
	temp = append(temp, 0, 0, 0, 0)
	NTProofStr := hmacMd5(ntlmV2Hash, serverChallenge, temp)
	return append(NTProofStr, temp...)
}

func computeLmV2Response(ntlmV2Hash, serverChallenge, clientChallenge []byte) []byte {
	return append(hmacMd5(ntlmV2Hash, serverChallenge, clientChallenge), clientChallenge...)
}

func hmacMd5(key []byte, data ...[]byte) []byte {
	mac := hmac.New(md5.New, key)
	for _, d := range data {
		_, err := mac.Write(d)
		if err != nil {
			log.Println(err)
		}
	}
	return mac.Sum(nil)
}
