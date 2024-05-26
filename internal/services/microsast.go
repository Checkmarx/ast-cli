package services

import (
	"bytes"
	"fmt"
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers/microsastengine/scans"
)

type MicroSastService struct {
	microSastWrapper *scans.MicroSastWrapper
}

func NewMicroSastService(port int) *MicroSastService {
	return &MicroSastService{
		microSastWrapper: scans.NewMicroSastWrapper(port),
	}
}

func (s *MicroSastService) Scan(filePath string) (*scans.ScanResult, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Print(err)
	}
	b = replaceEnDashWithHyphen(b)
	b = replaceCurlyApostrophe(b)
	return s.microSastWrapper.Scan(filePath, b)
}

func (s *MicroSastService) CheckHealth() error {
	return s.microSastWrapper.CheckHealth()
}

func replaceEnDashWithHyphen(data []byte) []byte {
	enDashBytes := []byte{0xE2, 0x80, 0x93}
	hyphenByte := byte('-') // ASCII hyphen

	return bytes.Replace(data, enDashBytes, []byte{hyphenByte}, -1)
}

func replaceCurlyApostrophe(data []byte) []byte {
	curlyApostropheBytes := []byte{0xE2, 0x80, 0x99}
	asciiApostropheByte := byte('\'')

	return bytes.Replace(data, curlyApostropheBytes, []byte{asciiApostropheByte}, -1)
}
